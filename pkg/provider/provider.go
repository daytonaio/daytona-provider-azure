package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/daytonaio/daytona-provider-azure/internal"
	logwriters "github.com/daytonaio/daytona-provider-azure/internal/log"
	azureutil "github.com/daytonaio/daytona-provider-azure/pkg/provider/util"
	"github.com/daytonaio/daytona-provider-azure/pkg/types"
	"github.com/daytonaio/daytona/pkg/agent/ssh/config"
	"github.com/daytonaio/daytona/pkg/docker"
	"github.com/daytonaio/daytona/pkg/ssh"
	"github.com/daytonaio/daytona/pkg/tailscale"
	"github.com/daytonaio/daytona/pkg/workspace/project"
	"tailscale.com/tsnet"

	"github.com/daytonaio/daytona/pkg/logs"
	"github.com/daytonaio/daytona/pkg/provider"
	"github.com/daytonaio/daytona/pkg/provider/util"
	"github.com/daytonaio/daytona/pkg/workspace"
)

type AzureProvider struct {
	BasePath           *string
	DaytonaDownloadUrl *string
	DaytonaVersion     *string
	ServerUrl          *string
	NetworkKey         *string
	ApiUrl             *string
	ApiPort            *uint32
	ServerPort         *uint32
	LogsDir            *string
	tsnetConn          *tsnet.Server
}

func (a *AzureProvider) Initialize(req provider.InitializeProviderRequest) (*util.Empty, error) {
	a.BasePath = &req.BasePath
	a.DaytonaDownloadUrl = &req.DaytonaDownloadUrl
	a.DaytonaVersion = &req.DaytonaVersion
	a.ServerUrl = &req.ServerUrl
	a.NetworkKey = &req.NetworkKey
	a.ApiUrl = &req.ApiUrl
	a.ApiPort = &req.ApiPort
	a.ServerPort = &req.ServerPort
	a.LogsDir = &req.LogsDir

	return new(util.Empty), nil
}

func (a *AzureProvider) GetInfo() (provider.ProviderInfo, error) {
	label := "Azure"

	return provider.ProviderInfo{
		Label:   &label,
		Name:    "azure-provider",
		Version: internal.Version,
	}, nil
}

func (a *AzureProvider) GetTargetManifest() (*provider.ProviderTargetManifest, error) {
	return types.GetTargetManifest(), nil
}

func (a *AzureProvider) GetPresetTargets() (*[]provider.ProviderTarget, error) {
	return new([]provider.ProviderTarget), nil
}

func (a *AzureProvider) CreateWorkspace(workspaceReq *provider.WorkspaceRequest) (*util.Empty, error) {
	if a.DaytonaDownloadUrl == nil {
		return nil, errors.New("DaytonaDownloadUrl not set. Did you forget to call Initialize")
	}
	logWriter, cleanupFunc := a.getWorkspaceLogWriter(workspaceReq.Workspace.Id)
	defer cleanupFunc()

	targetOptions, err := types.ParseTargetOptions(workspaceReq.TargetOptions)
	if err != nil {
		logWriter.Write([]byte("Failed to parse target options: " + err.Error() + "\n"))
		return nil, err
	}

	initScript := fmt.Sprintf(`curl -sfL -H "Authorization: Bearer %s" %s | bash`, workspaceReq.Workspace.ApiKey, *a.DaytonaDownloadUrl)
	err = azureutil.CreateWorkspace(workspaceReq.Workspace, targetOptions, initScript, logWriter)
	if err != nil {
		logWriter.Write([]byte("Failed to create workspace: " + err.Error() + "\n"))
		return nil, err
	}

	agentSpinner := logwriters.ShowSpinner(logWriter, "Waiting for the agent to start", "Agent started")
	err = a.waitForDial(workspaceReq.Workspace.Id, 10*time.Minute)
	close(agentSpinner)
	if err != nil {
		logWriter.Write([]byte("Failed to dial: " + err.Error() + "\n"))
		return nil, err
	}

	client, err := a.getDockerClient(workspaceReq.Workspace.Id)
	if err != nil {
		logWriter.Write([]byte("Failed to get client: " + err.Error() + "\n"))
		return nil, err
	}

	workspaceDir := getWorkspaceDir(workspaceReq.Workspace.Id)
	sshClient, err := tailscale.NewSshClient(a.tsnetConn, &ssh.SessionConfig{
		Hostname: workspaceReq.Workspace.Id,
		Port:     config.SSH_PORT,
	})
	if err != nil {
		logWriter.Write([]byte("Failed to create ssh client: " + err.Error() + "\n"))
		return new(util.Empty), err
	}
	defer sshClient.Close()

	return new(util.Empty), client.CreateWorkspace(workspaceReq.Workspace, workspaceDir, logWriter, sshClient)
}

func (a *AzureProvider) StartWorkspace(workspaceReq *provider.WorkspaceRequest) (*util.Empty, error) {
	logWriter, cleanupFunc := a.getWorkspaceLogWriter(workspaceReq.Workspace.Id)
	defer cleanupFunc()

	targetOptions, err := types.ParseTargetOptions(workspaceReq.TargetOptions)
	if err != nil {
		logWriter.Write([]byte("Failed to parse target options: " + err.Error() + "\n"))
		return nil, err
	}

	err = a.waitForDial(workspaceReq.Workspace.Id, 10*time.Minute)
	if err != nil {
		logWriter.Write([]byte("Failed to dial: " + err.Error() + "\n"))
		return nil, err
	}

	return new(util.Empty), azureutil.StartWorkspace(workspaceReq.Workspace, targetOptions)
}

func (a *AzureProvider) StopWorkspace(workspaceReq *provider.WorkspaceRequest) (*util.Empty, error) {
	logWriter, cleanupFunc := a.getWorkspaceLogWriter(workspaceReq.Workspace.Id)
	defer cleanupFunc()

	targetOptions, err := types.ParseTargetOptions(workspaceReq.TargetOptions)
	if err != nil {
		logWriter.Write([]byte("Failed to parse target options: " + err.Error() + "\n"))
		return nil, err
	}

	return new(util.Empty), azureutil.StopWorkspace(workspaceReq.Workspace, targetOptions)
}

func (a *AzureProvider) DestroyWorkspace(workspaceReq *provider.WorkspaceRequest) (*util.Empty, error) {
	logWriter, cleanupFunc := a.getWorkspaceLogWriter(workspaceReq.Workspace.Id)
	defer cleanupFunc()

	targetOptions, err := types.ParseTargetOptions(workspaceReq.TargetOptions)
	if err != nil {
		logWriter.Write([]byte("Failed to parse target options: " + err.Error() + "\n"))
		return nil, err
	}

	return new(util.Empty), azureutil.DeleteWorkspace(workspaceReq.Workspace, targetOptions)
}

func (a *AzureProvider) GetWorkspaceInfo(workspaceReq *provider.WorkspaceRequest) (*workspace.WorkspaceInfo, error) {
	workspaceInfo, err := a.getWorkspaceInfo(workspaceReq)
	if err != nil {
		return nil, err
	}

	var projectInfos []*project.ProjectInfo
	for _, project := range workspaceReq.Workspace.Projects {
		projectInfo, err := a.GetProjectInfo(&provider.ProjectRequest{
			TargetOptions: workspaceReq.TargetOptions,
			Project:       project,
		})
		if err != nil {
			return nil, err
		}
		projectInfos = append(projectInfos, projectInfo)
	}
	workspaceInfo.Projects = projectInfos

	return workspaceInfo, nil
}

func (a *AzureProvider) CheckRequirements() (*[]provider.RequirementStatus, error) {
	results := []provider.RequirementStatus{}
	return &results, nil 
}

func (a *AzureProvider) CreateProject(projectReq *provider.ProjectRequest) (*util.Empty, error) {
	logWriter, cleanupFunc := a.getProjectLogWriter(projectReq.Project.WorkspaceId, projectReq.Project.Name)
	defer cleanupFunc()
	logWriter.Write([]byte("\033[?25h\n"))

	dockerClient, err := a.getDockerClient(projectReq.Project.WorkspaceId)
	if err != nil {
		logWriter.Write([]byte("Failed to get docker client: " + err.Error() + "\n"))
		return nil, err
	}

	sshClient, err := tailscale.NewSshClient(a.tsnetConn, &ssh.SessionConfig{
		Hostname: projectReq.Project.WorkspaceId,
		Port:     config.SSH_PORT,
	})
	if err != nil {
		logWriter.Write([]byte("Failed to create ssh client: " + err.Error() + "\n"))
		return new(util.Empty), err
	}
	defer sshClient.Close()

	return new(util.Empty), dockerClient.CreateProject(&docker.CreateProjectOptions{
		Project:    projectReq.Project,
		ProjectDir: getProjectDir(projectReq),
		Cr:         projectReq.ContainerRegistry,
		LogWriter:  logWriter,
		Gpc:        projectReq.GitProviderConfig,
		SshClient:  sshClient,
	})
}

func (a *AzureProvider) StartProject(projectReq *provider.ProjectRequest) (*util.Empty, error) {
	if a.DaytonaDownloadUrl == nil {
		return nil, errors.New("DaytonaDownloadUrl not set. Did you forget to call Initialize")
	}
	logWriter, cleanupFunc := a.getProjectLogWriter(projectReq.Project.WorkspaceId, projectReq.Project.Name)
	defer cleanupFunc()

	dockerClient, err := a.getDockerClient(projectReq.Project.WorkspaceId)
	if err != nil {
		logWriter.Write([]byte("Failed to get docker client: " + err.Error() + "\n"))
		return nil, err
	}

	sshClient, err := tailscale.NewSshClient(a.tsnetConn, &ssh.SessionConfig{
		Hostname: projectReq.Project.WorkspaceId,
		Port:     config.SSH_PORT,
	})
	if err != nil {
		logWriter.Write([]byte("Failed to create ssh client: " + err.Error() + "\n"))
		return new(util.Empty), err
	}
	defer sshClient.Close()

	return new(util.Empty), dockerClient.StartProject(&docker.CreateProjectOptions{
		Project:    projectReq.Project,
		ProjectDir: getProjectDir(projectReq),
		Cr:         projectReq.ContainerRegistry,
		LogWriter:  logWriter,
		Gpc:        projectReq.GitProviderConfig,
		SshClient:  sshClient,
	}, *a.DaytonaDownloadUrl)
}

func (a *AzureProvider) StopProject(projectReq *provider.ProjectRequest) (*util.Empty, error) {
	logWriter, cleanupFunc := a.getProjectLogWriter(projectReq.Project.WorkspaceId, projectReq.Project.Name)
	defer cleanupFunc()

	dockerClient, err := a.getDockerClient(projectReq.Project.WorkspaceId)
	if err != nil {
		logWriter.Write([]byte("Failed to get docker client: " + err.Error() + "\n"))
		return nil, err
	}

	return new(util.Empty), dockerClient.StopProject(projectReq.Project, logWriter)
}

func (a *AzureProvider) DestroyProject(projectReq *provider.ProjectRequest) (*util.Empty, error) {
	logWriter, cleanupFunc := a.getProjectLogWriter(projectReq.Project.WorkspaceId, projectReq.Project.Name)
	defer cleanupFunc()

	dockerClient, err := a.getDockerClient(projectReq.Project.WorkspaceId)
	if err != nil {
		logWriter.Write([]byte("Failed to get docker client: " + err.Error() + "\n"))
		return nil, err
	}

	sshClient, err := tailscale.NewSshClient(a.tsnetConn, &ssh.SessionConfig{
		Hostname: projectReq.Project.WorkspaceId,
		Port:     config.SSH_PORT,
	})
	if err != nil {
		logWriter.Write([]byte("Failed to create ssh client: " + err.Error() + "\n"))
		return new(util.Empty), err
	}
	defer sshClient.Close()

	return new(util.Empty), dockerClient.DestroyProject(projectReq.Project, getProjectDir(projectReq), sshClient)
}

func (a *AzureProvider) GetProjectInfo(projectReq *provider.ProjectRequest) (*project.ProjectInfo, error) {
	logWriter, cleanupFunc := a.getProjectLogWriter(projectReq.Project.WorkspaceId, projectReq.Project.Name)
	defer cleanupFunc()

	dockerClient, err := a.getDockerClient(projectReq.Project.WorkspaceId)
	if err != nil {
		logWriter.Write([]byte("Failed to get docker client: " + err.Error() + "\n"))
		return nil, err
	}

	return dockerClient.GetProjectInfo(projectReq.Project)
}

func (a *AzureProvider) getWorkspaceInfo(workspaceReq *provider.WorkspaceRequest) (*workspace.WorkspaceInfo, error) {
	logWriter, cleanupFunc := a.getWorkspaceLogWriter(workspaceReq.Workspace.Id)
	defer cleanupFunc()

	targetOptions, err := types.ParseTargetOptions(workspaceReq.TargetOptions)
	if err != nil {
		logWriter.Write([]byte("Failed to parse target options: " + err.Error() + "\n"))
		return nil, err
	}

	vm, err := azureutil.GetVirtualMachine(workspaceReq.Workspace, targetOptions)
	if err != nil {
		return nil, err
	}

	metadata := types.ToWorkspaceMetadata(vm)
	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	return &workspace.WorkspaceInfo{
		Name:             workspaceReq.Workspace.Name,
		ProviderMetadata: string(jsonMetadata),
	}, nil
}

func (a *AzureProvider) getWorkspaceLogWriter(workspaceId string) (io.Writer, func()) {
	logWriter := io.MultiWriter(&logwriters.InfoLogWriter{})
	cleanupFunc := func() {}

	if a.LogsDir != nil {
		loggerFactory := logs.NewLoggerFactory(a.LogsDir, nil)
		wsLogWriter := loggerFactory.CreateWorkspaceLogger(workspaceId, logs.LogSourceProvider)
		logWriter = io.MultiWriter(&logwriters.InfoLogWriter{}, wsLogWriter)
		cleanupFunc = func() { wsLogWriter.Close() }
	}

	return logWriter, cleanupFunc
}

func (a *AzureProvider) getProjectLogWriter(workspaceId string, projectName string) (io.Writer, func()) {
	logWriter := io.MultiWriter(&logwriters.InfoLogWriter{})
	cleanupFunc := func() {}

	if a.LogsDir != nil {
		loggerFactory := logs.NewLoggerFactory(a.LogsDir, nil)
		projectLogWriter := loggerFactory.CreateProjectLogger(workspaceId, projectName, logs.LogSourceProvider)
		logWriter = io.MultiWriter(&logwriters.InfoLogWriter{}, projectLogWriter)
		cleanupFunc = func() { projectLogWriter.Close() }
	}

	return logWriter, cleanupFunc
}

func getWorkspaceDir(workspaceId string) string {
	return fmt.Sprintf("/home/daytona/%s", workspaceId)
}

func getProjectDir(projectReq *provider.ProjectRequest) string {
	return path.Join(
		getWorkspaceDir(projectReq.Project.WorkspaceId),
		fmt.Sprintf("%s-%s", projectReq.Project.WorkspaceId, projectReq.Project.Name),
	)
}

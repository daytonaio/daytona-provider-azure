package provider

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	azureutil "github.com/daytonaio/daytona-provider-azure/pkg/provider/util"
	"github.com/daytonaio/daytona-provider-azure/pkg/types"
	"github.com/daytonaio/daytona/pkg/provider"
	"github.com/daytonaio/daytona/pkg/workspace"
)

var (
	tenantId       = os.Getenv("AZURE_TENANT_ID")
	clientId       = os.Getenv("AZURE_CLIENT_ID")
	clientSecret   = os.Getenv("AZURE_CLIENT_SECRET")
	subscriptionId = os.Getenv("AZURE_SUBSCRIPTION_ID")

	azureProvider = &AzureProvider{}
	targetOptions = &types.TargetOptions{
		Region:         "centralus",
		TenantId:       tenantId,
		ClientId:       clientId,
		ClientSecret:   clientSecret,
		SubscriptionId: subscriptionId,
		ImageURN:       "Canonical:ubuntu-24_04-lts:server:latest",
		VMSize:         "Standard_B2s",
		DiskType:       "StandardSSD_LRS",
		DiskSize:       30,
	}

	workspaceReq *provider.WorkspaceRequest
)

func TestCreateWorkspace(t *testing.T) {
	_, _ = azureProvider.CreateWorkspace(workspaceReq)

	_, err := azureutil.GetVirtualMachine(workspaceReq.Workspace, targetOptions)
	if err != nil {
		t.Fatalf("Error getting machine: %s", err)
	}
}

func TestWorkspaceInfo(t *testing.T) {
	workspaceInfo, err := azureProvider.GetWorkspaceInfo(workspaceReq)
	if err != nil {
		t.Fatalf("Error getting workspace info: %s", err)
	}

	var workspaceMetadata types.WorkspaceMetadata
	err = json.Unmarshal([]byte(workspaceInfo.ProviderMetadata), &workspaceMetadata)
	if err != nil {
		t.Fatalf("Error unmarshalling workspace metadata: %s", err)
	}

	vm, err := azureutil.GetVirtualMachine(workspaceReq.Workspace, targetOptions)
	if err != nil {
		t.Fatalf("Error getting machine: %s", err)
	}

	expectedMetadata := types.ToWorkspaceMetadata(vm)

	if expectedMetadata.VirtualMachineId != workspaceMetadata.VirtualMachineId {
		t.Fatalf("Expected vm id %s, got %s",
			expectedMetadata.VirtualMachineId,
			workspaceMetadata.VirtualMachineId,
		)
	}

	if expectedMetadata.VirtualMachineName != workspaceMetadata.VirtualMachineName {
		t.Fatalf("Expected vm name %s, got %s",
			expectedMetadata.VirtualMachineName,
			workspaceMetadata.VirtualMachineName,
		)
	}

	if expectedMetadata.VirtualMachineSizeType != workspaceMetadata.VirtualMachineSizeType {
		t.Fatalf("Expected vm size type %s, got %s",
			expectedMetadata.VirtualMachineSizeType,
			workspaceMetadata.VirtualMachineSizeType,
		)
	}

	if expectedMetadata.Location != workspaceMetadata.Location {
		t.Fatalf("Expected vm location %s, got %s",
			expectedMetadata.Location,
			workspaceMetadata.Location,
		)
	}

	if expectedMetadata.Created != workspaceMetadata.Created {
		t.Fatalf("Expected vm created at %s, got %s",
			expectedMetadata.Created,
			workspaceMetadata.Created,
		)
	}
}

func TestDestroyWorkspace(t *testing.T) {
	_, err := azureProvider.DestroyWorkspace(workspaceReq)
	if err != nil {
		t.Fatalf("Error destroying workspace: %s", err)
	}
	time.Sleep(3 * time.Second)

	_, err = azureutil.GetVirtualMachine(workspaceReq.Workspace, targetOptions)
	if err == nil {
		t.Fatalf("Error destroyed workspace still exists")
	}
}

func init() {
	_, err := azureProvider.Initialize(provider.InitializeProviderRequest{
		BasePath:           "/tmp/workspaces",
		DaytonaDownloadUrl: "https://download.daytona.io/daytona/install.sh",
		DaytonaVersion:     "latest",
		ServerUrl:          "",
		ApiUrl:             "",
		LogsDir:            "/tmp/logs",
	})
	if err != nil {
		panic(err)
	}

	opts, err := json.Marshal(targetOptions)
	if err != nil {
		panic(err)
	}

	workspaceReq = &provider.WorkspaceRequest{
		TargetOptions: string(opts),
		Workspace: &workspace.Workspace{
			Id:   "123",
			Name: "workspace",
		},
	}
}

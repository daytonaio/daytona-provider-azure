package util

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/daytonaio/daytona-provider-azure/pkg/types"
	"github.com/daytonaio/daytona/pkg/workspace"
)

func CreateWorkspace(workspace *workspace.Workspace, opts *types.TargetOptions, initScript string) error {
	cred, err := getClientCredentials(opts)
	if err != nil {
		return err
	}

	resourceGroupName, err := initResourceGroup(opts)
	if err != nil {
		return err
	}

	envVars := workspace.EnvVars
	envVars["DAYTONA_AGENT_LOG_FILE_PATH"] = "/home/daytona/.daytona-agent.log"

	customData := `#!/bin/bash
useradd -m -d /home/daytona daytona

curl -fsSL https://get.docker.com | bash

# Modify Docker daemon configuration
cat > /etc/docker/daemon.json <<EOF
{
  "hosts": ["unix:///var/run/docker.sock", "tcp://0.0.0.0:2375"]
}
EOF

# Create a systemd drop-in file to modify the Docker service
mkdir -p /etc/systemd/system/docker.service.d
cat > /etc/systemd/system/docker.service.d/override.conf <<EOF
[Service]
ExecStart=
ExecStart=/usr/bin/dockerd
EOF

systemctl daemon-reload
systemctl restart docker
systemctl start docker

usermod -aG docker daytona

if grep -q sudo /etc/group; then
	usermod -aG sudo,docker daytona
elif grep -q wheel /etc/group; then
	usermod -aG wheel,docker daytona
fi

echo "daytona ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/91-daytona

`

	for k, v := range envVars {
		customData += fmt.Sprintf("export %s=%s\n", k, v)
	}
	customData += initScript
	customData += `
echo '[Unit]
Description=Daytona Agent Service
After=network.target

[Service]
User=daytona
ExecStart=/usr/local/bin/daytona agent --host
Restart=always
`

	for k, v := range envVars {
		customData += fmt.Sprintf("Environment='%s=%s'\n", k, v)
	}

	customData += `
[Install]
WantedBy=multi-user.target' > /etc/systemd/system/daytona-agent.service
systemctl daemon-reload
systemctl enable daytona-agent.service
systemctl start daytona-agent.service
`

	customDataEncoded := base64.StdEncoding.EncodeToString([]byte(customData))
	return createVirtualMachine(workspace.Id, resourceGroupName, customDataEncoded, opts, cred)
}

func StartWorkspace(workspace *workspace.Workspace, opts *types.TargetOptions) error {
	cred, err := getClientCredentials(opts)
	if err != nil {
		return err
	}

	computeClient, err := armcompute.NewVirtualMachinesClient(opts.SubscriptionId, cred, nil)
	if err != nil {
		return err
	}

	vmName := getResourceName(workspace.Id)
	resourceGroup := getResourceGroupName(opts)

	pollerResp, err := computeClient.BeginStart(context.Background(), resourceGroup, vmName, nil)
	if err != nil {
		return err
	}

	_, err = pollerResp.PollUntilDone(context.Background(), nil)
	if err != nil {
		return err
	}

	return nil
}

func StopWorkspace(workspace *workspace.Workspace, opts *types.TargetOptions) error {
	cred, err := getClientCredentials(opts)
	if err != nil {
		return err
	}

	computeClient, err := armcompute.NewVirtualMachinesClient(opts.SubscriptionId, cred, nil)
	if err != nil {
		return err
	}

	vmName := getResourceName(workspace.Id)
	resourceGroupName := getResourceGroupName(opts)

	pollerResp, err := computeClient.BeginDeallocate(context.Background(), resourceGroupName, vmName, nil)
	if err != nil {
		return err
	}

	_, err = pollerResp.PollUntilDone(context.Background(), nil)
	if err != nil {
		return err
	}

	return nil
}

func DeleteWorkspace(workspace *workspace.Workspace, opts *types.TargetOptions) error {
	cred, err := getClientCredentials(opts)
	if err != nil {
		return err
	}

	err = deleteVirtualMachine(workspace.Id, opts, cred)
	if err != nil {
		return fmt.Errorf("cannot delete virtual machine: %+v", err)
	}

	err = deleteDisk(workspace.Id, opts, cred)
	if err != nil {
		return fmt.Errorf("cannot delete instance disk: %+v", err)
	}

	err = deleteNetworkInterface(workspace.Id, opts, cred)
	if err != nil {
		return fmt.Errorf("cannot delete network interface: %+v", err)
	}

	vNetName := getResourceName(fmt.Sprintf("vnet-%s", workspace.Id))
	subnetName := getResourceName(fmt.Sprintf("subnet-%s", workspace.Id))

	err = deleteSubnet(vNetName, subnetName, opts, cred)
	if err != nil {
		return fmt.Errorf("cannot delete subnet: %+v", err)
	}

	err = deleteVirtualNetwork(vNetName, opts, cred)
	if err != nil {
		return fmt.Errorf("cannot delete virtual network: %+v", err)
	}

	return nil
}

// getResourceName generates a machine name for the provided workspace.
func getResourceName(identifier string) string {
	return fmt.Sprintf("daytona-%s", identifier)
}
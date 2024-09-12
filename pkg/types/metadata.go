package types

import (
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
)

type WorkspaceMetadata struct {
	VirtualMachineId       string
	VirtualMachineName     string
	VirtualMachineSizeType string
	Location               string
	Created                string
}

// ToWorkspaceMetadata converts and maps values from an armcompute.VirtualMachine to a WorkspaceMetadata.
func ToWorkspaceMetadata(vm *armcompute.VirtualMachine) WorkspaceMetadata {
	metadata := WorkspaceMetadata{}

	if vm.ID != nil {
		metadata.VirtualMachineId = *vm.ID
	}

	if vm.Name != nil {
		metadata.VirtualMachineName = *vm.Name
	}

	if vm.Properties != nil && vm.Properties.HardwareProfile != nil && vm.Properties.HardwareProfile.VMSize != nil {
		metadata.VirtualMachineSizeType = string(*vm.Properties.HardwareProfile.VMSize)
	}

	if vm.Location != nil {
		metadata.Location = *vm.Location
	}

	if vm.Properties != nil && vm.Properties.TimeCreated != nil {
		metadata.Created = vm.Properties.TimeCreated.String()
	}

	return metadata
}

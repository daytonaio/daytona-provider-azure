package util

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"github.com/daytonaio/daytona-provider-azure/pkg/types"
)

// deleteVirtualMachine deletes a virtual machine instance in the specified resource group and workspace.
func deleteVirtualMachine(targetId string, opts *types.TargetOptions, cred azcore.TokenCredential) error {
	resourceGroupName := getResourceGroupName(opts)

	computeClient, err := armcompute.NewVirtualMachinesClient(opts.SubscriptionId, cred, nil)
	if err != nil {
		return err
	}

	vmName := getResourceName(targetId)

	pollerResp, err := computeClient.BeginDelete(context.Background(), resourceGroupName, vmName, nil)
	if err != nil {
		return err
	}

	_, err = pollerResp.PollUntilDone(context.Background(), nil)
	return err
}

// deleteDisk deletes a disk associated with virtual machine instance in a workspace.
func deleteDisk(targetId string, opts *types.TargetOptions, cred azcore.TokenCredential) error {
	resourceGroupName := getResourceGroupName(opts)

	diskClient, err := armcompute.NewDisksClient(opts.SubscriptionId, cred, nil)
	if err != nil {
		return err
	}

	vmDiskName := getResourceName(fmt.Sprintf("%s-disk", targetId))

	pollerResp, err := diskClient.BeginDelete(context.Background(), resourceGroupName, vmDiskName, nil)
	if err != nil {
		return err
	}

	_, err = pollerResp.PollUntilDone(context.Background(), nil)
	return err
}

// deleteVirtualNetwork deletes a virtual network in the specified resource group and workspace.
func deleteVirtualNetwork(vNetName string, opts *types.TargetOptions, cred azcore.TokenCredential) error {
	vnetClient, err := armnetwork.NewVirtualNetworksClient(opts.SubscriptionId, cred, nil)
	if err != nil {
		return err
	}

	resourceGroupName := getResourceGroupName(opts)
	pollerResp, err := vnetClient.BeginDelete(context.Background(), resourceGroupName, vNetName, nil)
	if err != nil {
		return err
	}

	_, err = pollerResp.PollUntilDone(context.Background(), nil)
	return err
}

// deleteSubnet deletes a subnet in a specified virtual network and resource group.
func deleteSubnet(vNetName, subnetName string, opts *types.TargetOptions, cred azcore.TokenCredential) error {
	subnetsClient, err := armnetwork.NewSubnetsClient(opts.SubscriptionId, cred, nil)
	if err != nil {
		return err
	}

	resourceGroupName := getResourceGroupName(opts)
	pollerResp, err := subnetsClient.BeginDelete(context.Background(), resourceGroupName, vNetName, subnetName, nil)
	if err != nil {
		return err
	}

	_, err = pollerResp.PollUntilDone(context.Background(), nil)
	return err
}

// deleteNetworkInterface deletes a network interface in a specified Azure subscription
// and resource group. It uses the given workspace ID, target options, and Azure Token
// Credential to authenticate the request. The function returns an error if the deletion
// process encounters any errors.
func deleteNetworkInterface(targetId string, opts *types.TargetOptions, cred azcore.TokenCredential) error {
	nicClient, err := armnetwork.NewInterfacesClient(opts.SubscriptionId, cred, nil)
	if err != nil {
		return err
	}

	resourceGroupName := getResourceGroupName(opts)
	ifaceName := getResourceName(fmt.Sprintf("iface-%s", targetId))

	pollerResp, err := nicClient.BeginDelete(context.Background(), resourceGroupName, ifaceName, nil)
	if err != nil {
		return err
	}

	_, err = pollerResp.PollUntilDone(context.Background(), nil)
	return err
}

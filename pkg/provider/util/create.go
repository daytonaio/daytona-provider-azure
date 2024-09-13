package util

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/daytonaio/daytona-provider-azure/pkg/types"
	"github.com/daytonaio/daytona/pkg/workspace"
	"github.com/sethvargo/go-password/password"
)

const (
	defaultResourceGroup = "daytona"
)

func initResourceGroup(opts *types.TargetOptions) (string, error) {
	cred, err := getClientCredentials(opts)
	if err != nil {
		return "", fmt.Errorf("failed to get client credentials: %w", err)
	}

	client, err := armresources.NewResourceGroupsClient(opts.SubscriptionId, cred, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create resource groups client: %w", err)
	}

	var resourceGroupName string

	if opts.ResourceGroup != "" {
		_, err = client.Get(context.Background(), opts.ResourceGroup, nil)
		if err != nil {
			return "", fmt.Errorf("failed to get resource group %s: %w", opts.ResourceGroup, err)
		}
		resourceGroupName = opts.ResourceGroup
	} else {
		_, err = client.Get(context.Background(), defaultResourceGroup, nil)
		if err != nil {
			_, err = client.CreateOrUpdate(
				context.Background(),
				defaultResourceGroup,
				armresources.ResourceGroup{Location: &opts.Region},
				nil,
			)
			if err != nil {
				return "", fmt.Errorf("failed to create resource group '%s': %w", defaultResourceGroup, err)
			}
		}

		resourceGroupName = defaultResourceGroup
	}

	return resourceGroupName, nil
}

// createVirtualMachine creates a new virtual machine instance in the specified Azure workspace.
func createVirtualMachine(workspaceId, resourceGroupName, customData string, opts *types.TargetOptions, cred azcore.TokenCredential) error {
	vNet, err := createVirtualNetwork(workspaceId, resourceGroupName, opts, cred)
	if err != nil {
		return fmt.Errorf("cannot create virtual network: %+v", err)
	}

	subnet, err := createSubnet(workspaceId, resourceGroupName, *vNet.Name, opts, cred)
	if err != nil {
		return fmt.Errorf("cannot create subnet: %+v", err)
	}

	iface, err := createNetworkInterface(workspaceId, resourceGroupName, *subnet.ID, opts, cred)
	if err != nil {
		return fmt.Errorf("cannot create network interface:%+v", err)
	}

	computeClient, err := armcompute.NewVirtualMachinesClient(opts.SubscriptionId, cred, nil)
	if err != nil {
		return err
	}

	vmName := getResourceName(workspaceId)
	vmDiskName := getResourceName(fmt.Sprintf("%s-disk", workspaceId))

	publisher, offer, sku, version, err := extractURNParts(opts.ImageURN)
	if err != nil {
		return err
	}

	pwd, err := password.Generate(12, 3, 3, false, true)
	if err != nil {
		return err
	}

	pollerResp, err := computeClient.BeginCreateOrUpdate(
		context.Background(),
		resourceGroupName,
		vmName,
		armcompute.VirtualMachine{
			Location: &opts.Region,
			Identity: &armcompute.VirtualMachineIdentity{
				Type: to.Ptr(armcompute.ResourceIdentityTypeNone),
			},
			Properties: &armcompute.VirtualMachineProperties{
				OSProfile: &armcompute.OSProfile{
					ComputerName:  to.Ptr(vmName),
					AdminUsername: to.Ptr("daytona"),
					AdminPassword: to.Ptr(pwd),
					CustomData:    to.Ptr(customData),
				},
				HardwareProfile: &armcompute.HardwareProfile{
					VMSize: to.Ptr(armcompute.VirtualMachineSizeTypes(opts.VMSize)),
				},
				StorageProfile: &armcompute.StorageProfile{
					ImageReference: &armcompute.ImageReference{
						Publisher: to.Ptr(publisher),
						Offer:     to.Ptr(offer),
						SKU:       to.Ptr(sku),
						Version:   to.Ptr(version),
					},
					OSDisk: &armcompute.OSDisk{
						Name:         to.Ptr(vmDiskName),
						CreateOption: to.Ptr(armcompute.DiskCreateOptionTypesFromImage),
						Caching:      to.Ptr(armcompute.CachingTypesReadWrite),
						ManagedDisk: &armcompute.ManagedDiskParameters{
							StorageAccountType: to.Ptr(
								armcompute.StorageAccountTypes(opts.DiskType),
							),
						},
						DiskSizeGB: to.Ptr[int32](int32(opts.DiskSize)),
					},
				},
				NetworkProfile: &armcompute.NetworkProfile{
					NetworkInterfaces: []*armcompute.NetworkInterfaceReference{
						{
							ID: iface.ID,
						},
					},
				},
			},
		}, nil)
	if err != nil {
		return err
	}

	_, err = pollerResp.PollUntilDone(context.Background(), nil)
	return err
}

// createVirtualNetwork creates a virtual network in the specified resource group.
// If the virtual network already exists, it returns the existing virtual network.
// Otherwise, it creates a new virtual network.
func createVirtualNetwork(workspaceId, resourceGroupName string, opts *types.TargetOptions, cred azcore.TokenCredential) (*armnetwork.VirtualNetwork, error) {
	vnetClient, err := armnetwork.NewVirtualNetworksClient(opts.SubscriptionId, cred, nil)
	if err != nil {
		return nil, err
	}

	vNetName := getResourceName(fmt.Sprintf("vnet-%s", workspaceId))
	vNetResp, err := vnetClient.Get(context.Background(), resourceGroupName, vNetName, nil)
	if err == nil {
		return &vNetResp.VirtualNetwork, nil
	}

	// virtual network does not exist create new one
	pollerResp, err := vnetClient.BeginCreateOrUpdate(
		context.Background(),
		resourceGroupName,
		vNetName,
		armnetwork.VirtualNetwork{
			Location: to.Ptr(opts.Region),
			Properties: &armnetwork.VirtualNetworkPropertiesFormat{
				AddressSpace: &armnetwork.AddressSpace{
					AddressPrefixes: []*string{
						to.Ptr("10.10.0.0/16"),
					},
				},
			},
		}, nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := pollerResp.PollUntilDone(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	return &resp.VirtualNetwork, nil
}

// createSubnet creates a subnet for a virtual network.
func createSubnet(workspaceId, resourceGroupName, vNetName string, opts *types.TargetOptions, cred azcore.TokenCredential) (*armnetwork.Subnet, error) {
	subnetsClient, err := armnetwork.NewSubnetsClient(opts.SubscriptionId, cred, nil)
	if err != nil {
		return nil, err
	}

	subnetName := getResourceName(fmt.Sprintf("subnet-%s", workspaceId))
	pollerResp, err := subnetsClient.BeginCreateOrUpdate(
		context.Background(),
		resourceGroupName,
		vNetName,
		subnetName,
		armnetwork.Subnet{
			Properties: &armnetwork.SubnetPropertiesFormat{
				AddressPrefix: to.Ptr("10.10.10.0/24"),
			},
		}, nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := pollerResp.PollUntilDone(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	return &resp.Subnet, nil
}

// createNetworkInterface creates a network interface.
func createNetworkInterface(workspaceId, resourceGroupName, subnetId string, opts *types.TargetOptions, cred azcore.TokenCredential) (*armnetwork.Interface, error) {
	nicClient, err := armnetwork.NewInterfacesClient(opts.SubscriptionId, cred, nil)
	if err != nil {
		return nil, err
	}

	ifaceName := getResourceName(fmt.Sprintf("iface-%s", workspaceId))
	pollerResponse, err := nicClient.BeginCreateOrUpdate(
		context.Background(),
		resourceGroupName,
		ifaceName,
		armnetwork.Interface{
			Location: to.Ptr(opts.Region),
			Properties: &armnetwork.InterfacePropertiesFormat{
				IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
					{
						Name: to.Ptr("ipConfig"),
						Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
							PrivateIPAllocationMethod: to.Ptr(armnetwork.IPAllocationMethodDynamic),
							Subnet: &armnetwork.Subnet{
								ID: to.Ptr(subnetId),
							},
						},
					},
				},
			},
		}, nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := pollerResponse.PollUntilDone(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	return &resp.Interface, err
}

func GetVirtualMachine(workspace *workspace.Workspace, opts *types.TargetOptions) (*armcompute.VirtualMachine, error) {
	cred, err := getClientCredentials(opts)
	if err != nil {
		return nil, err
	}

	computeClient, err := armcompute.NewVirtualMachinesClient(opts.SubscriptionId, cred, nil)
	if err != nil {
		return nil, err
	}

	resourceGroupName := getResourceGroupName(opts)
	vmName := getResourceName(workspace.Id)

	resp, err := computeClient.Get(context.Background(), resourceGroupName, vmName, nil)
	if err != nil {
		return nil, err
	}

	return &resp.VirtualMachine, nil
}

// getClientCredentials returns a new instance of azidentity.ClientSecretCredential
// using the provided options.
func getClientCredentials(opts *types.TargetOptions) (*azidentity.ClientSecretCredential, error) {
	return azidentity.NewClientSecretCredential(
		opts.TenantId,
		opts.ClientId,
		opts.ClientSecret,
		nil,
	)
}

// getResourceGroupName returns the resource group name from the given options.
func getResourceGroupName(opts *types.TargetOptions) string {
	if opts.ResourceGroup == "" {
		return defaultResourceGroup
	}
	return opts.ResourceGroup
}

// Function to extract publisher, offer, sku, and version from an Azure URN string
func extractURNParts(urn string) (publisher, offer, sku, version string, err error) {
	parts := strings.Split(urn, ":")

	// URN should have exactly 4 parts
	if len(parts) != 4 {
		return "", "", "", "", errors.New("invalid URN format")
	}

	publisher, offer, sku, version = parts[0], parts[1], parts[2], parts[3]
	return publisher, offer, sku, version, nil
}

package types

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/daytonaio/daytona/pkg/provider"
)

type TargetOptions struct {
	Region         string `json:"Region"`
	TenantId       string `json:"Tenant Id"`
	ClientId       string `json:"Client Id"`
	ClientSecret   string `json:"Client Secret"`
	SubscriptionId string `json:"Subscription Id"`
	ResourceGroup  string `json:"Resource Group"`
	ImageURN       string `json:"Image URN"`
	VMSize         string `json:"VM Size"`
	DiskType       string `json:"Disk Type"`
	DiskSize       int    `json:"Disk Size"`
}

func GetTargetManifest() *provider.ProviderTargetManifest {
	return &provider.ProviderTargetManifest{
		"Region": provider.ProviderTargetProperty{
			Type:         provider.ProviderTargetPropertyTypeString,
			DefaultValue: "centralus",
			Description: "The geographic area where Azure resources are hosted. Default is centralus.\n" +
				"List of available regions can be retrieved using the command:\n\"az account list-locations -o table\"",
			Suggestions: regions,
		},
		"Tenant Id": provider.ProviderTargetProperty{
			Type:        provider.ProviderTargetPropertyTypeString,
			InputMasked: true,
			Description: "Leave blank if you've set the AZURE_TENANT_ID environment variable, or enter your Tenant Id here.\n" +
				"To find the this, look for \"tenant\" in the output after generating client credentials.\nhttps://learn.microsoft.com/en-us/cli/azure/azure-cli-sp-tutorial-1?tabs=bash",
		},
		"Client Id": provider.ProviderTargetProperty{
			Type:        provider.ProviderTargetPropertyTypeString,
			InputMasked: true,
			Description: "Leave blank if you've set the AZURE_CLIENT_ID environment variable, or enter your Client Id here.\n" +
				"To find the this, look for \"appId\" in the output after generating client credentials.\nhttps://learn.microsoft.com/en-us/cli/azure/azure-cli-sp-tutorial-1?tabs=bash",
		},
		"Client Secret": provider.ProviderTargetProperty{
			Type:        provider.ProviderTargetPropertyTypeString,
			InputMasked: true,
			Description: "Leave blank if you've set the AZURE_CLIENT_SECRET environment variable, or enter your Client Secret here.\n" +
				"To find the this, look for \"password\" in the output after generating client credentials\nhttps://learn.microsoft.com/en-us/cli/azure/azure-cli-sp-tutorial-1?tabs=bash",
		},
		"Subscription Id": provider.ProviderTargetProperty{
			Type:        provider.ProviderTargetPropertyTypeString,
			InputMasked: true,
			Description: "Leave blank if you've set the AZURE_SUBSCRIPTION_ID environment variable, or enter your Subscription Id here.\n" +
				"How to find subscription id:\nhttps://learn.microsoft.com/en-us/azure/azure-portal/get-subscription-tenant-id#find-your-azure-subscription",
		},
		"Resource Group": provider.ProviderTargetProperty{
			Type:        provider.ProviderTargetPropertyTypeString,
			InputMasked: false,
			Description: "If not set, Daytona will create a \"daytona\" resource group.\n" +
				"How to create resource group:\nhttps://learn.microsoft.com/en-us/azure/azure-resource-manager/management/manage-resource-groups-portal",
		},
		"Image URN": provider.ProviderTargetProperty{
			Type:         provider.ProviderTargetPropertyTypeString,
			DefaultValue: "Canonical:ubuntu-24_04-lts:server:latest",
			Description: "The identifier of the Azure virtual machine image to launch an instance. Default is Canonical:ubuntu-24_04-lts:server:latest.\n" +
				"List of available images:\nhttps://learn.microsoft.com/en-us/azure/virtual-machines/linux/cli-ps-findimage",
			Suggestions: imagesUrns,
		},
		"VM Size": provider.ProviderTargetProperty{
			Type:         provider.ProviderTargetPropertyTypeString,
			DefaultValue: "Standard_B2s",
			Description: "The size of the Azure machine. Default is Standard_A2_v2.\n" +
				"List of available sizes:\nhttps://learn.microsoft.com/en-us/azure/virtual-machines/sizes/overview/" +
				"List of available sizes per location can be retrieved using the command:\naz vm list-sizes --location <your-region> --output table",
			Suggestions: vmSizes,
		},
		"Disk Type": provider.ProviderTargetProperty{
			Type:         provider.ProviderTargetPropertyTypeString,
			DefaultValue: "StandardSSD_LRS",
			Description: "The type of the azure managed disk. Default is StandardSSD_LRS.\n" +
				"List of available disk types:\nhttps://docs.microsoft.com/azure/virtual-machines/linux/disks-types" +
				"List of available disk types per location can be retrieved using the command:\naz vm list-skus --location <your-region> --output table",
			Suggestions: diskTypes,
		},
		"Disk Size": provider.ProviderTargetProperty{
			Type:         provider.ProviderTargetPropertyTypeInt,
			DefaultValue: "30",
			Description:  "The size of the instance volume, in GB. Default is 30 GB. It is recommended that the disk size should be more than 30 GB.",
		},
	}
}

// ParseTargetOptions parses the target options from the JSON string.
func ParseTargetOptions(optionsJson string) (*TargetOptions, error) {
	var targetOptions TargetOptions
	err := json.Unmarshal([]byte(optionsJson), &targetOptions)
	if err != nil {
		return nil, err
	}

	if targetOptions.TenantId == "" {
		tenantId, ok := os.LookupEnv("AZURE_TENANT_ID")
		if ok {
			targetOptions.TenantId = tenantId
		}
	}

	if targetOptions.ClientId == "" {
		clientId, ok := os.LookupEnv("AZURE_CLIENT_ID")
		if ok {
			targetOptions.ClientId = clientId
		}
	}

	if targetOptions.ClientSecret == "" {
		clientSecret, ok := os.LookupEnv("AZURE_CLIENT_SECRET")
		if ok {
			targetOptions.ClientSecret = clientSecret
		}
	}

	if targetOptions.SubscriptionId == "" {
		subscriptionId, ok := os.LookupEnv("AZURE_SUBSCRIPTION_ID")
		if ok {
			targetOptions.SubscriptionId = subscriptionId
		}
	}

	if targetOptions.TenantId == "" {
		return nil, fmt.Errorf("tenant id not set in env/target options")
	}
	if targetOptions.ClientId == "" {
		return nil, fmt.Errorf("client id not set in env/target options")
	}
	if targetOptions.ClientSecret == "" {
		return nil, fmt.Errorf("client secret not set in env/target options")
	}
	if targetOptions.SubscriptionId == "" {
		return nil, fmt.Errorf("subscription id not set in env/target options")
	}

	return &targetOptions, nil
}

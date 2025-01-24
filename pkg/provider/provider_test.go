package provider

import (
	"os"
	"testing"
	"time"

	azureutil "github.com/daytonaio/daytona-provider-azure/pkg/provider/util"
	"github.com/daytonaio/daytona-provider-azure/pkg/types"
	"github.com/daytonaio/daytona/pkg/models"
	"github.com/daytonaio/daytona/pkg/provider"
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

	targetReq *provider.TargetRequest
)

func TestCreateTarget(t *testing.T) {
	_, _ = azureProvider.CreateTarget(targetReq)

	_, err := azureutil.GetVirtualMachine(targetReq.Target, targetOptions)
	if err != nil {
		t.Fatalf("Error getting machine: %s", err)
	}
}

func TestDestroyTarget(t *testing.T) {
	_, err := azureProvider.DestroyTarget(targetReq)
	if err != nil {
		t.Fatalf("Error destroying target: %s", err)
	}
	time.Sleep(3 * time.Second)

	_, err = azureutil.GetVirtualMachine(targetReq.Target, targetOptions)
	if err == nil {
		t.Fatalf("Error destroyed target still exists")
	}
}

func init() {
	_, err := azureProvider.Initialize(provider.InitializeProviderRequest{
		BasePath:           "/tmp/targets",
		DaytonaDownloadUrl: "https://download.daytona.io/daytona/install.sh",
		DaytonaVersion:     "latest",
		ServerUrl:          "",
		ApiUrl:             "",
		TargetLogsDir:      "/tmp/logs",
	})
	if err != nil {
		panic(err)
	}

	targetReq = &provider.TargetRequest{
		Target: &models.Target{
			Id:   "123",
			Name: "target",
		},
	}
}

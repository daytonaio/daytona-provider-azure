package types

import (
	"reflect"
	"testing"
)

func TestGetTargetManifest(t *testing.T) {
	targetManifest := GetTargetManifest()
	if targetManifest == nil {
		t.Fatalf("Expected target manifest but got nil")
	}

	fields := [10]string{"Region", "Tenant Id", "Client Id", "Client Secret",
		"Subscription Id", "Image URN", "VM Size", "Disk Type", "Disk Size", "Resource Group",
	}
	for _, field := range fields {
		if _, ok := (*targetManifest)[field]; !ok {
			t.Errorf("Expected field %s in target manifest but it was not found", field)
		}
	}
}

func TestParseTargetOptions(t *testing.T) {
	tests := []struct {
		name        string
		optionsJson string
		envVars     map[string]string
		want        *TargetOptions
		wantErr     bool
	}{
		{
			name: "Valid JSON with all fields",
			optionsJson: `{
				"Tenant Id": "tenant-id-123",
				"Client Id": "client-id-123",
				"Client Secret": "client-secret-123",
				"Subscription Id": "subscription-id-123"
			}`,
			want: &TargetOptions{
				TenantId:       "tenant-id-123",
				ClientId:       "client-id-123",
				ClientSecret:   "client-secret-123",
				SubscriptionId: "subscription-id-123",
			},
			wantErr: false,
		},
		{
			name: "Valid JSON with missing fields, using env vars",
			optionsJson: `{
				"Tenant Id": "tenant-id-123",
				"Client Id": "client-id-123"
			}`,
			envVars: map[string]string{
				"AZURE_CLIENT_SECRET":   "client-secret-123",
				"AZURE_SUBSCRIPTION_ID": "subscription-id-123",
			},
			want: &TargetOptions{
				TenantId:       "tenant-id-123",
				ClientId:       "client-id-123",
				ClientSecret:   "client-secret-123",
				SubscriptionId: "subscription-id-123",
			},
			wantErr: false,
		},
		{
			name:        "Invalid JSON",
			optionsJson: `{"Tenant Id": "tenant-id-123", "Client Id": "client-id-123"`,
			wantErr:     true,
		},
		{
			name: "Missing all required fields in both JSON and env vars",
			optionsJson: `{
				"Region": "us-east-1"
			}`,
			wantErr: true,
		},
		{
			name:        "Empty JSON",
			optionsJson: `{}`,
			envVars: map[string]string{
				"AZURE_TENANT_ID":       "tenant-id-123",
				"AZURE_CLIENT_ID":       "client-id-123",
				"AZURE_CLIENT_SECRET":   "client-secret-123",
				"AZURE_SUBSCRIPTION_ID": "subscription-id-123",
			},
			want: &TargetOptions{
				TenantId:       "tenant-id-123",
				ClientId:       "client-id-123",
				ClientSecret:   "client-secret-123",
				SubscriptionId: "subscription-id-123",
			},
			wantErr: false,
		},
		{
			name: "Partial JSON with some valid env vars",
			optionsJson: `{
				"Tenant Id": "tenant-id-123"
			}`,
			envVars: map[string]string{
				"AZURE_CLIENT_ID":       "client-id-123",
				"AZURE_CLIENT_SECRET":   "client-secret-123",
				"AZURE_SUBSCRIPTION_ID": "subscription-id-123",
			},
			want: &TargetOptions{
				TenantId:       "tenant-id-123",
				ClientId:       "client-id-123",
				ClientSecret:   "client-secret-123",
				SubscriptionId: "subscription-id-123",
			},
			wantErr: false,
		},
		{
			name: "JSON with additional non-required fields",
			optionsJson: `{
				"Tenant Id": "tenant-id-123",
				"Client Id": "client-id-123",
				"Client Secret": "client-secret-123",
				"Subscription Id": "subscription-id-123",
				"ExtraField": "extra-value"
			}`,
			want: &TargetOptions{
				TenantId:       "tenant-id-123",
				ClientId:       "client-id-123",
				ClientSecret:   "client-secret-123",
				SubscriptionId: "subscription-id-123",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			got, err := ParseTargetOptions(tt.optionsJson)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTargetOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseTargetOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}

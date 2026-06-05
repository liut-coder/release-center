package appreleases

import "testing"

func TestIntegrationCredentialStatusesDetectConfiguredEnvVars(t *testing.T) {
	statuses := integrationCredentialStatuses(func(key string) string {
		switch key {
		case "GITHUB_WEBHOOK_SECRET":
			return "secret-value"
		case "CLOUDFLARE_API_TOKEN":
			return "cf-value"
		default:
			return ""
		}
	})
	webhook := findIntegrationCredentialStatus(statuses, "github_webhook_release_center")
	if webhook == nil || !webhook.Configured || len(webhook.ConfiguredEnvVars) != 1 || webhook.ConfiguredEnvVars[0] != "GITHUB_WEBHOOK_SECRET" {
		t.Fatalf("expected webhook secret configured by env name only, got %#v", webhook)
	}
	cloudflare := findIntegrationCredentialStatus(statuses, "cf_token_release_prod")
	if cloudflare == nil || !cloudflare.Configured || len(cloudflare.ConfiguredEnvVars) != 1 || cloudflare.ConfiguredEnvVars[0] != "CLOUDFLARE_API_TOKEN" {
		t.Fatalf("expected cloudflare token configured by fallback env name, got %#v", cloudflare)
	}
	for _, item := range statuses {
		for _, envVar := range item.ConfiguredEnvVars {
			if envVar == "secret-value" || envVar == "cf-value" {
				t.Fatalf("credential status leaked a secret value: %#v", item)
			}
		}
	}
}

func findIntegrationCredentialStatus(items []IntegrationCredentialStatus, ref string) *IntegrationCredentialStatus {
	for i := range items {
		if items[i].Ref == ref {
			return &items[i]
		}
	}
	return nil
}

package appreleases

import (
	"context"
	"os"
	"strings"
)

type IntegrationCredentialOverview struct {
	Credentials []IntegrationCredentialStatus `json:"credentials"`
	MessageZh   string                        `json:"message_zh,omitempty"`
}

type IntegrationCredentialStatus struct {
	Ref               string   `json:"ref"`
	Name              string   `json:"name"`
	Kind              string   `json:"kind"`
	Provider          string   `json:"provider"`
	Usage             string   `json:"usage"`
	EnvVars           []string `json:"env_vars"`
	Configured        bool     `json:"configured"`
	ConfiguredEnvVars []string `json:"configured_env_vars,omitempty"`
}

func (s *Service) IntegrationCredentials(ctx context.Context) (IntegrationCredentialOverview, error) {
	_ = ctx
	return IntegrationCredentialOverview{
		Credentials: integrationCredentialStatuses(os.Getenv),
		MessageZh:   "凭证引用已读取",
	}, nil
}

func integrationCredentialStatuses(lookup func(string) string) []IntegrationCredentialStatus {
	items := []IntegrationCredentialStatus{
		{
			Ref:      "github_token_release_center",
			Name:     "GitHub Release Center Token",
			Kind:     "access_token",
			Provider: "github",
			Usage:    "code_repository.credential_ref",
			EnvVars: []string{
				"RELEASE_CENTER_CREDENTIAL_GITHUB_TOKEN_RELEASE_CENTER",
				"GITHUB_TOKEN_RELEASE_CENTER",
				"GITHUB_TOKEN",
				"GH_TOKEN",
			},
		},
		{
			Ref:      "github_webhook_release_center",
			Name:     "GitHub Webhook Secret",
			Kind:     "webhook_secret",
			Provider: "github",
			Usage:    "code_repository.webhook_secret_ref",
			EnvVars: []string{
				"GITHUB_WEBHOOK_SECRET",
				"WEBHOOK_SECRET",
			},
		},
		{
			Ref:      "cf_token_release_prod",
			Name:     "Cloudflare Release Prod Token",
			Kind:     "api_token",
			Provider: "cloudflare",
			Usage:    "deployment_target.credential_ref",
			EnvVars: []string{
				"RELEASE_CENTER_CREDENTIAL_CF_TOKEN_RELEASE_PROD",
				"CLOUDFLARE_API_TOKEN_CF_TOKEN_RELEASE_PROD",
				"CF_TOKEN_RELEASE_PROD",
				"CLOUDFLARE_API_TOKEN",
			},
		},
		{
			Ref:      "cf_token_release_dev",
			Name:     "Cloudflare Release Dev Token",
			Kind:     "api_token",
			Provider: "cloudflare",
			Usage:    "deployment_target.credential_ref",
			EnvVars: []string{
				"RELEASE_CENTER_CREDENTIAL_CF_TOKEN_RELEASE_DEV",
				"CLOUDFLARE_API_TOKEN_CF_TOKEN_RELEASE_DEV",
				"CF_TOKEN_RELEASE_DEV",
				"CLOUDFLARE_API_TOKEN",
			},
		},
		{
			Ref:      "cloudflare_account_id",
			Name:     "Cloudflare Account ID",
			Kind:     "account_id",
			Provider: "cloudflare",
			Usage:    "worker environment",
			EnvVars: []string{
				"CLOUDFLARE_ACCOUNT_ID",
				"CF_ACCOUNT_ID",
			},
		},
		{
			Ref:      "release_center_ci_token",
			Name:     "Release Center CI / Worker Token",
			Kind:     "bearer_token",
			Provider: "release_center",
			Usage:    "worker and CI API",
			EnvVars: []string{
				"CI_TOKEN",
				"GAME_HELPER_CI_TOKEN",
			},
		},
	}
	for i := range items {
		items[i].ConfiguredEnvVars = configuredEnvVars(lookup, items[i].EnvVars)
		items[i].Configured = len(items[i].ConfiguredEnvVars) > 0
	}
	return items
}

func configuredEnvVars(lookup func(string) string, envVars []string) []string {
	configured := make([]string, 0, len(envVars))
	for _, envVar := range envVars {
		if strings.TrimSpace(lookup(envVar)) != "" {
			configured = append(configured, envVar)
		}
	}
	return configured
}

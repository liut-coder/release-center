package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/liut-coder/game-helper-server/internal/modules/appreleases"
)

func TestWorkerTaskCommandPrefersPreparedCommandArray(t *testing.T) {
	metadata := mustRawJSON(t, map[string]any{
		"command":          "buildctl all release-center",
		"prepared_command": []string{"wrangler", "pages", "deploy", "dist"},
	})
	spec, err := workerTaskCommand(appreleases.WorkerTaskAdmin{TaskType: "deploy", Action: "cloudflare_pages", Metadata: metadata})
	if err != nil {
		t.Fatalf("workerTaskCommand returned error: %v", err)
	}
	if spec.Source != "prepared_command" {
		t.Fatalf("source = %q, want prepared_command", spec.Source)
	}
	if spec.Shell != "" {
		t.Fatalf("shell command should be empty for array command, got %q", spec.Shell)
	}
	if got := spec.Args; len(got) != 4 || got[0] != "wrangler" || got[3] != "dist" {
		t.Fatalf("args = %#v", got)
	}
}

func TestWorkerTaskCommandReadsStringCommand(t *testing.T) {
	metadata := mustRawJSON(t, map[string]any{
		"command": "buildctl all release-center",
	})
	spec, err := workerTaskCommand(appreleases.WorkerTaskAdmin{TaskType: "build", Action: "all", Metadata: metadata})
	if err != nil {
		t.Fatalf("workerTaskCommand returned error: %v", err)
	}
	if spec.Source != "command" || spec.Shell != "buildctl all release-center" {
		t.Fatalf("spec = %#v", spec)
	}
}

func TestWorkerTaskCommandReadsActionCommands(t *testing.T) {
	metadata := mustRawJSON(t, map[string]any{
		"commands": map[string]any{
			"verify": []any{"go", "test", "./..."},
		},
	})
	spec, err := workerTaskCommand(appreleases.WorkerTaskAdmin{TaskType: "build", Action: "verify", Metadata: metadata})
	if err != nil {
		t.Fatalf("workerTaskCommand returned error: %v", err)
	}
	if spec.Source != "commands.verify" {
		t.Fatalf("source = %q, want commands.verify", spec.Source)
	}
	if got := spec.Args; len(got) != 3 || got[0] != "go" || got[2] != "./..." {
		t.Fatalf("args = %#v", got)
	}
}

func TestWorkerTaskCommandInjectsScopedCloudflareCredential(t *testing.T) {
	t.Setenv("RELEASE_CENTER_CREDENTIAL_CF_TOKEN_RELEASE_PROD", "scoped-token")
	t.Setenv("CLOUDFLARE_API_TOKEN", "global-token")
	metadata := mustRawJSON(t, map[string]any{
		"provider":              "cloudflare_pages",
		"credential_ref":        "cf_token_release_prod",
		"cloudflare_account_id": "account-1",
		"prepared_command":      []string{"wrangler", "pages", "deploy", "dist"},
	})

	spec, err := workerTaskCommand(appreleases.WorkerTaskAdmin{TaskType: "deploy", Action: "cloudflare_pages", Metadata: metadata})
	if err != nil {
		t.Fatalf("workerTaskCommand returned error: %v", err)
	}
	if spec.CredentialRef != "cf_token_release_prod" {
		t.Fatalf("CredentialRef = %q", spec.CredentialRef)
	}
	if spec.Env["CLOUDFLARE_API_TOKEN"] != "scoped-token" {
		t.Fatalf("CLOUDFLARE_API_TOKEN was not injected from scoped credential env: %#v", spec.Env)
	}
	if spec.Env["CLOUDFLARE_ACCOUNT_ID"] != "account-1" {
		t.Fatalf("CLOUDFLARE_ACCOUNT_ID was not injected: %#v", spec.Env)
	}
	if spec.EnvSources["CLOUDFLARE_API_TOKEN"] != "RELEASE_CENTER_CREDENTIAL_CF_TOKEN_RELEASE_PROD" {
		t.Fatalf("unexpected env source: %#v", spec.EnvSources)
	}
}

func TestWorkerTaskCommandRequiresScopedCloudflareCredential(t *testing.T) {
	t.Setenv("RELEASE_CENTER_CREDENTIAL_CF_TOKEN_MISSING", "")
	t.Setenv("CLOUDFLARE_API_TOKEN_CF_TOKEN_MISSING", "")
	t.Setenv("CF_TOKEN_MISSING", "")
	t.Setenv("CLOUDFLARE_API_TOKEN", "")
	metadata := mustRawJSON(t, map[string]any{
		"provider":         "cloudflare_pages",
		"credential_ref":   "cf_token_missing",
		"prepared_command": []string{"wrangler", "pages", "deploy", "dist"},
	})

	_, err := workerTaskCommand(appreleases.WorkerTaskAdmin{TaskType: "deploy", Action: "cloudflare_pages", Metadata: metadata})
	if err == nil {
		t.Fatal("expected missing credential env error")
	}
	if got := err.Error(); got != `cloudflare credential_ref "cf_token_missing" is set but no matching token env was found` {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestWorkerExecutionMetadataDoesNotExposeCredentialValue(t *testing.T) {
	metadata := workerExecutionMetadata(workerCommandSpec{
		Source:        "prepared_command",
		Display:       "wrangler pages deploy dist",
		CredentialRef: "cf_token_release_prod",
		Env: map[string]string{
			"CLOUDFLARE_API_TOKEN": "scoped-token",
		},
		EnvSources: map[string]string{
			"CLOUDFLARE_API_TOKEN": "RELEASE_CENTER_CREDENTIAL_CF_TOKEN_RELEASE_PROD",
		},
	}, 0, 0, []byte("https://release-center.pages.dev\n"))

	if metadata["worker_credential_ref"] != "cf_token_release_prod" {
		t.Fatalf("credential ref missing from metadata: %#v", metadata)
	}
	keys, ok := metadata["worker_env_keys"].([]string)
	if !ok || len(keys) != 1 || keys[0] != "CLOUDFLARE_API_TOKEN" {
		t.Fatalf("unexpected env keys metadata: %#v", metadata)
	}
	if text, err := json.Marshal(metadata); err != nil || string(text) == "" || containsString(string(text), "scoped-token") {
		t.Fatalf("metadata leaked token or failed to marshal: %s err=%v", string(text), err)
	}
}

func TestWorkerOutputLinesKeepsTail(t *testing.T) {
	var lines []byte
	for i := 0; i < 205; i++ {
		lines = append(lines, []byte("line\n")...)
	}
	got := workerOutputLines(lines)
	if len(got) != 200 {
		t.Fatalf("len = %d, want 200", len(got))
	}
}

func TestFirstURLTrimsPunctuation(t *testing.T) {
	got := firstURL(`deployed to https://example.pages.dev).`)
	if got != "https://example.pages.dev" {
		t.Fatalf("firstURL = %q", got)
	}
}

func mustRawJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func containsString(value, needle string) bool {
	return strings.Contains(value, needle)
}

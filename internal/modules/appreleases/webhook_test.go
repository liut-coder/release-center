package appreleases

import (
	"net/http/httptest"
	"testing"
)

func TestWebhookEventFromRequestNormalizesCreateTagRef(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/webhooks/github", nil)
	req.Header.Set("X-GitHub-Event", "create")

	event := webhookEventFromRequest("github", req, map[string]any{
		"ref_type": "tag",
		"ref":      "v1.2.3",
		"repository": map[string]any{
			"full_name": "liut-coder/release-center",
		},
	})

	if event.Ref != "refs/tags/v1.2.3" {
		t.Fatalf("expected normalized tag ref, got %q", event.Ref)
	}
	if event.Repository != "liut-coder/release-center" {
		t.Fatalf("expected repository full name, got %q", event.Repository)
	}
}

func TestWebhookEventFromRequestNormalizesWorkflowBranch(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/webhooks/github", nil)
	req.Header.Set("X-GitHub-Event", "workflow_run")

	event := webhookEventFromRequest("github", req, map[string]any{
		"workflow_run": map[string]any{
			"head_branch": "main",
			"head_sha":    "abc123",
		},
	})

	if event.Ref != "refs/heads/main" {
		t.Fatalf("expected normalized workflow branch, got %q", event.Ref)
	}
	if event.CommitSHA != "abc123" {
		t.Fatalf("expected workflow head sha, got %q", event.CommitSHA)
	}
}

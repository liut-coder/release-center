package appreleases

import "testing"

func TestCloudflareDeploymentCommandPages(t *testing.T) {
	cmd := cloudflareDeploymentCommand(DeploymentTargetAdmin{
		Provider:              "cloudflare_pages",
		CloudflareProjectName: "release-center-admin",
	}, CreateDeploymentRequest{Metadata: map[string]any{"artifact_path": "web/dist"}})

	want := []string{"wrangler", "pages", "deploy", "web/dist", "--project-name", "release-center-admin"}
	if len(cmd) != len(want) {
		t.Fatalf("expected %d args, got %#v", len(want), cmd)
	}
	for i := range want {
		if cmd[i] != want[i] {
			t.Fatalf("cmd[%d] = %q, want %q", i, cmd[i], want[i])
		}
	}
}

func TestCloudflareDeploymentCommandR2(t *testing.T) {
	cmd := cloudflareDeploymentCommand(DeploymentTargetAdmin{
		Provider:             "cloudflare_r2",
		CloudflareBucketName: "release-artifacts",
	}, CreateDeploymentRequest{Metadata: map[string]any{
		"artifact_path": "dist/web.tar.gz",
		"object_key":    "web.tar.gz",
	}})

	want := []string{"wrangler", "r2", "object", "put", "release-artifacts/web.tar.gz", "--file", "dist/web.tar.gz"}
	if len(cmd) != len(want) {
		t.Fatalf("expected %d args, got %#v", len(want), cmd)
	}
	for i := range want {
		if cmd[i] != want[i] {
			t.Fatalf("cmd[%d] = %q, want %q", i, cmd[i], want[i])
		}
	}
}

func TestNormalizeDeploymentStatus(t *testing.T) {
	if got := normalizeDeploymentStatus(" SUCCESS "); got != "success" {
		t.Fatalf("expected success, got %q", got)
	}
	if got := normalizeDeploymentStatus("unknown"); got != "queued" {
		t.Fatalf("expected fallback queued, got %q", got)
	}
}

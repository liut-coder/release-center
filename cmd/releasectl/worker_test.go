package main

import (
	"encoding/json"
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

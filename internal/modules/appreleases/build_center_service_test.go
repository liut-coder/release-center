package appreleases

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTailLogLinesReturnsTailAndTruncation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "buildctl.log")
	if err := os.WriteFile(path, []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	lines, truncated, err := tailLogLines(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !truncated {
		t.Fatal("expected log tail to be marked truncated")
	}
	if len(lines) != 2 || lines[0] != "two" || lines[1] != "three" {
		t.Fatalf("unexpected lines: %#v", lines)
	}
}

func TestTailLogLinesHandlesEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "buildctl.log")
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	lines, truncated, err := tailLogLines(path, 200)
	if err != nil {
		t.Fatal(err)
	}
	if truncated || len(lines) != 0 {
		t.Fatalf("expected empty non-truncated lines, got truncated=%v lines=%#v", truncated, lines)
	}
}

func TestWebhookRefMatchesSupportsSimpleGlob(t *testing.T) {
	for _, tc := range []struct {
		name    string
		pattern string
		ref     string
		want    bool
	}{
		{name: "all", pattern: "*", ref: "refs/heads/main", want: true},
		{name: "exact", pattern: "refs/heads/main", ref: "refs/heads/main", want: true},
		{name: "tag glob", pattern: "refs/tags/*", ref: "refs/tags/v1.0.0", want: true},
		{name: "branch mismatch", pattern: "refs/heads/release/*", ref: "refs/heads/main", want: false},
		{name: "middle glob", pattern: "refs/heads/*/hotfix", ref: "refs/heads/prod/hotfix", want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := webhookRefMatches(tc.pattern, tc.ref); got != tc.want {
				t.Fatalf("webhookRefMatches(%q, %q) = %v, want %v", tc.pattern, tc.ref, got, tc.want)
			}
		})
	}
}

func TestWebhookRouteAllowsPushAndTagSwitches(t *testing.T) {
	route := WebhookBuildRoute{EventType: "push", TriggerOnPush: true, TriggerOnTag: false}
	if !webhookRouteAllowsEvent(route, WebhookEventRequest{EventType: "push", Ref: "refs/heads/main"}) {
		t.Fatal("expected branch push to be allowed")
	}
	if webhookRouteAllowsEvent(route, WebhookEventRequest{EventType: "push", Ref: "refs/tags/v1.0.0"}) {
		t.Fatal("expected tag push to be blocked when trigger_on_tag is false")
	}
	if webhookRouteAllowsEvent(route, WebhookEventRequest{EventType: "workflow_run", Ref: "refs/heads/main"}) {
		t.Fatal("expected different event type to be blocked")
	}
}

func TestBuildRefFromWebhookRef(t *testing.T) {
	for _, tc := range []struct {
		ref  string
		want string
	}{
		{ref: "refs/heads/main", want: "main"},
		{ref: "refs/tags/v1.0.0", want: "v1.0.0"},
		{ref: "feature/demo", want: "feature/demo"},
	} {
		if got := buildRefFromWebhookRef(tc.ref); got != tc.want {
			t.Fatalf("buildRefFromWebhookRef(%q) = %q, want %q", tc.ref, got, tc.want)
		}
	}
}

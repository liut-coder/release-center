package appreleases

import (
	"strings"
	"testing"
)

func TestNormalizeReleaseUnitType(t *testing.T) {
	if got := normalizeReleaseUnitType(" WEB "); got != "web" {
		t.Fatalf("expected web, got %q", got)
	}
	if got := normalizeReleaseUnitType("custom"); got != "custom" {
		t.Fatalf("expected custom passthrough, got %q", got)
	}
}

func TestNormalizeReleasePlanStatus(t *testing.T) {
	if got := normalizeReleasePlanStatus(" PAUSED "); got != "paused" {
		t.Fatalf("expected paused, got %q", got)
	}
	if got := normalizeReleasePlanStatus("unknown"); got != "draft" {
		t.Fatalf("expected draft fallback, got %q", got)
	}
}

func TestNormalizeReleasePlanArtifactsDefaultsNameAndType(t *testing.T) {
	items := normalizeReleasePlanArtifacts([]ReleasePlanArtifactRequest{{FileName: "web-dist.tar.gz"}})
	if len(items) != 1 {
		t.Fatalf("expected one artifact, got %#v", items)
	}
	if items[0].ArtifactName != "web-dist.tar.gz" || items[0].ArtifactType != "artifact" {
		t.Fatalf("unexpected artifact defaults: %#v", items[0])
	}
}

func TestDefaultReleasePlanKeyIncludesUnitEnvironmentAndVersion(t *testing.T) {
	key := defaultReleasePlanKey(CreateReleasePlanRequest{
		UnitKey:        "admin-web",
		EnvironmentKey: "prod",
		VersionName:    "1.0.0",
	})
	if !strings.HasPrefix(key, "admin-web-prod-1.0.0-") {
		t.Fatalf("unexpected key prefix: %q", key)
	}
}

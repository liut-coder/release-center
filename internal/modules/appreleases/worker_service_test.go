package appreleases

import "testing"

func TestNormalizeWorkerLabelsDeduplicatesAndLowercases(t *testing.T) {
	labels := normalizeWorkerLabels([]string{" Linux ", "linux", "Docker", "", "ANDROID"})
	want := []string{"linux", "docker", "android"}
	if len(labels) != len(want) {
		t.Fatalf("expected %d labels, got %#v", len(want), labels)
	}
	for i := range want {
		if labels[i] != want[i] {
			t.Fatalf("labels[%d] = %q, want %q", i, labels[i], want[i])
		}
	}
}

func TestNormalizeWorkerStatus(t *testing.T) {
	if got := normalizeWorkerStatus(" BUSY "); got != "busy" {
		t.Fatalf("expected busy, got %q", got)
	}
	if got := normalizeWorkerStatus("unexpected"); got != "online" {
		t.Fatalf("expected fallback online, got %q", got)
	}
}

func TestNormalizeWorkerLogLinesKeepsTail(t *testing.T) {
	lines := make([]string, 205)
	for i := range lines {
		lines[i] = "line"
	}
	lines[204] = "last\r\n"
	got := normalizeWorkerLogLines(lines)
	if len(got) != 200 {
		t.Fatalf("expected 200 lines, got %d", len(got))
	}
	if got[len(got)-1] != "last" {
		t.Fatalf("expected CRLF trim, got %q", got[len(got)-1])
	}
}

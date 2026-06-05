package appreleases

import (
	"context"
	"encoding/json"
	"testing"
)

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

func TestSaveWorkerTaskArtifactsMirrorsBuildCenterArtifacts(t *testing.T) {
	store := &workerArtifactMirrorStore{}
	service := &Service{store: store}

	resp, err := service.SaveWorkerTaskArtifacts(context.Background(), "task-1", WorkerTaskArtifactsRequest{
		WorkerKey:  "worker-1",
		LeaseToken: "lease-1",
		Artifacts: []WorkerTaskArtifact{{
			Name:         "admin-web-dist",
			ArtifactType: "web_dist",
			FileName:     "dist.tar.gz",
			LocalPath:    "/srv/builds/dist.tar.gz",
			SizeBytes:    1024,
			SHA256:       "abc123",
			DownloadURL:  "https://example.test/dist.tar.gz",
			Metadata:     map[string]any{"channel": "stable"},
		}},
	})
	if err != nil {
		t.Fatalf("SaveWorkerTaskArtifacts() error = %v", err)
	}
	if !resp.OK || resp.Task == nil {
		t.Fatalf("unexpected response: %#v", resp)
	}
	assertWorkerArtifactMirror(t, store, "run-1", "uploaded")
}

func TestCompleteWorkerTaskMirrorsBuildCenterArtifacts(t *testing.T) {
	store := &workerArtifactMirrorStore{}
	service := &Service{store: store}

	resp, err := service.CompleteWorkerTask(context.Background(), "task-1", WorkerTaskCompleteRequest{
		WorkerKey:  "worker-1",
		LeaseToken: "lease-1",
		Artifacts: []WorkerTaskArtifact{{
			Name:         "server-bin",
			ArtifactType: "binary",
			FileName:     "release-center-server",
			LocalPath:    "/srv/builds/release-center-server",
			SizeBytes:    2048,
			SHA256:       "def456",
		}},
	})
	if err != nil {
		t.Fatalf("CompleteWorkerTask() error = %v", err)
	}
	if !resp.OK || resp.Task == nil {
		t.Fatalf("unexpected response: %#v", resp)
	}
	assertWorkerArtifactMirror(t, store, "run-1", "local")
}

func assertWorkerArtifactMirror(t *testing.T, store *workerArtifactMirrorStore, runID, uploadStatus string) {
	t.Helper()
	if store.mirroredRunID != runID {
		t.Fatalf("expected mirrored run %q, got %q", runID, store.mirroredRunID)
	}
	if len(store.mirroredArtifacts) != 1 {
		t.Fatalf("expected one mirrored artifact, got %#v", store.mirroredArtifacts)
	}
	artifact := store.mirroredArtifacts[0]
	if artifact.RunID != runID || artifact.UploadStatus != uploadStatus {
		t.Fatalf("unexpected mirrored artifact: %#v", artifact)
	}
	var metadata map[string]any
	if err := json.Unmarshal(artifact.Metadata, &metadata); err != nil {
		t.Fatalf("artifact metadata should be json: %v", err)
	}
	if metadata["source"] != "worker_task_artifact" || metadata["worker_task_id"] != "task-1" || metadata["task_type"] != "build" {
		t.Fatalf("unexpected artifact metadata: %#v", metadata)
	}
}

type workerArtifactMirrorStore struct {
	Store
	WorkerStore
	WorkerArtifactMirrorStore
	mirroredRunID     string
	mirroredArtifacts []BuildCenterRunArtifact
}

func (s *workerArtifactMirrorStore) SaveWorkerTaskArtifacts(_ context.Context, taskID string, _ WorkerTaskArtifactsRequest) (WorkerTaskAdmin, error) {
	return workerArtifactMirrorTask(taskID), nil
}

func (s *workerArtifactMirrorStore) CompleteWorkerTask(_ context.Context, taskID string, _ WorkerTaskCompleteRequest) (WorkerTaskAdmin, error) {
	return workerArtifactMirrorTask(taskID), nil
}

func (s *workerArtifactMirrorStore) ReplaceBuildCenterRunArtifacts(_ context.Context, runID string, artifacts []BuildCenterRunArtifact) error {
	s.mirroredRunID = runID
	s.mirroredArtifacts = artifacts
	return nil
}

func workerArtifactMirrorTask(taskID string) WorkerTaskAdmin {
	return WorkerTaskAdmin{
		ID:         taskID,
		WorkerID:   "worker-id-1",
		BuildRunID: "run-1",
		TaskType:   "build",
		Action:     "all",
		Status:     "running",
	}
}

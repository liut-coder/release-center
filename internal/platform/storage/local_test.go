package storage

import (
	"context"
	"strings"
	"testing"
)

func TestLocalOpenRejectsPathTraversal(t *testing.T) {
	store := NewLocal(t.TempDir())
	_, err := store.Open(context.Background(), "../secret.txt")
	if err == nil || !strings.Contains(err.Error(), "invalid storage key") {
		t.Fatalf("expected invalid storage key error, got %v", err)
	}
}

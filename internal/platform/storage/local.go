package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type Local struct {
	root string
}

func NewLocal(root string) *Local {
	return &Local{root: root}
}

func (s *Local) Save(ctx context.Context, namespace, filename string, body io.Reader) (string, int64, error) {
	select {
	case <-ctx.Done():
		return "", 0, ctx.Err()
	default:
	}

	key := filepath.Join(namespace, uuid.NewString()+"-"+filepath.Base(filename))
	fullPath := filepath.Join(s.root, key)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return "", 0, err
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	n, err := io.Copy(file, body)
	if err != nil {
		return "", 0, err
	}
	return key, n, nil
}

func (s *Local) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	cleanKey := filepath.Clean(key)
	if cleanKey == "." || filepath.IsAbs(cleanKey) || cleanKey == ".." || strings.HasPrefix(cleanKey, ".."+string(os.PathSeparator)) {
		return nil, fmt.Errorf("invalid storage key")
	}
	return os.Open(filepath.Join(s.root, cleanKey))
}

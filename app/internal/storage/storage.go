package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"menata.id/app/internal/auth"
)

// Store abstracts uploaded-file bytes behind Put/Get/Delete, keyed by an
// opaque token this package generates -- see doc.go for why this exists.
type Store interface {
	// Put writes data and returns the key it can later be retrieved or
	// deleted by. contentType decides the key's extension (extensionFor)
	// and is also what a later Get returns back out.
	Put(data []byte, contentType string) (key string, err error)

	// Get reads back the bytes stored under key, plus the content type
	// inferred from key's extension (contentTypeForExtension) -- empty
	// string if the extension isn't one of the known types.
	Get(key string) (data []byte, contentType string, err error)

	// Delete removes the file stored under key. Deleting a key that
	// doesn't exist is not an error.
	Delete(key string) error
}

// LocalDisk is the day-one Store backend -- behavior-identical to
// prototype/go's own uploads/ directory (CAP-F06): files live under dir,
// gitignored, created on first Put via os.MkdirAll.
type LocalDisk struct {
	dir string
}

// NewLocalDisk returns a Store backed by the local filesystem, writing
// under dir (created on first Put if it doesn't exist yet).
func NewLocalDisk(dir string) *LocalDisk {
	return &LocalDisk{dir: dir}
}

func (s *LocalDisk) Put(data []byte, contentType string) (string, error) {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return "", err
	}
	token, err := auth.NewToken() // 32 bytes of entropy, base64url -- Get's own trust boundary
	if err != nil {
		return "", err
	}
	key := token + extensionFor(contentType)
	if err := os.WriteFile(filepath.Join(s.dir, key), data, 0o644); err != nil {
		return "", err
	}
	return key, nil
}

func (s *LocalDisk) Get(key string) ([]byte, string, error) {
	safeKey := filepath.Base(key) // strips any path-traversal attempt before touching the filesystem
	data, err := os.ReadFile(filepath.Join(s.dir, safeKey))
	if err != nil {
		return nil, "", err
	}
	return data, contentTypeForExtension(filepath.Ext(safeKey)), nil
}

func (s *LocalDisk) Delete(key string) error {
	safeKey := filepath.Base(key)
	err := os.Remove(filepath.Join(s.dir, safeKey))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete %s: %w", safeKey, err)
	}
	return nil
}

func extensionFor(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "application/pdf":
		return ".pdf"
	default:
		return ""
	}
}

// contentTypeForExtension is extensionFor's reverse, used by Get --
// deliberately a fixed table, not the OS's mime.types (stdlib's
// mime.TypeByExtension falls back to reading /etc/mime.types, which isn't
// guaranteed present or identical across every host this might run on).
func contentTypeForExtension(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".pdf":
		return "application/pdf"
	default:
		return ""
	}
}

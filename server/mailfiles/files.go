// Package mailfiles owns durable, private capture files on the persistent volume.
package mailfiles

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

var safeID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)

type Store struct{ root string }

func New(root string) (*Store, error) {
	if root == "" {
		return nil, errors.New("EMAIL_SOURCE_DIR is required")
	}
	if e := os.MkdirAll(root, 0700); e != nil {
		return nil, e
	}
	f, e := os.CreateTemp(root, ".probe-")
	if e != nil {
		return nil, e
	}
	f.Close()
	os.Remove(f.Name())
	return &Store{root}, nil
}
func (s *Store) path(source, id string) (string, error) {
	if !safeID.MatchString(source) || !safeID.MatchString(id) {
		return "", errors.New("invalid source identifier")
	}
	return filepath.Join(s.root, source, id), nil
}
func (s *Store) SaveOriginal(ctx context.Context, id string, raw io.Reader) (string, int64, error) {
	return s.Save(ctx, id, "original", raw, 20<<20)
}
func (s *Store) Save(ctx context.Context, source, id string, raw io.Reader, limit int64) (digest string, size int64, err error) {
	path, e := s.path(source, id)
	if e != nil {
		return "", 0, e
	}
	if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return "", 0, e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".write-")
	if e != nil {
		return "", 0, e
	}
	defer os.Remove(f.Name())
	defer f.Close()
	h := sha256.New()
	size, e = io.Copy(io.MultiWriter(f, h), io.LimitReader(raw, limit+1))
	if e != nil {
		return "", 0, e
	}
	if size > limit {
		return "", 0, errors.New("attachment exceeds limit")
	}
	if e = ctx.Err(); e != nil {
		return "", 0, e
	}
	if e = f.Sync(); e != nil {
		return "", 0, e
	}
	if e = f.Close(); e != nil {
		return "", 0, e
	}
	if e = os.Rename(f.Name(), path); e != nil {
		return "", 0, e
	}
	d, e := os.Open(filepath.Dir(path))
	if e != nil {
		return "", 0, e
	}
	e = d.Sync()
	d.Close()
	if e != nil {
		return "", 0, e
	}
	return hex.EncodeToString(h.Sum(nil)), size, nil
}
func (s *Store) Open(source, id string) (io.ReadCloser, error) {
	p, e := s.path(source, id)
	if e != nil {
		return nil, e
	}
	return os.Open(p)
}
func (s *Store) DeleteSource(source string) error {
	p, e := s.path(source, "original")
	if e != nil {
		return e
	}
	return os.RemoveAll(filepath.Dir(p))
}

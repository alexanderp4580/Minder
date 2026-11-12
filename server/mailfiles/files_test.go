package mailfiles

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOriginalBytesAndPaths(t *testing.T) {
	s, e := New(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	raw := []byte{0, 255, 13, 10, 42}
	_, n, e := s.SaveOriginal(context.Background(), "source", bytes.NewReader(raw))
	if e != nil || n != 5 {
		t.Fatal(n, e)
	}
	f, e := s.Open("source", "original")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	got, _ := io.ReadAll(f)
	if !bytes.Equal(raw, got) {
		t.Fatal(got)
	}
	if _, e = s.Open("../escape", "original"); e == nil {
		t.Fatal("traversal accepted")
	}
	if e = s.DeleteSource("source"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Open("source", "original"); e == nil {
		t.Fatal("deleted content accessible")
	}
}

func TestStaleUnreferencedFilesAreRemoved(t *testing.T) {
	root := t.TempDir()
	files, e := New(root)
	if e != nil {
		t.Fatal(e)
	}
	if _, _, e = files.SaveOriginal(context.Background(), "orphan", bytes.NewBufferString("mail")); e != nil {
		t.Fatal(e)
	}
	old := time.Now().Add(-48 * time.Hour)
	if e = os.Chtimes(filepath.Join(root, "orphan"), old, old); e != nil {
		t.Fatal(e)
	}
	if e = files.Cleanup(map[string]bool{}, time.Now()); e != nil {
		t.Fatal(e)
	}
	if _, e = files.Open("orphan", "original"); e == nil {
		t.Fatal("orphan retained")
	}
}

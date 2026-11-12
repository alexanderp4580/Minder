package mailfiles

import (
	"bytes"
	"context"
	"io"
	"testing"
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

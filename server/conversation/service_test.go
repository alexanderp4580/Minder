package conversation_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexanderp4580/Minder/server/conversation"
	"github.com/alexanderp4580/Minder/server/domain"
	"github.com/alexanderp4580/Minder/server/store"
)

type heldReply struct{ started, cancelled, release chan struct{} }

func (r heldReply) Reply(ctx context.Context, _ string) ([]domain.Entry, error) {
	close(r.started)
	<-ctx.Done()
	close(r.cancelled)
	<-r.release
	return nil, ctx.Err()
}
func TestCloseCancelsAndWaitsForAcceptedReply(t *testing.T) {
	database, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	reply := heldReply{make(chan struct{}), make(chan struct{}), make(chan struct{})}
	service := conversation.New(context.Background(), database, reply)
	accepted, err := service.Accept("  Hello  ")
	if err != nil || accepted.Entry.Text != "Hello" {
		t.Fatal(accepted, err)
	}
	<-reply.started
	if _, err = service.Accept("Second"); !errors.Is(err, domain.ErrBusy) {
		t.Fatalf("want busy, got %v", err)
	}
	closed := make(chan struct{})
	go func() { service.Close(); close(closed) }()
	select {
	case <-reply.cancelled:
	case <-time.After(time.Second):
		t.Fatal("accepted reply not cancelled")
	}
	select {
	case <-closed:
		t.Fatal("close returned before reply finished")
	default:
	}
	close(reply.release)
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("close did not finish")
	}
	if _, err = service.Accept("After close"); err == nil {
		t.Fatal("accepted after close")
	}
	entries, err := database.History(0, 20)
	if err != nil || len(entries) != 2 || entries[1].Kind != "error" || entries[1].Text != "Minder could not finish. Try again." {
		t.Fatalf("accepted reply not recorded: %+v %v", entries, err)
	}
}

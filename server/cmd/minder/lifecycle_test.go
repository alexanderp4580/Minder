package main

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestServeWaitsForWorkerShutdown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	workerCancelled, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- serve(ctx, &http.Server{}, listener, func(ctx context.Context) { <-ctx.Done(); close(workerCancelled); <-release })
	}()
	cancel()
	select {
	case <-workerCancelled:
	case <-time.After(time.Second):
		t.Fatal("worker was not cancelled")
	}
	select {
	case err := <-done:
		t.Fatalf("returned before worker stopped: %v", err)
	default:
	}
	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not stop")
	}
}

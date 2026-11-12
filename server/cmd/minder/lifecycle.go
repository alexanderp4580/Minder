package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// serve owns the HTTP and worker lifetimes. Storage closes after both return.
func serve(parent context.Context, server *http.Server, listener net.Listener, worker func(context.Context)) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	server.BaseContext = func(net.Listener) context.Context { return ctx }
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		worker(ctx)
		slog.Info("background worker stopped")
	}()
	serverDone := make(chan error, 1)
	go func() { serverDone <- server.Serve(listener) }()
	var err error
	select {
	case err = <-serverDone:
		slog.Error("http server stopped unexpectedly", "error", err)
		cancel()
		server.Close()
	case <-ctx.Done():
		slog.Info("shutdown requested", "grace_s", 10)
		started := time.Now()
		timeout, stop := context.WithTimeout(context.Background(), 10*time.Second)
		err = server.Shutdown(timeout)
		stop()
		if err != nil {
			slog.Warn("graceful shutdown failed; closing connections", "error", err, "waited_ms", time.Since(started).Milliseconds())
			server.Close()
		}
		<-serverDone
		slog.Info("http server stopped", "waited_ms", time.Since(started).Milliseconds())
	}
	cancel()
	<-workerDone
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

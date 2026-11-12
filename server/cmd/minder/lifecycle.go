package main

import (
	"context"
	"errors"
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
	go func() { defer close(workerDone); worker(ctx) }()
	serverDone := make(chan error, 1)
	go func() { serverDone <- server.Serve(listener) }()
	var err error
	select {
	case err = <-serverDone:
		cancel()
		server.Close()
	case <-ctx.Done():
		timeout, stop := context.WithTimeout(context.Background(), 10*time.Second)
		err = server.Shutdown(timeout)
		stop()
		if err != nil {
			server.Close()
		}
		<-serverDone
	}
	cancel()
	<-workerDone
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

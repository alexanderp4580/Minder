package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/alexanderp4580/Minder/server/access"
	"github.com/alexanderp4580/Minder/server/api"
	"github.com/alexanderp4580/Minder/server/assistant"
	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/config"
	"github.com/alexanderp4580/Minder/server/conversation"
	"github.com/alexanderp4580/Minder/server/openrouter"
	"github.com/alexanderp4580/Minder/server/push"
	"github.com/alexanderp4580/Minder/server/reminder"
	"github.com/alexanderp4580/Minder/server/store"
)

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run() error {
	settings, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	authentication, err := access.FromEnvironment(os.Getenv)
	if err != nil {
		return err
	}
	database, err := store.Open(settings.DatabasePath)
	if err != nil {
		return errors.New("Database could not open")
	}
	defer database.Close()
	commands := command.New(database)
	model := &openrouter.Client{Key: settings.ModelKey, Model: settings.Model, URL: settings.ModelEndpoint}
	assistantService := assistant.New(database, commands, model, settings.MonthlyLimitEUR, settings.USDToEUR)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	messages := conversation.New(ctx, database, assistantService)
	defer messages.Close()
	reminders := reminder.New(database)
	handler := api.New(database, commands, messages, reminders, authentication, settings.WebDir)
	handler.Branding = settings.BrandDir
	srv := &http.Server{
		Addr:              settings.ListenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}
	worker := &reminder.Worker{Repository: database, Scheduler: reminders}
	if raw := settings.FirebaseServiceAccount; raw != "" {
		sender, err := push.NewFCM(raw)
		if err != nil {
			return errors.New("Firebase service account configuration is invalid")
		}
		worker.Sender = sender
		pushConfig, err := api.ParsePushConfig(settings.FirebaseAndroidConfig)
		if err != nil || pushConfig == nil || pushConfig.ProjectID != sender.ProjectID {
			return errors.New("Firebase Android configuration is invalid")
		}
		handler.PushConfig = pushConfig
	} else {
		log.Print("Firebase push is not configured; reminders remain available in chat")
	}
	listener, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return err
	}
	log.Printf("Minder listening on %s", listener.Addr())
	return serve(ctx, srv, listener, worker.Run)
}

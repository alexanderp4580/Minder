package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	"github.com/alexanderp4580/Minder/server/capture"
	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/config"
	"github.com/alexanderp4580/Minder/server/conversation"
	"github.com/alexanderp4580/Minder/server/mailauth"
	"github.com/alexanderp4580/Minder/server/mailfiles"
	"github.com/alexanderp4580/Minder/server/mailparse"
	"github.com/alexanderp4580/Minder/server/openrouter"
	"github.com/alexanderp4580/Minder/server/push"
	"github.com/alexanderp4580/Minder/server/reminder"
	"github.com/alexanderp4580/Minder/server/store"
)

func main() {
	logger, err := newLogger(os.Getenv, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	slog.SetDefault(logger)
	if err := run(logger); err != nil {
		slog.Error("startup or run failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	settings, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	authentication, err := access.FromEnvironment(os.Getenv)
	if err != nil {
		return err
	}
	authMode := os.Getenv("AUTH_MODE")
	if authMode == "" {
		authMode = "cloudflare"
	}
	logStartup(logger, settings, authMode)
	if authentication == nil {
		slog.Warn("authentication is disabled (development or test mode)")
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
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
	worker := &reminder.Worker{Repository: database, Scheduler: reminders}
	var emailService *capture.Service
	if settings.Email.Enabled {
		verifier, e := mailauth.FromEnvironment(os.Getenv)
		if e != nil {
			return e
		}
		files, e := mailfiles.New(settings.Email.SourceDir)
		if e != nil {
			return errors.New("Email source storage is unavailable")
		}
		emailService = capture.New(database, files, assistantService)
		emailService.Parser = &mailparse.Parser{}
		emailService.Preview = command.PreviewIn
		handler.EmailVerifier = verifier
		handler.EmailIngestor = emailService
		handler.Captures = emailService
		slog.Info("email capture configured", "enabled", true)
	}
	if raw := settings.FirebaseServiceAccount; raw != "" {
		sender, err := push.NewFCM(raw)
		if err != nil {
			return errors.New("Firebase service account configuration is invalid")
		}
		worker.Sender = sender
		if emailService != nil {
			emailService.Sender = sender
		}
		slog.Info("firebase push configured", "project_id", sender.ProjectID)
		pushConfig, err := api.ParsePushConfig(settings.FirebaseAndroidConfig)
		if err != nil || pushConfig == nil || pushConfig.ProjectID != sender.ProjectID {
			return errors.New("Firebase Android configuration is invalid")
		}
		handler.PushConfig = pushConfig
	} else {
		slog.Warn("firebase push is not configured; reminders remain available in chat")
	}
	listener, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return err
	}
	slog.Info("Minder listening", "addr", listener.Addr().String())
	workers := []func(context.Context){worker.Run}
	if emailService != nil {
		workers = append(workers, emailService.Run)
	}
	return serve(ctx, srv, listener, func(ctx context.Context) { runWorkers(ctx, workers...) })
}

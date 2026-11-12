package main

import (
	"context"
	"log"
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
	"github.com/alexanderp4580/Minder/server/reminder"
	"github.com/alexanderp4580/Minder/server/store"
)

func main() {
	settings, err := config.Load(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	authentication, err := access.FromEnvironment(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	database, err := store.Open(settings.DatabasePath)
	if err != nil {
		log.Fatal("Database could not open")
	}
	defer database.Close()
	commands := command.New(database)
	model := &assistant.OpenRouter{Key: settings.ModelKey, Model: settings.Model, URL: settings.ModelEndpoint}
	assistantService := assistant.New(database, commands, model, settings.MonthlyLimitEUR, settings.USDToEUR)
	handler := api.New(database, commands, assistantService, authentication, settings.WebDir)
	handler.Branding = settings.BrandDir
	srv := &http.Server{
		Addr:              settings.ListenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	worker := &reminder.Worker{Store: database}
	if raw := settings.FirebaseServiceAccount; raw != "" {
		sender, err := reminder.NewFCM(raw)
		if err != nil {
			log.Fatal("Firebase service account configuration is invalid")
		}
		worker.Sender = sender
		pushConfig, err := api.ParsePushConfig(settings.FirebaseAndroidConfig)
		if err != nil || pushConfig == nil || pushConfig.ProjectID != sender.ProjectID {
			log.Fatal("Firebase Android configuration is invalid")
		}
		handler.PushConfig = pushConfig
	} else {
		log.Print("Firebase push is not configured; reminders remain available in chat")
	}
	go worker.Run(ctx)
	go func() {
		<-ctx.Done()
		timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(timeout); err != nil {
			log.Print("Server shutdown did not finish before the deadline")
		}
	}()
	log.Printf("Minder listening on %s", srv.Addr)
	if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

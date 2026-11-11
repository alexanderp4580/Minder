package main

import (
	"context"
	"github.com/alexanderp4580/Minder/server/api"
	"github.com/alexanderp4580/Minder/server/assistant"
	"github.com/alexanderp4580/Minder/server/command"
	"github.com/alexanderp4580/Minder/server/reminder"
	"github.com/alexanderp4580/Minder/server/store"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	_ "time/tzdata"
)

func env(k, f string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return f
}
func number(k string, f float64) float64 {
	if v := os.Getenv(k); v != "" {
		n, e := strconv.ParseFloat(v, 64)
		if e != nil || n <= 0 {
			log.Fatalf("Invalid %s", k)
		}
		return n
	}
	return f
}
func main() {
	mode := env("AUTH_MODE", "cloudflare")
	environment := env("ENV", "production")
	if e := api.ValidateMode(mode, environment); e != nil {
		log.Fatal(e)
	}
	s, e := store.Open(env("DATABASE_PATH", "minder.db"))
	if e != nil {
		log.Fatal("Database could not open")
	}
	defer s.Close()
	c := command.New(s)
	endpoint := os.Getenv("OPENROUTER_URL")
	if environment == "production" && endpoint != "" && endpoint != "https://openrouter.ai/api/v1/chat/completions" {
		log.Fatal("Custom model endpoint is forbidden in production")
	}
	model := &assistant.OpenRouter{Key: os.Getenv("OPENROUTER_API_KEY"), Model: env("MODEL", "anthropic/claude-haiku-5.5"), URL: endpoint}
	a := assistant.New(s, c, model, number("MONTHLY_LIMIT_EUR", 5), number("USD_TO_EUR", 1))
	var verifier *api.Verifier
	if mode == "cloudflare" {
		verifier = &api.Verifier{Issuer: env("CF_ISSUER", "https://liveleds.cloudflareaccess.com"), Audience: env("CF_AUDIENCE", "11f11785388ae33f2f78007aa26db1bf047457ff7f19758eddd797ef4c151c54"), Email: env("OWNER_EMAIL", "alexanderp4580@gmail.com")}
	}
	handler := api.New(s, c, a, verifier, env("WEB_DIR", "../app/build/web"))
	srv := &http.Server{Addr: env("LISTEN_ADDR", "127.0.0.1:8080"), Handler: handler, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 16}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	worker := &reminder.Worker{Store: s}
	if raw := os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON"); raw != "" {
		sender, e := reminder.NewFCM(raw)
		if e != nil {
			log.Fatal("Firebase service account configuration is invalid")
		}
		worker.Sender = sender
		config, e := api.ParsePushConfig(os.Getenv("FIREBASE_ANDROID_CONFIG"))
		if e != nil || config == nil || config.ProjectID != sender.ProjectID {
			log.Fatal("Firebase Android configuration is invalid")
		}
		handler.PushConfig = config
	} else {
		log.Print("Firebase push is not configured; reminders remain available in chat")
	}
	go worker.Run(ctx)
	go func() {
		<-ctx.Done()
		timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(timeout)
	}()
	log.Printf("Minder listening on %s (%s)", srv.Addr, mode)
	if e = srv.ListenAndServe(); e != nil && e != http.ErrServerClosed {
		log.Fatal(e)
	}
}

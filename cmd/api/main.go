package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/samluu/waracle-task/internal/config"
	"github.com/samluu/waracle-task/internal/email"
	"github.com/samluu/waracle-task/internal/httpapi"
	"github.com/samluu/waracle-task/internal/store"
)

const (
	emailDelay     = 2 * time.Second // the brief's simulated time to send one email
	emailQueueSize = 100             // confirmations waiting to send before new ones are dropped
	emailWorkers   = 4               // emails sent at once
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	st, err := store.Open(connectCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()

	// Booking confirmations are sent in the background. The LogSender stands in for a mail provider: it
	// takes emailDelay to "send" each one, then logs it.
	emails := email.NewQueue(email.LogSender{Delay: emailDelay, Logger: slog.Default()}, slog.Default(), emailQueueSize, emailWorkers)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewRouter(st, emails),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Listen before serving so a bind failure (e.g. port in use) returns here,
	// before the shutdown goroutine starts and before we log "listening".
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", srv.Addr)
	if err != nil {
		return err
	}

	shutdownDone := make(chan struct{})
	go func() {
		<-ctx.Done()
		slog.Info("shutting down")
		// Under docker's 10s stop grace period, so shutdown finishes before SIGKILL.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutdown", "err", err)
		}
		// Requests have finished, so no more emails can be queued: send the ones waiting, within what's
		// left of the shutdown time. Any still unsent are logged by the queue.
		if err := emails.Close(shutdownCtx); err != nil {
			slog.Error("confirmation emails left unsent at shutdown", "err", err)
		}
		close(shutdownDone)
	}()

	slog.Info("listening", "addr", ln.Addr().String())
	if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	// Serve returns as soon as Shutdown starts; wait for in-flight requests.
	<-shutdownDone
	return nil
}

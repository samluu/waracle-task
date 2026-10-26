// Command devdata seeds the database with test hotels and rooms, or resets it.
//
//	devdata seed
//	devdata reset
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/samluu/waracle-task/internal/store"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(os.Args[1:]); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 || (args[0] != "seed" && args[0] != "reset") {
		return errors.New("usage: devdata seed|reset")
	}
	cmd := args[0]

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	st, err := store.Open(ctx, dbURL)
	if err != nil {
		return err
	}
	defer st.Close()

	switch cmd {
	case "seed":
		err = st.Seed(ctx)
	case "reset":
		err = st.Reset(ctx)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", cmd, err)
	}
	slog.Info("complete", "command", cmd)
	return nil
}

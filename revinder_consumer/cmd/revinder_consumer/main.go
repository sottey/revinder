package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sottey/revinder_consumer/internal/bridge"
	"github.com/sottey/revinder_consumer/internal/consumer"
	"github.com/sottey/revinder_consumer/internal/targets/reminders"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var (
		bridgeURL = flag.String("bridge-url", envDefault("REVINDER_BRIDGE_BASE_URL", "http://127.0.0.1:9120"), "revinder_bridge base URL")
		token     = flag.String("token", envDefault("REVINDER_BRIDGE_TOKEN", envDefault("HOME_TASKS_TOKEN", "")), "revinder_bridge bearer token")
		target    = flag.String("target", "reminders", "target plugin")
		interval  = flag.Duration("interval", 30*time.Second, "poll interval")
		once      = flag.Bool("once", false, "process pending items once and exit")
	)
	flag.Parse()

	if *token == "" {
		return fmt.Errorf("REVINDER_BRIDGE_TOKEN is required")
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	targetPlugin, err := newTarget(*target)
	if err != nil {
		return err
	}
	logger.Info("consumer_started", "bridge_url", *bridgeURL, "target", *target, "interval", interval.String(), "once", *once)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	c := consumer.New(bridge.NewClient(*bridgeURL, *token), targetPlugin, logger)
	if *once {
		if err := c.ProcessOnce(ctx); err != nil {
			return err
		}
		logger.Info("consumer_finished")
		return nil
	}

	return c.Run(ctx, *interval)
}

func newTarget(name string) (consumer.Target, error) {
	switch name {
	case "reminders":
		return reminders.New(), nil
	default:
		return nil, fmt.Errorf("unknown target %q", name)
	}
}

func envDefault(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

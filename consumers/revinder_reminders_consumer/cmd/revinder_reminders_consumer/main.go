package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sottey/revinder/consumers/revinder_reminders_consumer/internal/bridge"
	"github.com/sottey/revinder/consumers/revinder_reminders_consumer/internal/consumer"
	"github.com/sottey/revinder/consumers/revinder_reminders_consumer/internal/reminders"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var (
		configPath = flag.String("config", "", "JSON config file")
		bridgeURL  = flag.String("bridge-url", envDefault("REVINDER_BRIDGE_BASE_URL", "http://127.0.0.1:9120"), "revinder_bridge base URL")
		token      = flag.String("token", envDefault("REVINDER_BRIDGE_TOKEN", envDefault("HOME_TASKS_TOKEN", "")), "revinder_bridge bearer token")
		interval   = flag.Duration("interval", 30*time.Second, "poll interval")
		once       = flag.Bool("once", false, "process pending items once and exit")
	)
	flag.Parse()

	if *configPath != "" {
		cfg, err := loadConfig(*configPath)
		if err != nil {
			return err
		}
		visited := visitedFlags()
		if cfg.BridgeURL != "" && !visited["bridge-url"] {
			*bridgeURL = cfg.BridgeURL
		}
		if cfg.Token != "" && !visited["token"] {
			*token = cfg.Token
		}
		if cfg.IntervalSeconds > 0 && !visited["interval"] {
			*interval = time.Duration(cfg.IntervalSeconds) * time.Second
		}
	}

	if *token == "" {
		return fmt.Errorf("REVINDER_BRIDGE_TOKEN is required")
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("consumer_started", "bridge_url", *bridgeURL, "consumer", "apple_reminders", "interval", interval.String(), "once", *once)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	c := consumer.New(bridge.NewClient(*bridgeURL, *token), reminders.New(), logger)
	if *once {
		if err := c.ProcessOnce(ctx); err != nil {
			return err
		}
		logger.Info("consumer_finished")
		return nil
	}

	return c.Run(ctx, *interval)
}

func envDefault(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

type config struct {
	BridgeURL       string `json:"bridge_url"`
	Token           string `json:"token"`
	IntervalSeconds int    `json:"interval_seconds"`
}

func loadConfig(path string) (config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config{}, err
	}

	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config{}, err
	}
	return cfg, nil
}

func visitedFlags() map[string]bool {
	visited := map[string]bool{}
	flag.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})
	return visited
}

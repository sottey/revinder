/*
Copyright © 2026 sottey

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sottey/revinder_bridge/internal/config"
	"github.com/sottey/revinder_bridge/internal/httpapi"
	"github.com/sottey/revinder_bridge/internal/store"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the revinder_bridge API server",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		token := os.Getenv("HOME_TASKS_TOKEN")
		if token == "" {
			return fmt.Errorf("HOME_TASKS_TOKEN is required")
		}

		db, err := store.Open(cfg.DatabasePath)
		if err != nil {
			return err
		}
		defer db.Close()

		server := &http.Server{
			Addr:         cfg.ServerAddress(),
			Handler:      httpapi.NewRouter(db, token, logger),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		errCh := make(chan error, 1)
		go func() {
			logger.Info("server_started", "address", cfg.ServerAddress(), "database_path", cfg.DatabasePath)
			errCh <- server.ListenAndServe()
		}()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigCh)

		select {
		case err := <-errCh:
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return err
		case sig := <-sigCh:
			logger.Info("server_stopping", "signal", sig.String())
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			return err
		}

		logger.Info("server_stopped")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

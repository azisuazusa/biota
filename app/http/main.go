// Command server runs the BIOTA marketplace sales insight web app.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/azisuazusa/biota/app"
	salesctrl "github.com/azisuazusa/biota/app/http/controller/sales"
	"github.com/azisuazusa/biota/services/sales"
	"github.com/azisuazusa/biota/services/sales/repo_file"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	healthcheck := flag.Bool("healthcheck", false, "GET /healthz on the local server and exit 0 (ok) or 1")
	flag.Parse()

	port := env("PORT", "8080")
	if *healthcheck {
		os.Exit(runHealthcheck(port))
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(port); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func runHealthcheck(port string) int {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/healthz")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "unhealthy:", resp.Status)
		return 1
	}
	return 0
}

func run(port string) error {
	dataDir := env("DATA_DIR", "/data")
	maxMB, err := strconv.Atoi(env("MAX_UPLOAD_MB", "10"))
	if err != nil || maxMB <= 0 {
		return fmt.Errorf("invalid MAX_UPLOAD_MB %q", os.Getenv("MAX_UPLOAD_MB"))
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	svc, err := sales.New(ctx, repo_file.NewDatasetRepo(dataDir), repo_file.NewConfigRepo(dataDir))
	if err != nil {
		return fmt.Errorf("init service: %w", err)
	}
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           app.NewRouter(salesctrl.New(svc, maxMB)),
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", srv.Addr, "data_dir", dataDir, "max_upload_mb", maxMB)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
	}
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

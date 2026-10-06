package main

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

	"github.com/Rombond/tipsarr/backend/internal/api"
	"github.com/Rombond/tipsarr/backend/internal/auth"
	"github.com/Rombond/tipsarr/backend/internal/config"
	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/Rombond/tipsarr/backend/internal/server"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

func main() {
	cfg := config.Load()
	setupLogging(cfg.LogLevel)

	if len(os.Args) > 1 && os.Args[1] == "openapi" {
		if err := printSpec(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := run(cfg); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(cfg config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := os.MkdirAll(cfg.ConfigDir, 0o755); err != nil {
		return err
	}
	st, err := store.Open(ctx, cfg.DBURL)
	if err != nil {
		return err
	}
	defer st.Close()
	slog.Info("database ready", "engine", st.Dialect)

	if cfg.JellyfinURL != "" {
		if cur, _ := st.GetSetting(ctx, auth.SettingJellyfinURL); cur == "" {
			if err := st.SetSetting(ctx, auth.SettingJellyfinURL, cfg.JellyfinURL); err != nil {
				return err
			}
			slog.Info("Jellyfin URL seeded from TIPSARR_JELLYFIN_URL")
		}
	}

	handler, _ := server.New(api.Deps{
		Store: st, Auth: auth.New(st), Media: media.New(st, ""),
		DryRun: cfg.DryRun, ConfigDir: cfg.ConfigDir,
	})
	if cfg.DryRun {
		slog.Warn("DRY-RUN is ON: nothing will be sent to Radarr/Sonarr (set TIPSARR_DRY_RUN=false to disable)")
	}
	srv := &http.Server{Addr: cfg.Addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second}

	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()

	slog.Info("listening", "addr", cfg.Addr, "version", api.Version)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// printSpec writes the OpenAPI spec to stdout without opening a DB.
func printSpec() error {
	_, a := server.New(api.Deps{})
	b, err := a.OpenAPI().MarshalJSON()
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(append(b, '\n'))
	return err
}

func setupLogging(level string) {
	var l slog.Level
	_ = l.UnmarshalText([]byte(level))
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: l})))
}

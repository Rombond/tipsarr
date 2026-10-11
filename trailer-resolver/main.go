// Command trailer-resolver downloads YouTube trailers with yt-dlp into a folder that Tipsarr serves to its apps.
// It is optional: Tipsarr works without it, the Teasers tab is then hidden. See README.md for the YouTube terms.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		os.Exit(healthcheck())
	}
	cfg, err := loadConfig()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	st, err := NewStore(cfg.CacheDir, time.Now)
	if err != nil {
		slog.Error("cache folder", "err", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	w := NewWorker(cfg, st, execRunner{cfg: cfg}, time.Now)
	go w.Run(ctx)
	go housekeeping(ctx, cfg, st)

	srv := &http.Server{Addr: cfg.Addr, Handler: NewServer(cfg, st, w).Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	slog.Info("trailer-resolver listening", "addr", cfg.Addr, "cache", cfg.CacheDir)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}

// housekeeping evicts old videos every 10 minutes and updates yt-dlp once a day (YouTube changes often).
func housekeeping(ctx context.Context, cfg Config, st *Store) {
	evict := time.NewTicker(10 * time.Minute)
	update := time.NewTicker(24 * time.Hour)
	defer evict.Stop()
	defer update.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-evict.C:
			if n := st.Evict(cfg.CacheLimit, cfg.UnusedAfter); n > 0 {
				slog.Info("cache cleaned", "deleted", n)
			}
		case <-update.C:
			if !cfg.AutoUpdate {
				continue
			}
			uctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			out, err := exec.CommandContext(uctx, cfg.YtDlp, "-U").CombinedOutput()
			cancel()
			if err != nil {
				slog.Warn("yt-dlp update failed", "err", oneLine(string(out)))
			}
		}
	}
}

// healthcheck is what the Docker HEALTHCHECK runs: the service answers on its own port.
func healthcheck() int {
	addr := env("RESOLVER_ADDR", ":8090")
	if addr[0] == ':' {
		addr = "127.0.0.1" + addr
	}
	c := http.Client{Timeout: 3 * time.Second}
	resp, err := c.Get("http://" + addr + "/healthz")
	if err != nil || resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

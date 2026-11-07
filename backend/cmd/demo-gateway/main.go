// Command demo-gateway runs the public Tipsarr demo: every visitor gets a throwaway Tipsarr process of their own
// (admin rights included) that is deleted when the session ends. See deployments/demo/README.md.
//
// It runs three listeners:
//
//   - DEMO_LISTEN (public): hands out sessions and proxies each one to its sandbox;
//
//   - DEMO_INTERNAL_ADDR (loopback only): the fake Jellyfin/Radarr/Sonarr and the shared TMDB proxy the sandboxes use;
//
//   - DEMO_ADMIN_LISTEN (private): /healthz and /metrics.
//
//     demo-gateway          serve (builds the template first when needed)
//     demo-gateway seed     only (re)build the template, then exit
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/demo"
)

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if n, err := strconv.Atoi(env(key, "")); err == nil {
		return n
	}
	return def
}

func envDur(key string, def time.Duration) time.Duration {
	if d, err := time.ParseDuration(env(key, "")); err == nil && d > 0 {
		return d
	}
	return def
}

func parseCIDRs(s string) ([]*net.IPNet, error) {
	var out []*net.IPNet
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part == "" {
			continue
		}
		if !strings.Contains(part, "/") {
			if strings.Contains(part, ":") {
				part += "/128"
			} else {
				part += "/32"
			}
		}
		_, n, err := net.ParseCIDR(part)
		if err != nil {
			return nil, fmt.Errorf("DEMO_TRUSTED_PROXIES: %w", err)
		}
		out = append(out, n)
	}
	return out, nil
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	if err := run(len(os.Args) > 1 && os.Args[1] == "seed"); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(seedOnly bool) error {
	data := env("DEMO_DATA_DIR", "./demo-data")
	internal := env("DEMO_INTERNAL_ADDR", "127.0.0.1:18200")
	if host, _, err := net.SplitHostPort(internal); err != nil || !net.ParseIP(host).IsLoopback() {
		return errors.New("DEMO_INTERNAL_ADDR must be a loopback ip:port")
	}
	bin := env("DEMO_TIPSARR_BIN", "tipsarr")
	if p, err := exec.LookPath(bin); err == nil {
		bin = p
	}
	trusted, err := parseCIDRs(env("DEMO_TRUSTED_PROXIES", ""))
	if err != nil {
		return err
	}
	key := os.Getenv("TMDB_API_KEY")
	if key == "" {
		slog.Warn("TMDB_API_KEY is empty: titles, posters and requests will not work")
	}

	sandboxes := filepath.Join(data, "sandboxes")
	// orphans of a previous run (the processes died with it: Pdeathsig)
	if err := os.RemoveAll(sandboxes); err != nil {
		return err
	}
	if err := os.MkdirAll(sandboxes, 0o700); err != nil {
		return err
	}

	tmdbRate, _ := strconv.ParseFloat(env("DEMO_TMDB_RATE", "5"), 64)
	ttl := envDur("DEMO_TMDB_TTL", 24*time.Hour)
	tmdbLimit := demo.NewLimiter(tmdbRate, max(1, int(tmdbRate*2)))
	proxy := demo.NewProxy(filepath.Join(data, "cache"),
		demo.Upstream{Prefix: "/tmdb/", Base: env("DEMO_TMDB_UPSTREAM", "https://api.themoviedb.org"), TTL: ttl, Limiter: tmdbLimit, TMDBKey: key},
		demo.Upstream{Prefix: "/tmdbimg/", Base: env("DEMO_TMDB_IMAGE_UPSTREAM", "https://image.tmdb.org"), TTL: 30 * 24 * time.Hour, Limiter: demo.NewLimiter(20, 40)},
		demo.Upstream{Prefix: "/boxoffice/", Base: env("DEMO_BOXOFFICE_UPSTREAM", "https://www.boxofficemojo.com"), TTL: 12 * time.Hour, Limiter: demo.NewLimiter(0.5, 2)},
	)
	inLn, err := net.Listen("tcp", internal)
	if err != nil {
		return err
	}
	go func() {
		_ = (&http.Server{Handler: demo.Services(time.Now(), proxy), ReadHeaderTimeout: 10 * time.Second}).Serve(inLn)
	}()

	sp := &procSpawner{Bin: bin, Template: filepath.Join(data, "template"), Root: sandboxes, Internal: internal,
		PublicURL: env("DEMO_PUBLIC_URL", ""), MemLimit: env("DEMO_CHILD_MEMORY", "160MiB"), LogLevel: env("DEMO_CHILD_LOG_LEVEL", "warn"),
		Ready: envDur("DEMO_CHILD_READY_TIMEOUT", 30*time.Second)}
	// the template is built by a sandbox without a template
	tpl := sp.Template
	sp.Template = ""
	if err := ensureTemplate(sp, tpl); err != nil {
		return fmt.Errorf("demo template: %w", err)
	}
	sp.Template = tpl
	if seedOnly {
		return nil
	}

	cfg := Config{
		Warm: envInt("DEMO_WARM", 3), MaxSessions: envInt("DEMO_MAX_SESSIONS", 20),
		IdleTimeout: envDur("DEMO_IDLE_TIMEOUT", 30*time.Minute), MaxLifetime: envDur("DEMO_MAX_LIFETIME", 60*time.Minute),
		SessionsPerIP: envInt("DEMO_SESSIONS_PER_IP", 5), IPWindow: envDur("DEMO_IP_WINDOW", time.Hour),
		TrustedProxies: trusted, MaxBody: int64(envInt("DEMO_MAX_BODY_BYTES", 5<<20)), CookieSecure: env("DEMO_COOKIE_SECURE", "false") == "true",
	}
	g := NewGateway(cfg, sp)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go g.Run(10*time.Second, ctx.Done())

	adm := http.NewServeMux()
	adm.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok\n")) })
	adm.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		g.Metrics(w, map[string]int64{"demo_tmdb_cache_hits_total": proxy.Hits.Load(), "demo_tmdb_upstream_fetches_total": proxy.Misses.Load(),
			"demo_tmdb_errors_total": proxy.Errors.Load()})
	})
	admSrv := &http.Server{Addr: env("DEMO_ADMIN_LISTEN", "127.0.0.1:9090"), Handler: adm, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := admSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("admin listener", "err", err)
		}
	}()

	pub := &http.Server{Addr: env("DEMO_LISTEN", ":8080"), Handler: g, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 2 * time.Minute}
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = pub.Shutdown(sctx)
		_ = admSrv.Shutdown(sctx)
	}()
	slog.Info("demo gateway listening", "public", pub.Addr, "admin", admSrv.Addr, "internal", internal, "warm", cfg.Warm, "max", cfg.MaxSessions)
	if err := pub.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	g.Close()
	return nil
}

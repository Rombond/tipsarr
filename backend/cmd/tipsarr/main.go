package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/api"
	"github.com/Rombond/tipsarr/backend/internal/auth"
	"github.com/Rombond/tipsarr/backend/internal/avatars"
	"github.com/Rombond/tipsarr/backend/internal/boxoffice"
	"github.com/Rombond/tipsarr/backend/internal/config"
	"github.com/Rombond/tipsarr/backend/internal/events"
	"github.com/Rombond/tipsarr/backend/internal/i18n"
	"github.com/Rombond/tipsarr/backend/internal/issues"
	"github.com/Rombond/tipsarr/backend/internal/jobs"
	"github.com/Rombond/tipsarr/backend/internal/library"
	"github.com/Rombond/tipsarr/backend/internal/marks"
	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/Rombond/tipsarr/backend/internal/notify"
	"github.com/Rombond/tipsarr/backend/internal/playback"
	"github.com/Rombond/tipsarr/backend/internal/push"
	"github.com/Rombond/tipsarr/backend/internal/ratings"
	"github.com/Rombond/tipsarr/backend/internal/requests"
	"github.com/Rombond/tipsarr/backend/internal/server"
	"github.com/Rombond/tipsarr/backend/internal/stats"
	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/Rombond/tipsarr/backend/internal/suggestions"
	"github.com/Rombond/tipsarr/backend/internal/teasers"
	"github.com/Rombond/tipsarr/backend/internal/trailers"
)

func main() {
	cfg := config.Load()
	setupLogging(cfg.LogLevel)

	if len(os.Args) > 1 && os.Args[1] == "openapi" {
		if err := printSpec(len(os.Args) > 2 && os.Args[2] == "yaml"); err != nil {
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

	if err := dropPrivileges(cfg.ConfigDir); err != nil {
		return err
	}
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

	lib := library.New(st)
	jm := jobs.New(st)
	jm.Register(jobs.Job{Name: "library-sync", Every: 6 * time.Hour, InitialDelay: 10 * time.Second, Run: func(ctx context.Context) (i18n.Msg, error) {
		if !lib.Configured(ctx) {
			return i18n.M("srv.job.skip.no_jellyfin_key"), jobs.ErrSkipped
		}
		r, err := lib.SyncLibrary(ctx)
		return r.Msg(), err
	}})
	jm.Register(jobs.Job{Name: "history-sync", Every: time.Hour, InitialDelay: 40 * time.Second, Run: func(ctx context.Context) (i18n.Msg, error) {
		if !lib.Configured(ctx) {
			return i18n.M("srv.job.skip.no_jellyfin_key"), jobs.ErrSkipped
		}
		r, err := lib.SyncHistory(ctx, "")
		return r.Msg(), err
	}})
	hub := events.New()
	jm.OnChange = func(name, status string, msg i18n.Msg, text string) {
		hub.Publish("sync.status", "", map[string]any{"job": name, "state": status, "message": text, "messageCode": msg.Code, "messageParams": msg.StringParams(), "messageArgs": msg.OrderedArgs()}, true)
	}
	// TIPSARR_TMDB_URL, TIPSARR_TMDB_IMAGE_URL and TIPSARR_BOXOFFICE_URL point the server at fakes (see dev/README.md); unset = the real services.
	mediaSvc := media.New(st, os.Getenv("TIPSARR_TMDB_URL"))
	notifier := notify.New(st, cfg.DryRun)
	pusher := push.New(st, cfg.DryRun)
	notifier.SetPusher(pusher)
	sugg := suggestions.New(st, mediaSvc, hub)
	// Trailer files come from the optional trailer-resolver sidecar, which writes them into a shared folder that is mounted
	// read-only here (TIPSARR_TRAILERS_DIR, default <config>/trailers). The feature is off until an admin turns it on.
	trailerDir := os.Getenv("TIPSARR_TRAILERS_DIR")
	if trailerDir == "" {
		trailerDir = filepath.Join(cfg.ConfigDir, "trailers")
	}
	trailerFiles := trailers.New(st, trailerDir)
	jm.Register(jobs.Job{Name: "trailer-files-sync", Every: time.Minute, InitialDelay: 45 * time.Second, Quiet: true, Run: trailerFiles.Sync})
	lib.OnHistoryChanged = sugg.QueueRefresh
	box := boxoffice.New(st, mediaSvc, os.Getenv("TIPSARR_BOXOFFICE_URL"))
	jm.Register(jobs.Job{Name: "boxoffice-refresh", Every: 12 * time.Hour, InitialDelay: time.Minute, Run: box.Refresh})
	reqSvc := requests.New(st, mediaSvc, hub, notifier, cfg.DryRun)
	reqSvc.SetLibrary(lib)
	jm.Register(jobs.Job{Name: "request-poll", Every: 15 * time.Second, InitialDelay: 20 * time.Second, Quiet: true, Run: func(ctx context.Context) (i18n.Msg, error) {
		n, err := reqSvc.Poll(ctx)
		return i18n.M("srv.job.poll.inflight", "count", n), err
	}})

	jm.Register(jobs.Job{Name: "servarr-import", Every: time.Hour, InitialDelay: 90 * time.Second, Run: reqSvc.ImportFromServarr})
	jm.Register(jobs.Job{Name: "housekeeping", Every: time.Hour, InitialDelay: 2 * time.Minute, Quiet: true, Run: func(ctx context.Context) (i18n.Msg, error) {
		if err := st.PurgeExpiredSessions(ctx); err != nil {
			return i18n.Msg{}, err
		}
		return i18n.M("srv.job.housekeeping.purged"), st.PurgeExpiredCache(ctx)
	}})
	pb := playback.New(st, mediaSvc)
	jm.Register(jobs.Job{Name: "playback-sync", Every: time.Hour, InitialDelay: 2*time.Minute + 30*time.Second, Run: func(ctx context.Context) (i18n.Msg, error) {
		msg, err := pb.Sync(ctx)
		switch {
		case errors.Is(err, playback.ErrNotInstalled):
			return i18n.M("srv.job.skip.playback_missing"), jobs.ErrSkipped
		case errors.Is(err, playback.ErrNoAPIKey):
			return i18n.M("srv.job.skip.no_jellyfin_key"), jobs.ErrSkipped
		}
		return msg, err
	}})
	jm.Start(ctx)

	// Optional (TIPSARR_SETUP_TOKEN=true): a fresh install must be claimed with a one-time token
	// that only appears in this log, so a stranger who reaches the port first cannot point Tipsarr
	// at their own "Jellyfin". Off by default: on a private network it is just friction.
	setupToken := ""
	if cur, _ := st.GetSetting(ctx, auth.SettingJellyfinURL); cur == "" && cfg.RequireSetupToken {
		raw := make([]byte, 12)
		if _, err := rand.Read(raw); err != nil {
			return err
		}
		setupToken = hex.EncodeToString(raw)
		slog.Warn("first run: open Tipsarr in a browser and enter this setup token", "token", setupToken)
	}

	handler, _ := server.New(api.Deps{
		Store: st, Stats: stats.New(st), Ratings: ratings.New(st), Auth: auth.New(st), Media: mediaSvc, Library: lib, Jobs: jm, Requests: reqSvc, Issues: issues.New(st, mediaSvc, hub, notifier), Avatars: avatars.New(cfg.ConfigDir, st), Suggestions: sugg, Teasers: teasers.New(st, mediaSvc, sugg, trailerFiles), Trailers: trailerFiles, BoxOffice: box, Marks: marks.New(st, mediaSvc), LoginLimiter: auth.NewLimiter(8, 10*time.Minute), SetupToken: setupToken, SecureCookies: cfg.SecureCookies, Hub: hub, Notify: notifier, Push: pusher,
		DryRun: cfg.DryRun, ConfigDir: cfg.ConfigDir, ImageBaseURL: os.Getenv("TIPSARR_TMDB_IMAGE_URL"),
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
// printSpec writes the OpenAPI spec: JSON 3.1 for the web client, or (yaml) the OpenAPI 3.0.3 YAML
// that the mobile client generators read, which every generator supports.
func printSpec(yaml bool) error {
	_, a := server.New(api.Deps{})
	var b []byte
	var err error
	if yaml {
		b, err = a.OpenAPI().DowngradeYAML()
	} else {
		b, err = a.OpenAPI().MarshalJSON()
	}
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

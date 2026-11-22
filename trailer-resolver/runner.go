package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Result describes a finished video.
type Result struct {
	Height   int
	Duration int
}

// Runner downloads one trailer to dst (an .mp4). Tests replace it.
type Runner interface {
	Download(ctx context.Context, key string, maxHeight int, dst string) (Result, error)
}

// DownloadError says whether trying again later can help.
type DownloadError struct {
	Msg       string
	Permanent bool
	// Video marks a problem with this video only (a retry may help): it does not count toward the circuit breaker,
	// which is about the server's address being blocked.
	Video bool
	// RetryIn is how long a Permanent failure stands before the key may be tried again (0: the default, 7 days).
	RetryIn time.Duration
}

func (e *DownloadError) Error() string { return e.Msg }

// execRunner runs yt-dlp, then ffmpeg to move the index to the start of the file so playback begins at once.
type execRunner struct {
	cfg Config
}

func (r execRunner) Download(ctx context.Context, key string, maxHeight int, dst string) (Result, error) {
	work, err := os.MkdirTemp(filepath.Dir(dst), ".work-"+key+"-")
	if err != nil {
		return Result{}, err
	}
	defer func() {
		if err := os.RemoveAll(work); err != nil {
			slog.Warn("could not remove work folder (cleaned up later)", "dir", work, "err", err)
		}
	}()

	h := strconv.Itoa(maxHeight)
	// H.264 + AAC plays on every phone; fall back to anything that fits the height.
	format := "bv*[height<=" + h + "][vcodec^=avc1]+ba[acodec^=mp4a]/b[height<=" + h + "][vcodec^=avc1]/bv*[height<=" + h + "]+ba/b[height<=" + h + "]"
	args := []string{
		"--no-playlist", "--no-warnings", "--no-progress", "--no-colors", "--no-mtime", "--ignore-config",
		"--socket-timeout", "20", "--retries", "2", "--fragment-retries", "2",
		"--max-filesize", "80M",
		// A trailer is short; a long film is not one. Videos with an unknown length pass.
		"--match-filters", "duration<=?" + strconv.Itoa(int(r.cfg.MaxDuration.Seconds())),
		"--js-runtimes", "deno:" + r.cfg.Deno,
	}
	if strings.Contains(r.cfg.FFmpeg, "/") { // a full path; otherwise yt-dlp finds ffmpeg on PATH
		args = append(args, "--ffmpeg-location", filepath.Dir(r.cfg.FFmpeg))
	}
	args = append(args,
		"-f", format, "--merge-output-format", "mp4",
		"-o", filepath.Join(work, "in.%(ext)s"),
	)
	stderr, err := r.ytdlp(ctx, args, work, key)
	if err != nil && loginWall(stderr) && ctx.Err() == nil {
		// Some videos only show a login wall to the default clients; the embed client plays them like the embedded player does.
		slog.Info("login wall, retrying with the embed client", "key", key)
		stderr, err = r.ytdlp(ctx, append(args, "--extractor-args", "youtube:player_client=web_embedded"), work, key)
		if err != nil && loginWall(stderr) {
			// embedding disabled or really gated: nothing else will work without an account
			return Result{}, &DownloadError{Msg: trim(oneLine(stderr), 240), Permanent: true}
		}
	}
	if err != nil {
		if ctx.Err() != nil {
			return Result{}, &DownloadError{Msg: "timeout", Video: true} // a big or slow video, not a blocked address
		}
		return Result{}, classify(stderr, err)
	}
	in := filepath.Join(work, "in.mp4")
	if st, err := os.Stat(in); err != nil || st.Size() == 0 {
		// yt-dlp exits 0 without a file when it skips a video: over the length limit (never a trailer, final) or over --max-filesize.
		if strings.Contains(stderr, "does not pass filter") {
			return Result{}, &DownloadError{Msg: "longer than " + r.cfg.MaxDuration.String() + ", not a trailer", Permanent: true}
		}
		// over --max-filesize at this height: the same thing happens on every try
		return Result{}, &DownloadError{Msg: "yt-dlp produced no mp4 file (probably over the size limit)", Permanent: true}
	}
	out := filepath.Join(work, "out.mp4")
	if b, err := exec.CommandContext(ctx, r.cfg.FFmpeg, "-y", "-v", "error", "-i", in, "-c", "copy", "-movflags", "+faststart", out).CombinedOutput(); err != nil {
		return Result{}, &DownloadError{Msg: "ffmpeg: " + oneLine(string(b)), Video: true}
	}
	res := r.probe(ctx, out)
	if err := os.Rename(out, dst); err != nil {
		return Result{}, err
	}
	return res, nil
}

// ytdlp runs yt-dlp once and returns what it wrote (stderr and stdout together).
func (r execRunner) ytdlp(ctx context.Context, args []string, work, key string) (string, error) {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, r.cfg.YtDlp, append(args, "https://www.youtube.com/watch?v="+key)...)
	cmd.Stderr = &stderr
	cmd.Stdout = &stderr // the "does not pass filter" notice is printed on stdout
	cmd.Env = append(os.Environ(), "TMPDIR="+work)
	err := cmd.Run()
	return stderr.String(), err
}

// loginWall reports yt-dlp's "Please sign in" error (not the "confirm you're not a bot" one, which is about the server's address).
func loginWall(stderr string) bool {
	low := strings.ToLower(stderr)
	return strings.Contains(low, "please sign in") || strings.Contains(low, "login required")
}

func (r execRunner) probe(ctx context.Context, file string) Result {
	out, err := exec.CommandContext(ctx, r.cfg.FFprobe, "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=height:format=duration", "-of", "json", file).Output()
	if err != nil {
		return Result{}
	}
	var p struct {
		Streams []struct {
			Height int `json:"height"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if json.Unmarshal(out, &p) != nil {
		return Result{}
	}
	res := Result{}
	if len(p.Streams) > 0 {
		res.Height = p.Streams[0].Height
	}
	if d, err := strconv.ParseFloat(p.Format.Duration, 64); err == nil {
		res.Duration = int(d + 0.5)
	}
	return res
}

// temporaryHints are about the server's address (rate limit, bot check). They are checked first: YouTube can join
// them to a "Video unavailable" reason, and they must never mark a key as final.
var temporaryHints = []string{
	"try again later", "rate-limit", "rate limit", "not a bot", "captcha", "ip is likely being blocked", "http error 429",
}

// laterHints are videos that do not exist yet (a scheduled premiere or live): final for a few hours only.
var laterHints = []string{
	"premieres in", "premiere will begin", "live event will begin", "this live event will", "is upcoming",
}

// laterRetry is how long a not-yet-published video waits before it is tried again.
const laterRetry = 6 * time.Hour

// permanentHints are yt-dlp messages that no retry will fix (for a while: final failures still expire).
// "Requested format is not available" is not one: it usually means yt-dlp could not solve YouTube's challenge
// (broken JS runtime, outdated solver, missing PO token), which is the server's problem, not the video's.
var permanentHints = []string{
	"video unavailable", "video is unavailable", "this video is not available", "private video", "has been removed", "who has blocked it",
	"copyright claim", "has been terminated", "members-only", "join this channel", "this video is no longer available",
	"available in your country", "age-restricted", "confirm your age", "is not a valid url", "unsupported url",
}

func classify(stderr string, err error) error {
	msg := oneLine(stderr)
	if msg == "" {
		msg = err.Error()
	}
	low := strings.ToLower(msg)
	if containsAny(low, temporaryHints) {
		return &DownloadError{Msg: trim(msg, 240)}
	}
	if containsAny(low, laterHints) {
		return &DownloadError{Msg: trim(msg, 240), Permanent: true, RetryIn: laterRetry}
	}
	if containsAny(low, permanentHints) {
		return &DownloadError{Msg: trim(msg, 240), Permanent: true}
	}
	return &DownloadError{Msg: trim(msg, 240)}
}

func containsAny(s string, hints []string) bool {
	for _, h := range hints {
		if strings.Contains(s, h) {
			return true
		}
	}
	return false
}

func oneLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	// the last ERROR line is the useful one
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], "ERROR") {
			return strings.TrimSpace(lines[i])
		}
	}
	if len(lines) > 0 {
		return strings.TrimSpace(lines[len(lines)-1])
	}
	return ""
}

func trim(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// Versions keeps the tool versions, read at start and after each yt-dlp update, so a health check runs no program.
type Versions struct {
	cfg Config

	refreshMu           sync.Mutex
	mu                  sync.Mutex
	loaded              bool
	ytdlp, deno, ffmpeg string
}

func NewVersions(cfg Config) *Versions { return &Versions{cfg: cfg} }

// Refresh reads the versions again. It never uses a caller's context: a health request that gives up must not
// leave empty versions behind. A tool that answered before and fails now keeps its old version (a broken yt-dlp
// then shows up as download failures and the breaker, not in the version). One read runs at a time.
func (v *Versions) Refresh() {
	v.refreshMu.Lock()
	defer v.refreshMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	yt, deno, ff := version(ctx, v.cfg.YtDlp, "--version"), version(ctx, v.cfg.Deno, "--version"), version(ctx, v.cfg.FFmpeg, "-version")
	v.mu.Lock()
	defer v.mu.Unlock()
	keep := func(old *string, now string) {
		if now != "" {
			*old = now
		}
	}
	keep(&v.ytdlp, yt)
	keep(&v.deno, deno)
	keep(&v.ffmpeg, ff)
	if yt != "" {
		v.loaded = true // until yt-dlp answers once, Get tries again
	}
}

// Get returns yt-dlp, Deno and ffmpeg versions ("" for a tool that does not run), reading them until yt-dlp answers.
func (v *Versions) Get() (ytdlp, deno, ffmpeg string) {
	v.mu.Lock()
	loaded := v.loaded
	v.mu.Unlock()
	if !loaded {
		v.Refresh()
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.ytdlp, v.deno, v.ffmpeg
}

// version runs "<bin> <arg>" and returns the first line, or "" if it fails.
func version(ctx context.Context, bin string, arg string) string {
	out, err := exec.CommandContext(ctx, bin, arg).Output()
	if err != nil {
		return ""
	}
	line, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	return trim(line, 120)
}

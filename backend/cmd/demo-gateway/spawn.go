package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/demo"
)

// procSpawner starts real Tipsarr processes in demo mode.
type procSpawner struct {
	Bin       string // tipsarr binary
	Template  string // template data directory (copied for each sandbox)
	Root      string // parent of the sandbox data directories
	Internal  string // the gateway's internal listener, host:port
	PublicURL string
	MemLimit  string // GOMEMLIMIT of each sandbox
	LogLevel  string
	Ready     time.Duration
	seq       atomic.Int64
}

// env is the whole environment of a sandbox: nothing is inherited, so the TMDB key never reaches it.
func (p *procSpawner) env(dir, addr string) []string {
	base := "http://" + p.Internal
	return []string{
		"PATH=/usr/local/bin:/usr/bin:/bin", "HOME=" + dir, "TZ=" + os.Getenv("TZ"),
		"GOMEMLIMIT=" + p.MemLimit, "GOMAXPROCS=1",
		"TIPSARR_DEMO_MODE=true", "TIPSARR_DEMO_ALLOW_DIAL=" + p.Internal, "TIPSARR_DEMO_PUBLIC_URL=" + p.PublicURL,
		"TIPSARR_ADDR=" + addr, "TIPSARR_CONFIG_DIR=" + dir, "TIPSARR_DRY_RUN=true", "TIPSARR_LOG_LEVEL=" + p.LogLevel,
		"TIPSARR_JELLYFIN_URL=" + base + demo.PathJellyfin,
		"TIPSARR_TMDB_URL=" + base + demo.PathTMDB,
		"TIPSARR_TMDB_IMAGE_URL=" + base + demo.PathTMDBImage,
		"TIPSARR_BOXOFFICE_URL=" + base + demo.PathBoxOffice,
	}
}

func freePort() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer l.Close()
	return l.Addr().String(), nil
}

// Spawn copies the template and starts a sandbox on it.
func (p *procSpawner) Spawn() (*Child, error) {
	dir := filepath.Join(p.Root, fmt.Sprintf("sb-%d-%d", os.Getpid(), p.seq.Add(1)))
	if p.Template != "" {
		if err := copyDir(p.Template, dir); err != nil {
			os.RemoveAll(dir)
			return nil, fmt.Errorf("copy template: %w", err)
		}
	} else if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return p.start(dir)
}

func (p *procSpawner) start(dir string) (*Child, error) {
	addr, err := freePort()
	if err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	cmd := exec.Command(p.Bin)
	cmd.Dir, cmd.Env = dir, p.env(dir, addr)
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	cmd.SysProcAttr = childAttrs()
	if err := cmd.Start(); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	c := &Child{Addr: addr, Dir: dir, Done: done}
	c.kill = func() {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
	}
	c.stop = func() {
		c.kill()
		os.RemoveAll(dir)
	}
	if err := waitReady(addr, done, p.Ready); err != nil {
		c.Stop()
		return nil, err
	}
	return c, nil
}

func waitReady(addr string, done <-chan struct{}, max time.Duration) error {
	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(max)
	for time.Now().Before(deadline) {
		select {
		case <-done:
			return errors.New("sandbox exited while starting")
		default:
		}
		if resp, err := client.Get("http://" + addr + "/api/v1/health"); err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return errors.New("sandbox not ready in " + strconv.Itoa(int(max.Seconds())) + "s")
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	})
}

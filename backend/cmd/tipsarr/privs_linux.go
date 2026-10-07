package main

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

// defaultID is the user the container runs as when it starts as root and PUID/PGID are not set.
const defaultID = 65532

// dropPrivileges lets the container follow the PUID/PGID convention of the *arr images: started
// as root it makes the config folder belong to PUID:PGID, then continues as that user. Started as
// a normal user (docker `user:`) it does nothing, and the folder must already be writable.
func dropPrivileges(configDir string) error {
	if os.Getuid() != 0 {
		return nil
	}
	uid, err := idFromEnv("PUID")
	if err != nil {
		return err
	}
	gid, err := idFromEnv("PGID")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	if uid == 0 && gid == 0 {
		slog.Warn("running as root (PUID=0): set PUID/PGID to a normal user")
		return nil
	}
	// only walk the tree when ownership actually changes (the image cache can be large)
	if fi, err := os.Stat(configDir); err == nil {
		if st, ok := fi.Sys().(*syscall.Stat_t); !ok || int(st.Uid) != uid || int(st.Gid) != gid {
			err := filepath.WalkDir(configDir, func(p string, _ fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				return os.Lchown(p, uid, gid)
			})
			if err != nil {
				return fmt.Errorf("give %s to %d:%d: %w", configDir, uid, gid, err)
			}
		}
	}
	if err := syscall.Setgroups([]int{}); err != nil {
		return err
	}
	if err := syscall.Setgid(gid); err != nil {
		return err
	}
	if err := syscall.Setuid(uid); err != nil {
		return err
	}
	slog.Info("running as", "uid", uid, "gid", gid)
	return nil
}

func idFromEnv(name string) (int, error) {
	v := os.Getenv(name)
	if v == "" {
		return defaultID, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("%s must be a number, got %q", name, v)
	}
	return n, nil
}

// Package avatars serves user pictures from three places, in this order: a picture the person
// uploaded in Tipsarr, the jpegPhoto attribute of their LDAP (LLDAP) account, and finally their
// Jellyfin picture (handled by the caller).
package avatars

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"image"
	_ "image/gif" // decoders for uploads
	"image/jpeg"
	_ "image/png"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/go-ldap/ldap/v3"
	"golang.org/x/image/draw"
)

// Setting keys of the optional LDAP source.
const (
	SettingLDAPURL      = "ldap.url"
	SettingLDAPBindDN   = "ldap.bind_dn"
	SettingLDAPPassword = "ldap.bind_password"
	SettingLDAPBaseDN   = "ldap.base_dn"
)

const (
	size       = 256
	maxUpload  = 4 << 20
	ldapMaxAge = 12 * time.Hour
	retryAfter = time.Hour
)

var (
	ErrTooLarge = errors.New("the picture is larger than 4 MB")
	ErrNotImage = errors.New("not a JPEG, PNG or GIF picture")
	idRe        = regexp.MustCompile(`^[A-Za-z0-9-]{8,64}$`)
)

type Service struct {
	dir   string
	store *store.Store

	mu     sync.Mutex
	failed map[string]time.Time // LDAP lookups that found nothing recently
}

func New(configDir string, s *store.Store) *Service {
	return &Service{dir: filepath.Join(configDir, "avatars"), store: s, failed: map[string]time.Time{}}
}

func (s *Service) upload(id string) string   { return filepath.Join(s.dir, id+".jpg") }
func (s *Service) ldapFile(id string) string { return filepath.Join(s.dir, "ldap-"+id+".jpg") }

// Path returns the file to serve for a user, or "" when Jellyfin's picture should be used.
func (s *Service) Path(ctx context.Context, id, username string) string {
	if !idRe.MatchString(id) {
		return ""
	}
	if _, err := os.Stat(s.upload(id)); err == nil {
		return s.upload(id)
	}
	s.refreshLDAP(ctx, id, username)
	if _, err := os.Stat(s.ldapFile(id)); err == nil {
		return s.ldapFile(id)
	}
	return ""
}

// Save stores an uploaded picture, cropped to a square and shrunk to 256 px.
func (s *Service) Save(id string, r io.Reader) error {
	if !idRe.MatchString(id) {
		return ErrNotImage
	}
	raw, err := io.ReadAll(io.LimitReader(r, maxUpload+1))
	if err != nil {
		return err
	}
	if len(raw) > maxUpload {
		return ErrTooLarge
	}
	out, err := normalise(raw)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	return writeAtomic(s.upload(id), out)
}

// Delete removes the uploaded picture (the LDAP or Jellyfin one shows again).
func (s *Service) Delete(id string) error {
	if !idRe.MatchString(id) {
		return nil
	}
	err := os.Remove(s.upload(id))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// HasUpload reports whether the person uploaded their own picture.
func (s *Service) HasUpload(id string) bool {
	_, err := os.Stat(s.upload(id))
	return idRe.MatchString(id) && err == nil
}

func normalise(raw []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, ErrNotImage
	}
	b := img.Bounds()
	side := min(b.Dx(), b.Dy())
	if side <= 0 {
		return nil, ErrNotImage
	}
	crop := image.Rect(b.Min.X+(b.Dx()-side)/2, b.Min.Y+(b.Dy()-side)/2, b.Min.X+(b.Dx()-side)/2+side, b.Min.Y+(b.Dy()-side)/2+side)
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, crop, draw.Over, nil)
	// the square is centred, so turning it after shrinking gives the same picture as turning the original
	upright := orient(dst, exifOrientation(raw))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, upright, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".av-*")
	if err != nil {
		return err
	}
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if werr != nil || cerr != nil {
		os.Remove(tmp.Name())
		return errors.Join(werr, cerr)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Chmod(path, 0o644)
}

// ---- LDAP (LLDAP) ------------------------------------------------------------------------------

type ldapConfig struct{ url, bindDN, password, baseDN string }

func (s *Service) ldapConfig(ctx context.Context) (ldapConfig, bool) {
	get := func(k string) string { v, _ := s.store.GetSetting(ctx, k); return strings.TrimSpace(v) }
	c := ldapConfig{get(SettingLDAPURL), get(SettingLDAPBindDN), get(SettingLDAPPassword), get(SettingLDAPBaseDN)}
	return c, c.url != "" && c.bindDN != "" && c.baseDN != ""
}

// Configured reports whether the LDAP source is set up.
func (s *Service) Configured(ctx context.Context) bool { _, ok := s.ldapConfig(ctx); return ok }

func (s *Service) refreshLDAP(ctx context.Context, id, username string) {
	cfg, ok := s.ldapConfig(ctx)
	if !ok || username == "" {
		return
	}
	if fi, err := os.Stat(s.ldapFile(id)); err == nil && time.Since(fi.ModTime()) < ldapMaxAge {
		return
	}
	s.mu.Lock()
	if t, bad := s.failed[id]; bad && time.Since(t) < retryAfter {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	photo, err := fetchPhoto(ctx, cfg, username)
	if err == nil && len(photo) > 0 {
		if out, nerr := normalise(photo); nerr == nil {
			if os.MkdirAll(s.dir, 0o755) == nil && writeAtomic(s.ldapFile(id), out) == nil {
				return
			}
		}
	}
	// no photo (or LDAP unreachable): keep an older copy, and do not ask again for an hour
	s.mu.Lock()
	s.failed[id] = time.Now()
	s.mu.Unlock()
}

// CheckLDAP binds with the given settings and reports a problem, if any (used when saving).
func CheckLDAP(ctx context.Context, rawURL, bindDN, password string) error {
	conn, err := dial(ctx, rawURL)
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Bind(bindDN, password)
}

func dial(ctx context.Context, rawURL string) (*ldap.Conn, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "ldap" && u.Scheme != "ldaps") || u.Host == "" {
		return nil, fmt.Errorf("the LDAP address must look like ldap://host:3890")
	}
	return ldap.DialURL(rawURL, ldap.DialWithDialer(&netDialer), ldap.DialWithTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12}))
}

func fetchPhoto(ctx context.Context, c ldapConfig, username string) ([]byte, error) {
	conn, err := dial(ctx, c.url)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if err := conn.Bind(c.bindDN, c.password); err != nil {
		return nil, err
	}
	res, err := conn.Search(ldap.NewSearchRequest(c.baseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, 5, false,
		fmt.Sprintf("(&(objectClass=person)(uid=%s))", ldap.EscapeFilter(username)), []string{"jpegPhoto"}, nil))
	if err != nil {
		return nil, err
	}
	if len(res.Entries) == 0 {
		return nil, nil
	}
	return res.Entries[0].GetRawAttributeValue("jpegPhoto"), nil
}

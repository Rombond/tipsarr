package store

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/uptrace/bun"
)

// Secrets at rest: credentials (Radarr/Sonarr and Jellyfin API keys, TMDB key, OIDC client secret, LDAP bind
// password, webhook secrets and auth, relay keys, dashboard API key) are stored AES-256-GCM encrypted, as
// "enc:v1:" + base64(nonce || ciphertext). Values without the prefix are legacy plaintext: they are still read, and
// EnableSecrets rewrites them encrypted once.

const (
	secretPrefix = "enc:v1:"
	// SecretKeyEnv holds the base64 encoded 32 byte key. Without it the key is read from (or generated in) SecretKeyFile.
	SecretKeyEnv = "TIPSARR_SECRET_KEY"
	// SecretKeyFile is the key file name in the config folder.
	SecretKeyFile = "secret.key"
	// settingSecretCheck holds an encrypted known value, so a wrong key is detected at start even with no secret saved.
	settingSecretCheck = "secrets.check"
	secretCheckValue   = "tipsarr"
)

// secretSettings are the settings keys stored encrypted.
var secretSettings = map[string]bool{
	"jellyfin.api_key":      true,
	"tmdb.api_key":          true,
	"oidc.client_secret":    true,
	"ldap.bind_password":    true,
	"push.relay_key":        true,
	"teasers.files.key":     true,
	"api.key":               true,
	"hooks.jellyfin_secret": true,
	settingSecretCheck:      true,
}

// ErrSecretKey means a stored secret cannot be decrypted with the configured key.
var ErrSecretKey = errors.New("stored secrets cannot be decrypted with this key: restore the " + SecretKeyFile +
	" file that was next to the database (or the previous " + SecretKeyEnv + "), or delete the secrets and enter them again")

type secretBox struct{ aead cipher.AEAD }

func newSecretBox(key []byte) (*secretBox, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("secret key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &secretBox{aead: aead}, nil
}

func (b *secretBox) seal(plain string) (string, error) {
	if b == nil || plain == "" || strings.HasPrefix(plain, secretPrefix) {
		return plain, nil
	}
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	out := b.aead.Seal(nonce, nonce, []byte(plain), nil)
	return secretPrefix + base64.RawStdEncoding.EncodeToString(out), nil
}

func (b *secretBox) open(v string) (string, error) {
	if !strings.HasPrefix(v, secretPrefix) {
		return v, nil // legacy plaintext
	}
	if b == nil {
		return "", ErrSecretKey
	}
	raw, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(v, secretPrefix))
	if err != nil || len(raw) < b.aead.NonceSize() {
		return "", ErrSecretKey
	}
	n := b.aead.NonceSize()
	plain, err := b.aead.Open(nil, raw[:n], raw[n:], nil)
	if err != nil {
		return "", ErrSecretKey
	}
	return string(plain), nil
}

// LoadSecretKey returns the key from TIPSARR_SECRET_KEY, or from <configDir>/secret.key, generating that file (0600)
// on first start.
func LoadSecretKey(configDir string) ([]byte, error) {
	if v := strings.TrimSpace(os.Getenv(SecretKeyEnv)); v != "" {
		k, err := base64.StdEncoding.DecodeString(v)
		if err != nil || len(k) != 32 {
			return nil, fmt.Errorf("%s must be 32 bytes encoded in base64 (openssl rand -base64 32)", SecretKeyEnv)
		}
		return k, nil
	}
	path := filepath.Join(configDir, SecretKeyFile)
	if b, err := os.ReadFile(path); err == nil {
		k, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
		if err != nil || len(k) != 32 {
			return nil, fmt.Errorf("%s is not a valid key (32 bytes in base64)", path)
		}
		return k, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	k := make([]byte, 32)
	if _, err := rand.Read(k); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create %s: %w", path, err)
	}
	if _, err := f.WriteString(base64.StdEncoding.EncodeToString(k) + "\n"); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	slog.Warn("generated the secrets encryption key: back it up with the database, without it the stored secrets must be entered again", "file", path)
	return k, nil
}

// EnableSecrets turns on encryption at rest with key: it checks the key against the stored secrets (ErrSecretKey when
// it does not match) and encrypts the values still stored in plaintext. Without it the store keeps plaintext (tests).
func (s *Store) EnableSecrets(ctx context.Context, key []byte) error {
	box, err := newSecretBox(key)
	if err != nil {
		return err
	}
	return s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var settings []Setting
		if err := tx.NewSelect().Model(&settings).Where("skey IN (?)", bun.In(secretSettingKeys())).Scan(ctx); err != nil {
			return err
		}
		hasCheck := false
		for _, st := range settings {
			plain, err := box.open(st.Value)
			if err != nil {
				return fmt.Errorf("setting %s: %w", st.Key, err)
			}
			if st.Key == settingSecretCheck {
				if plain != secretCheckValue {
					return ErrSecretKey
				}
				hasCheck = true
				continue
			}
			if err := reseal(ctx, tx, box, (*Setting)(nil), "svalue", st.Value, "skey = ?", st.Key); err != nil {
				return err
			}
		}
		if !hasCheck {
			v, err := box.seal(secretCheckValue)
			if err != nil {
				return err
			}
			if _, err := tx.NewInsert().Model(&Setting{Key: settingSecretCheck, Value: v}).Exec(ctx); err != nil {
				return err
			}
		}

		var insts []ServarrInstance
		if err := tx.NewSelect().Model(&insts).Scan(ctx); err != nil {
			return err
		}
		for _, in := range insts {
			if _, err := box.open(in.APIKey); err != nil {
				return fmt.Errorf("servarr %s: %w", in.Name, err)
			}
			if err := reseal(ctx, tx, box, (*ServarrInstance)(nil), "api_key", in.APIKey, "id = ?", in.ID); err != nil {
				return err
			}
		}

		var hooks []Webhook
		if err := tx.NewSelect().Model(&hooks).Scan(ctx); err != nil {
			return err
		}
		for _, w := range hooks {
			for col, v := range map[string]string{"secret": w.Secret, "headers": w.Headers, "auth_secret": w.AuthSecret} {
				if _, err := box.open(v); err != nil {
					return fmt.Errorf("webhook %s: %w", w.Name, err)
				}
				if err := reseal(ctx, tx, box, (*Webhook)(nil), col, v, "id = ?", w.ID); err != nil {
					return err
				}
			}
		}
		s.secrets = box
		return nil
	})
}

// reseal rewrites a plaintext value encrypted (no-op when empty or already encrypted).
func reseal(ctx context.Context, tx bun.Tx, box *secretBox, model any, col, v, where string, arg any) error {
	if v == "" || strings.HasPrefix(v, secretPrefix) {
		return nil
	}
	enc, err := box.seal(v)
	if err != nil {
		return err
	}
	_, err = tx.NewUpdate().Model(model).Set(col+" = ?", enc).Where(where, arg).Exec(ctx)
	return err
}

func secretSettingKeys() []string {
	out := make([]string, 0, len(secretSettings))
	for k := range secretSettings {
		out = append(out, k)
	}
	return out
}

// sealServarr / openServarr and sealWebhook / openWebhook convert the secret columns of a row.
func (s *Store) sealServarr(in *ServarrInstance) (*ServarrInstance, error) {
	row := *in
	var err error
	row.APIKey, err = s.secrets.seal(in.APIKey)
	return &row, err
}

func (s *Store) openServarr(in *ServarrInstance) error {
	var err error
	in.APIKey, err = s.secrets.open(in.APIKey)
	return err
}

func (s *Store) sealWebhook(w *Webhook) (*Webhook, error) {
	row := *w
	var err error
	if row.Secret, err = s.secrets.seal(w.Secret); err != nil {
		return nil, err
	}
	if row.Headers, err = s.secrets.seal(w.Headers); err != nil {
		return nil, err
	}
	row.AuthSecret, err = s.secrets.seal(w.AuthSecret)
	return &row, err
}

func (s *Store) openWebhook(w *Webhook) error {
	var err error
	if w.Secret, err = s.secrets.open(w.Secret); err != nil {
		return err
	}
	if w.Headers, err = s.secrets.open(w.Headers); err != nil {
		return err
	}
	w.AuthSecret, err = s.secrets.open(w.AuthSecret)
	return err
}

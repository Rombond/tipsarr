package store

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testKey(b byte) []byte { return bytes.Repeat([]byte{b}, 32) }

func rawSetting(t *testing.T, s *Store, key string) string {
	t.Helper()
	var v string
	if err := s.DB.NewSelect().Model((*Setting)(nil)).Column("svalue").Where("skey = ?", key).Scan(context.Background(), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestSecretsRoundtripAndLegacyMigration(t *testing.T) {
	ctx := context.Background()
	path := "sqlite:" + filepath.Join(t.TempDir(), "s.db")
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	// legacy install: plaintext secrets written before encryption existed
	if err := s.SetSetting(ctx, "tmdb.api_key", "tmdb-plain"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetSetting(ctx, "app.language", "fr"); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveServarr(ctx, &ServarrInstance{ID: "r1", Kind: "radarr", Name: "R", URL: "http://r", APIKey: "radarr-key"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveWebhook(ctx, &Webhook{ID: "w1", Name: "W", URL: "http://w", Secret: "hmac", Headers: `[{"name":"X-Token","value":"tok"}]`, AuthSecret: "pw"}); err != nil {
		t.Fatal(err)
	}

	if err := s.EnableSecrets(ctx, testKey(1)); err != nil {
		t.Fatal(err)
	}
	if v := rawSetting(t, s, "tmdb.api_key"); !strings.HasPrefix(v, secretPrefix) {
		t.Fatalf("legacy setting not migrated: %q", v)
	}
	if v := rawSetting(t, s, "app.language"); v != "fr" {
		t.Fatalf("non-secret setting changed: %q", v)
	}
	var raw ServarrInstance
	if err := s.DB.NewSelect().Model(&raw).Where("id = ?", "r1").Scan(ctx); err != nil || !strings.HasPrefix(raw.APIKey, secretPrefix) {
		t.Fatalf("servarr key not encrypted: %q %v", raw.APIKey, err)
	}
	var rawW Webhook
	if err := s.DB.NewSelect().Model(&rawW).Where("id = ?", "w1").Scan(ctx); err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{rawW.Secret, rawW.Headers, rawW.AuthSecret} {
		if !strings.HasPrefix(v, secretPrefix) {
			t.Fatalf("webhook column not encrypted: %q", v)
		}
	}

	// reads are transparent
	if v, _ := s.GetSetting(ctx, "tmdb.api_key"); v != "tmdb-plain" {
		t.Fatalf("got %q", v)
	}
	in, err := s.GetServarr(ctx, "r1")
	if err != nil || in.APIKey != "radarr-key" {
		t.Fatalf("servarr %+v %v", in, err)
	}
	w, err := s.GetWebhook(ctx, "w1")
	if err != nil || w.Secret != "hmac" || w.AuthSecret != "pw" || !strings.Contains(w.Headers, "tok") {
		t.Fatalf("webhook %+v %v", w, err)
	}

	// new writes are encrypted, the caller's struct stays plaintext
	nw := &Webhook{ID: "w2", Name: "W2", URL: "http://w", Secret: "s2"}
	if err := s.SaveWebhook(ctx, nw); err != nil || nw.Secret != "s2" {
		t.Fatalf("%v %q", err, nw.Secret)
	}
	if err := s.SetSetting(ctx, "oidc.client_secret", "oidc"); err != nil {
		t.Fatal(err)
	}
	if v := rawSetting(t, s, "oidc.client_secret"); !strings.HasPrefix(v, secretPrefix) {
		t.Fatalf("new setting stored in plaintext: %q", v)
	}
	if ok, err := s.SetSettingIfAbsent(ctx, "api.key", "dash"); !ok || err != nil {
		t.Fatal(ok, err)
	}
	if v := rawSetting(t, s, "api.key"); !strings.HasPrefix(v, secretPrefix) {
		t.Fatalf("api key stored in plaintext: %q", v)
	}
	_ = s.Close()

	// restart with the same key: no rotation needed, values still readable
	s, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.EnableSecrets(ctx, testKey(1)); err != nil {
		t.Fatal(err)
	}
	if v, _ := s.GetSetting(ctx, "oidc.client_secret"); v != "oidc" {
		t.Fatalf("got %q", v)
	}
	_ = s.Close()

	// wrong key: clear error at start
	s, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	err = s.EnableSecrets(ctx, testKey(2))
	if !errors.Is(err, ErrSecretKey) || !strings.Contains(err.Error(), SecretKeyFile) {
		t.Fatalf("want ErrSecretKey naming the key file, got %v", err)
	}
	// without the key the encrypted values are not returned as garbage
	if _, err := s.GetSetting(ctx, "oidc.client_secret"); !errors.Is(err, ErrSecretKey) {
		t.Fatalf("got %v", err)
	}
}

func TestWrongKeyWithoutSecrets(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, "sqlite:"+filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.EnableSecrets(ctx, testKey(1)); err != nil {
		t.Fatal(err)
	}
	if err := s.EnableSecrets(ctx, testKey(3)); !errors.Is(err, ErrSecretKey) {
		t.Fatalf("check value must catch a wrong key, got %v", err)
	}
}

func TestLoadSecretKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(SecretKeyEnv, "")
	k1, err := LoadSecretKey(dir)
	if err != nil || len(k1) != 32 {
		t.Fatal(k1, err)
	}
	fi, err := os.Stat(filepath.Join(dir, SecretKeyFile))
	if err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("key file mode %v %v", fi, err)
	}
	k2, _ := LoadSecretKey(dir)
	if !bytes.Equal(k1, k2) {
		t.Fatal("key file not reused")
	}
	t.Setenv(SecretKeyEnv, base64.StdEncoding.EncodeToString(testKey(7)))
	if k, err := LoadSecretKey(dir); err != nil || !bytes.Equal(k, testKey(7)) {
		t.Fatal("env key ignored", err)
	}
	t.Setenv(SecretKeyEnv, "short")
	if _, err := LoadSecretKey(dir); err == nil {
		t.Fatal("invalid env key accepted")
	}
}

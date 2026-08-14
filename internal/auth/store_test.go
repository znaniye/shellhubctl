package auth

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreSaveLoadRoundTrip(t *testing.T) {
	store := NewStoreAt(filepath.Join(t.TempDir(), "nested", "session.json"))
	want := &Session{
		Server:    "https://shellhub.example.com",
		Token:     "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjMifQ.signature",
		UserID:    "user-123",
		Username:  "alice",
		Email:     "alice@example.com",
		Tenant:    "tenant-acme",
		Role:      "owner",
		ExpiresAt: time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC),
	}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.Server != want.Server {
		t.Errorf("Server = %q, want %q", got.Server, want.Server)
	}

	if got.Token != want.Token {
		t.Errorf("Token = %q, want %q", got.Token, want.Token)
	}

	if got.UserID != want.UserID {
		t.Errorf("UserID = %q, want %q", got.UserID, want.UserID)
	}

	if got.Username != want.Username {
		t.Errorf("Username = %q, want %q", got.Username, want.Username)
	}

	if got.Email != want.Email {
		t.Errorf("Email = %q, want %q", got.Email, want.Email)
	}

	if got.Tenant != want.Tenant {
		t.Errorf("Tenant = %q, want %q", got.Tenant, want.Tenant)
	}

	if got.Role != want.Role {
		t.Errorf("Role = %q, want %q", got.Role, want.Role)
	}

	if !got.ExpiresAt.Equal(want.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", got.ExpiresAt, want.ExpiresAt)
	}
}

func TestStoreSavePermissions(t *testing.T) {
	dir := t.TempDir()
	store := NewStoreAt(filepath.Join(dir, "nested", "session.json"))

	if err := store.Save(&Session{Server: "https://s.example", Token: "tok"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	dirInfo, err := os.Stat(filepath.Join(dir, "nested"))
	if err != nil {
		t.Fatalf("stat session directory: %v", err)
	}

	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Errorf("session directory perm = %o, want 700", got)
	}

	fileInfo, err := os.Stat(store.Path())
	if err != nil {
		t.Fatalf("stat session file: %v", err)
	}

	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Errorf("session file perm = %o, want 600", got)
	}
}

func TestStoreSaveTightensPermissiveFile(t *testing.T) {
	store := NewStoreAt(filepath.Join(t.TempDir(), "session.json"))
	if err := os.WriteFile(store.Path(), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write permissive file: %v", err)
	}

	if err := os.Chmod(store.Path(), 0o644); err != nil {
		t.Fatalf("chmod permissive file: %v", err)
	}

	if err := store.Save(&Session{Server: "https://s.example", Token: "tok"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(store.Path())
	if err != nil {
		t.Fatalf("stat session file: %v", err)
	}

	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("session file perm = %o, want 600", got)
	}
}

func TestStoreLoadNoSession(t *testing.T) {
	store := NewStoreAt(filepath.Join(t.TempDir(), "session.json"))

	_, err := store.Load()
	if !errors.Is(err, ErrNoSession) {
		t.Fatalf("Load() error = %v, want ErrNoSession", err)
	}
}

func TestStoreLoadCorruptJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatalf("write corrupt session file: %v", err)
	}

	store := NewStoreAt(path)

	_, err := store.Load()
	if err == nil {
		t.Fatal("Load() expected an error for corrupt JSON")
	}

	if errors.Is(err, ErrNoSession) {
		t.Fatal("Load() returned ErrNoSession for corrupt JSON, want a diagnostic error")
	}
}

func TestStoreClear(t *testing.T) {
	store := NewStoreAt(filepath.Join(t.TempDir(), "session.json"))

	if err := store.Clear(); err != nil {
		t.Fatalf("Clear() without file error = %v", err)
	}

	if err := store.Save(&Session{Server: "https://s.example", Token: "tok"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := store.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	if _, err := os.Stat(store.Path()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("session file still exists after Clear(), stat error = %v", err)
	}
}

func TestNewStoreDefaultPath(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(base, "xdg"))

	store, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	want := filepath.Join(base, "xdg", "shellhub-tui", "session.json")
	if store.Path() != want {
		t.Errorf("Path() = %q, want %q", store.Path(), want)
	}
}

func TestNewStoreFallsBackToHome(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", filepath.Join(base, "home"))

	store, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	want := filepath.Join(base, "home", ".config", "shellhub-tui", "session.json")
	if store.Path() != want {
		t.Errorf("Path() = %q, want %q", store.Path(), want)
	}
}

package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrNoSession = errors.New("no session stored")

type Store struct {
	path string
}

func NewStore() (*Store, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("auth: resolve home directory: %w", err)
		}

		dir = filepath.Join(home, ".config")
	}

	return NewStoreAt(filepath.Join(dir, "shellhub-tui", "session.json")), nil
}

func NewStoreAt(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Load() (*Session, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoSession
		}

		return nil, fmt.Errorf("auth: read session file: %w", err)
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("auth: parse session file: %w", err)
	}

	return &sess, nil
}

func (s *Store) Save(sess *Session) error {
	dir := filepath.Dir(s.path)

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("auth: create session directory: %w", err)
	}

	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("auth: marshal session: %w", err)
	}

	tmp, err := os.CreateTemp(dir, "session-*.json")
	if err != nil {
		return fmt.Errorf("auth: create temporary session file: %w", err)
	}

	name := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(name)
	}()

	if err := tmp.Chmod(0o600); err != nil {
		return fmt.Errorf("auth: set session file permissions: %w", err)
	}

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("auth: write session file: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("auth: sync session file: %w", err)
	}

	if err := os.Rename(name, s.path); err != nil {
		return fmt.Errorf("auth: replace session file: %w", err)
	}

	return nil
}

func (s *Store) Clear() error {
	err := os.Remove(s.path)
	if err == nil || errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return fmt.Errorf("auth: remove session file: %w", err)
}

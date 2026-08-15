package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeKeyFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write key file: %v", err)
	}

	return path
}

func TestAPIKeySources(t *testing.T) {
	file := writeKeyFile(t, "  key-from-file  \n\n")

	tests := []struct {
		name    string
		env     string
		command string
		file    string
		want    string
	}{
		{
			name: "env only",
			env:  "  key-from-env  \n",
			want: "key-from-env",
		},
		{
			name:    "command only",
			command: "printf '  key-from-command  \\n\\n'",
			want:    "key-from-command",
		},
		{
			name: "file only",
			file: file,
			want: "key-from-file",
		},
		{
			name:    "all three",
			env:     "key-from-env",
			command: "printf 'key-from-command\\n'",
			file:    file,
			want:    "key-from-env",
		},
		{
			name:    "command and file",
			command: "printf 'key-from-command\\n'",
			file:    file,
			want:    "key-from-command",
		},
		{
			name:    "env empty falls to command",
			env:     "",
			command: "printf 'key-from-command\\n'",
			file:    file,
			want:    "key-from-command",
		},
		{
			name:    "env spaces falls to command",
			env:     "   ",
			command: "printf 'key-from-command\\n'",
			file:    file,
			want:    "key-from-command",
		},
		{
			name: "command empty falls to file",
			file: file,
			want: "key-from-file",
		},
		{
			name:    "command spaces falls to file",
			command: "   ",
			file:    file,
			want:    "key-from-file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SHELLHUB_API_KEY", tt.env)
			t.Setenv("SHELLHUB_API_KEY_COMMAND", tt.command)
			t.Setenv("SHELLHUB_API_KEY_FILE", tt.file)

			got, err := APIKey(context.Background())
			if err != nil {
				t.Fatalf("APIKey() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("APIKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPIKeyNoSource(t *testing.T) {
	t.Setenv("SHELLHUB_API_KEY", "")
	t.Setenv("SHELLHUB_API_KEY_COMMAND", "   ")
	t.Setenv("SHELLHUB_API_KEY_FILE", "")

	_, err := APIKey(context.Background())
	if !errors.Is(err, ErrNoAPIKey) {
		t.Fatalf("APIKey() error = %v, want ErrNoAPIKey", err)
	}

	msg := err.Error()

	for _, name := range []string{"SHELLHUB_API_KEY", "SHELLHUB_API_KEY_COMMAND", "SHELLHUB_API_KEY_FILE"} {
		if !strings.Contains(msg, name) {
			t.Errorf("ErrNoAPIKey message = %q, want it to mention %s", msg, name)
		}
	}
}

func TestAPIKeyCommandFailure(t *testing.T) {
	t.Setenv("SHELLHUB_API_KEY_COMMAND", "printf 'boom\\n' >&2; exit 3")

	_, err := APIKey(context.Background())
	if err == nil {
		t.Fatal("APIKey() expected an error")
	}

	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("APIKey() error = %v, want command stderr", err)
	}

	if !strings.Contains(err.Error(), "3") {
		t.Errorf("APIKey() error = %v, want exit code", err)
	}
}

func TestAPIKeyCommandEmptyOutput(t *testing.T) {
	t.Setenv("SHELLHUB_API_KEY_COMMAND", "true")

	_, err := APIKey(context.Background())
	if err == nil {
		t.Fatal("APIKey() expected an error")
	}

	if !strings.Contains(err.Error(), "no output") {
		t.Errorf("APIKey() error = %v, want empty output message", err)
	}
}

func TestAPIKeyCommandWhitespaceOutput(t *testing.T) {
	t.Setenv("SHELLHUB_API_KEY_COMMAND", "printf '  \\n'")

	_, err := APIKey(context.Background())
	if err == nil {
		t.Fatal("APIKey() expected an error")
	}

	if !strings.Contains(err.Error(), "no output") {
		t.Errorf("APIKey() error = %v, want empty output message", err)
	}
}

func TestAPIKeyFileMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-key")
	t.Setenv("SHELLHUB_API_KEY_FILE", path)

	_, err := APIKey(context.Background())
	if err == nil {
		t.Fatal("APIKey() expected an error")
	}

	if !strings.Contains(err.Error(), path) {
		t.Errorf("APIKey() error = %v, want it to name %q", err, path)
	}

	if strings.Contains(err.Error(), "empty") {
		t.Errorf("APIKey() error = %v, want missing-file error", err)
	}
}

func TestAPIKeyFileEmpty(t *testing.T) {
	path := writeKeyFile(t, "")
	t.Setenv("SHELLHUB_API_KEY_FILE", path)

	_, err := APIKey(context.Background())
	if err == nil {
		t.Fatal("APIKey() expected an error")
	}

	if !strings.Contains(err.Error(), path) {
		t.Errorf("APIKey() error = %v, want it to name %q", err, path)
	}

	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("APIKey() error = %v, want empty-file message", err)
	}

	if strings.Contains(err.Error(), "does not exist") {
		t.Errorf("APIKey() error = %v, want empty-file error", err)
	}
}

func TestAPIKeyErrorsNeverExposeKey(t *testing.T) {
	const sentinel = "sentinel-key-3f7a"

	t.Run("env", func(t *testing.T) {
		t.Setenv("SHELLHUB_API_KEY", sentinel)

		got, err := APIKey(context.Background())
		if err != nil {
			t.Fatalf("APIKey() error = %v", err)
		}

		if got != sentinel {
			t.Errorf("APIKey() = %q, want %q", got, sentinel)
		}
	})

	t.Run("command", func(t *testing.T) {
		t.Setenv("SHELLHUB_API_KEY_COMMAND", fmt.Sprintf("printf '%%s\\n' %s; exit 1", sentinel))

		_, err := APIKey(context.Background())
		if err == nil {
			t.Fatal("APIKey() expected an error")
		}

		if strings.Contains(err.Error(), sentinel) {
			t.Errorf("APIKey() error exposes the key: %v", err)
		}
	})

	t.Run("file", func(t *testing.T) {
		path := writeKeyFile(t, sentinel+"\n")
		t.Setenv("SHELLHUB_API_KEY_FILE", path)

		if err := os.Chmod(path, 0o000); err != nil {
			t.Fatalf("chmod key file: %v", err)
		}

		got, err := APIKey(context.Background())
		if err != nil {
			if strings.Contains(err.Error(), sentinel) {
				t.Errorf("APIKey() error exposes the key: %v", err)
			}

			return
		}

		if got != sentinel {
			t.Errorf("APIKey() = %q, want %q", got, sentinel)
		}
	})
}

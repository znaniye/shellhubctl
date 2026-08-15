package auth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var ErrNoAPIKey = errors.New("no API key found: set SHELLHUB_API_KEY to the key itself, SHELLHUB_API_KEY_COMMAND to a shell command that prints the key to stdout, or SHELLHUB_API_KEY_FILE to a file containing the key")

func APIKey(ctx context.Context) (string, error) {
	if key := strings.TrimSpace(os.Getenv("SHELLHUB_API_KEY")); key != "" {
		return key, nil
	}

	if command := strings.TrimSpace(os.Getenv("SHELLHUB_API_KEY_COMMAND")); command != "" {
		return apiKeyFromCommand(ctx, command)
	}

	if path := strings.TrimSpace(os.Getenv("SHELLHUB_API_KEY_FILE")); path != "" {
		return apiKeyFromFile(path)
	}

	return "", ErrNoAPIKey
}

func apiKeyFromCommand(ctx context.Context, command string) (string, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("auth: run API key command: %w", ctx.Err())
		}

		var exitErr *exec.ExitError

		if errors.As(err, &exitErr) {
			detail := strings.TrimSpace(stderr.String())
			if detail == "" {
				return "", fmt.Errorf("auth: API key command failed with exit code %d", exitErr.ExitCode())
			}

			return "", fmt.Errorf("auth: API key command failed with exit code %d: %s", exitErr.ExitCode(), detail)
		}

		return "", fmt.Errorf("auth: run API key command: %w", err)
	}

	key := strings.TrimSpace(stdout.String())
	if key == "" {
		return "", errors.New("auth: API key command produced no output")
	}

	return key, nil
}

func apiKeyFromFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("auth: API key file %q does not exist", path)
		}

		return "", fmt.Errorf("auth: read API key file %q: %w", path, err)
	}

	key := strings.TrimSpace(string(data))
	if key == "" {
		return "", fmt.Errorf("auth: API key file %q is empty", path)
	}

	return key, nil
}

package cli

import "testing"

func TestResolveServer(t *testing.T) {
	t.Setenv("SHELLHUB_URL", "")

	if got := resolveServer("https://flag.example.com"); got != "https://flag.example.com" {
		t.Errorf("resolveServer(flag) = %q, want the flag value", got)
	}

	if got := resolveServer(""); got != "https://cloud.shellhub.io" {
		t.Errorf("resolveServer(\"\") = %q, want https://cloud.shellhub.io", got)
	}

	t.Setenv("SHELLHUB_URL", "https://env.example.com")

	if got := resolveServer(""); got != "https://env.example.com" {
		t.Errorf("resolveServer(\"\") = %q, want $SHELLHUB_URL", got)
	}

	if got := resolveServer("https://flag.example.com"); got != "https://flag.example.com" {
		t.Errorf("resolveServer(flag) = %q, want the flag to win over the env", got)
	}
}

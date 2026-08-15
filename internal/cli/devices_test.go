package cli

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/znaniye/shellhubctl/internal/auth"
	"github.com/znaniye/shellhubctl/internal/selector"
)

func runDevicesCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()

	var out strings.Builder

	cmd := newRootCmd()
	cmd.SetOut(&out)
	cmd.SetArgs(append([]string{"devices"}, args...))

	err := cmd.Execute()

	return out.String(), err
}

func hostOf(t *testing.T, serverURL string) string {
	t.Helper()

	u, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("url.Parse(%q): %v", serverURL, err)
	}

	return u.Hostname()
}

func TestDevicesJSONOutput(t *testing.T) {
	var (
		mu       sync.Mutex
		gotQuery string
		gotKey   string
	)

	body := `[{"uid":"d-1","name":"web-01","online":true,"status":"accepted","namespace":"acme","tenant_id":"t-1","last_seen":"2024-01-02T03:04:05Z","created_at":"2024-01-01T00:00:00Z","info":{"id":"i-1","pretty_name":"Ubuntu 22.04","version":"22.04","arch":"amd64","platform":"linux"},"identity":{"mac":"aa:bb:cc:dd:ee:ff"},"tags":[{"tenant_id":"t-1","name":"builder"},{"name":"prod"}]}]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/devices" {
			t.Errorf("path = %q, want /api/devices", r.URL.Path)
		}

		mu.Lock()
		gotQuery = r.URL.RawQuery
		gotKey = r.Header.Get("X-API-Key")
		mu.Unlock()

		w.Header().Set("X-Total-Count", "1")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("SHELLHUB_API_KEY", "key-1")

	out, err := runDevicesCmd(t, "--server", srv.URL, "--ssh-user", "deploy")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	mu.Lock()

	if gotKey != "key-1" {
		t.Errorf("X-API-Key = %q, want key-1", gotKey)
	}

	for _, want := range []string{"status=accepted", "sort_by=name", "order_by=asc"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
	mu.Unlock()

	if got := strings.Count(out, `"tags":`); got != 1 {
		t.Errorf("tags key count = %d, want 1\noutput: %s", got, out)
	}

	var devices []deviceJSON
	if err := json.Unmarshal([]byte(out), &devices); err != nil {
		t.Fatalf("Unmarshal: %v\noutput: %s", err, out)
	}

	if len(devices) != 1 {
		t.Fatalf("len(devices) = %d, want 1", len(devices))
	}

	device := devices[0]

	if device.UID != "d-1" || device.Name != "web-01" || !device.Online || device.Status != "accepted" || device.Namespace != "acme" || device.TenantID != "t-1" {
		t.Errorf("device = %+v", device)
	}

	if device.Info.PrettyName != "Ubuntu 22.04" || device.Identity.MAC != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("info = %+v, identity = %+v", device.Info, device.Identity)
	}

	if want := []string{"builder", "prod"}; !reflect.DeepEqual(device.Tags, want) {
		t.Errorf("Tags = %v, want %v", device.Tags, want)
	}

	if want := "deploy@acme.web-01@" + hostOf(t, srv.URL); device.SSHID != want {
		t.Errorf("SSHID = %q, want %q", device.SSHID, want)
	}
}

func TestDevicesEmptyOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Total-Count", "0")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[]`)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("SHELLHUB_API_KEY", "key-1")

	out, err := runDevicesCmd(t, "--server", srv.URL)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if out != "[]\n" {
		t.Errorf("output = %q, want %q", out, "[]\n")
	}
}

func TestDevicesFilteredEmptyOutput(t *testing.T) {
	body := `[{"uid":"d-1","name":"web-01","online":true,"status":"accepted","namespace":"acme","info":{"pretty_name":"Ubuntu 22.04"}}]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Total-Count", "1")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("SHELLHUB_API_KEY", "key-1")

	out, err := runDevicesCmd(t, "--server", srv.URL, "--name", "nomatch")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if out != "[]\n" {
		t.Errorf("output = %q, want %q", out, "[]\n")
	}
}

func TestDevicesNameFilterApplies(t *testing.T) {
	body := `[{"uid":"d-1","name":"web-01","namespace":"acme","status":"accepted"},{"uid":"d-2","name":"db-01","namespace":"acme","status":"accepted"}]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Total-Count", "2")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("SHELLHUB_API_KEY", "key-1")

	out, err := runDevicesCmd(t, "--server", srv.URL, "--name", "^web")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var devices []deviceJSON
	if err := json.Unmarshal([]byte(out), &devices); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(devices) != 1 || devices[0].UID != "d-1" {
		t.Errorf("devices = %+v, want only d-1", devices)
	}
}

func TestDevicesSelectorOptions(t *testing.T) {
	cmd := newDevicesCmd()

	if err := cmd.Flags().Set("name", "web"); err != nil {
		t.Fatalf("Set name: %v", err)
	}

	if err := cmd.Flags().Set("name", "db"); err != nil {
		t.Fatalf("Set name: %v", err)
	}

	if err := cmd.Flags().Set("distro", "ubuntu"); err != nil {
		t.Fatalf("Set distro: %v", err)
	}

	if err := cmd.Flags().Set("tag", "prod"); err != nil {
		t.Fatalf("Set tag: %v", err)
	}

	if err := cmd.Flags().Set("tag", "eu"); err != nil {
		t.Fatalf("Set tag: %v", err)
	}

	if err := cmd.Flags().Set("online", "true"); err != nil {
		t.Fatalf("Set online: %v", err)
	}

	opts, err := devicesSelector(cmd)
	if err != nil {
		t.Fatalf("devicesSelector: %v", err)
	}

	want := selector.Options{
		Names:   []string{"web", "db"},
		Distros: []string{"ubuntu"},
		Tags:    []string{"prod", "eu"},
		Online:  true,
	}

	if !reflect.DeepEqual(opts, want) {
		t.Errorf("Options = %+v, want %+v", opts, want)
	}
}

func TestResolveSSHUser(t *testing.T) {
	t.Setenv("SHELLHUB_SSH_USER", "")

	if got := resolveSSHUser("deploy"); got != "deploy" {
		t.Errorf("resolveSSHUser(flag) = %q, want the flag value", got)
	}

	if got := resolveSSHUser(""); got != "root" {
		t.Errorf("resolveSSHUser(\"\") = %q, want root", got)
	}

	t.Setenv("SHELLHUB_SSH_USER", "deploy")

	if got := resolveSSHUser(""); got != "deploy" {
		t.Errorf("resolveSSHUser(\"\") = %q, want $SHELLHUB_SSH_USER", got)
	}

	if got := resolveSSHUser("alice"); got != "alice" {
		t.Errorf("resolveSSHUser(flag) = %q, want the flag to win over the env", got)
	}
}

func TestDevicesSSHUserFromEnv(t *testing.T) {
	body := `[{"uid":"d-1","name":"web-01","namespace":"acme","status":"accepted"}]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Total-Count", "1")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("SHELLHUB_API_KEY", "key-1")
	t.Setenv("SHELLHUB_SSH_USER", "deploy")

	out, err := runDevicesCmd(t, "--server", srv.URL)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if want := "deploy@acme.web-01@" + hostOf(t, srv.URL); !strings.Contains(out, want) {
		t.Errorf("output missing sshid %q:\n%s", want, out)
	}
}

func TestDevicesUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("SHELLHUB_API_KEY", "key-1")

	_, err := runDevicesCmd(t, "--server", srv.URL)
	if err == nil {
		t.Fatal("Execute: expected error")
	}

	if !strings.Contains(err.Error(), "rejected") {
		t.Errorf("error = %q, want a rejected-key message", err)
	}

	if strings.Contains(err.Error(), "permission") {
		t.Errorf("error = %q, must not be the forbidden message", err)
	}
}

func TestDevicesForbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("SHELLHUB_API_KEY", "key-1")

	_, err := runDevicesCmd(t, "--server", srv.URL)
	if err == nil {
		t.Fatal("Execute: expected error")
	}

	if !strings.Contains(err.Error(), "permission") {
		t.Errorf("error = %q, want a missing-permission message", err)
	}

	if strings.Contains(err.Error(), "rejected") {
		t.Errorf("error = %q, must not be the unauthorized message", err)
	}
}

func TestDevicesErrorHidesAPIKey(t *testing.T) {
	const key = "super-secret-key-42"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("SHELLHUB_API_KEY", key)

	var out, errOut strings.Builder

	cmd := newRootCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"devices", "--server", srv.URL})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute: expected error")
	}

	if strings.Contains(err.Error(), key) {
		t.Errorf("returned error leaks the API key: %q", err)
	}

	if strings.Contains(out.String(), key) {
		t.Errorf("stdout leaks the API key: %q", out.String())
	}

	if strings.Contains(errOut.String(), key) {
		t.Errorf("stderr leaks the API key: %q", errOut.String())
	}
}

func TestDevicesNoAPIKey(t *testing.T) {
	t.Setenv("SHELLHUB_API_KEY", "")
	t.Setenv("SHELLHUB_API_KEY_COMMAND", "")
	t.Setenv("SHELLHUB_API_KEY_FILE", "")

	_, err := runDevicesCmd(t, "--server", "http://example.com")
	if err == nil {
		t.Fatal("Execute: expected error")
	}

	if !errors.Is(err, auth.ErrNoAPIKey) {
		t.Errorf("error = %v, want ErrNoAPIKey", err)
	}
}

func TestDevicesInvalidRegex(t *testing.T) {
	t.Setenv("SHELLHUB_API_KEY", "key-1")

	_, err := runDevicesCmd(t, "--server", "http://example.com", "--name", "[")
	if err == nil {
		t.Fatal("Execute: expected error")
	}

	if !strings.Contains(err.Error(), "invalid regex") {
		t.Errorf("error = %q, want an invalid-regex message", err)
	}
}

func TestDevicesSSHIDErrorNamesDevice(t *testing.T) {
	body := `[{"uid":"d-7","name":"broken-01","namespace":"","status":"accepted"},{"uid":"d-8","name":"fine-01","namespace":"acme","status":"accepted"}]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Total-Count", "2")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("SHELLHUB_API_KEY", "key-1")

	_, err := runDevicesCmd(t, "--server", srv.URL)
	if err == nil {
		t.Fatal("Execute: expected error")
	}

	if !strings.Contains(err.Error(), "broken-01") {
		t.Errorf("error = %q, want it to name the offending device", err)
	}

	if !strings.Contains(err.Error(), "namespace is empty") {
		t.Errorf("error = %q, want the underlying validation error", err)
	}
}

func TestDevicesSSHIDErrorFallsBackToUID(t *testing.T) {
	body := `[{"uid":"d-9","name":"","namespace":"","status":"accepted"}]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Total-Count", "1")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("SHELLHUB_API_KEY", "key-1")

	_, err := runDevicesCmd(t, "--server", srv.URL)
	if err == nil {
		t.Fatal("Execute: expected error")
	}

	if !strings.Contains(err.Error(), "d-9") {
		t.Errorf("error = %q, want it to fall back to the device UID", err)
	}
}

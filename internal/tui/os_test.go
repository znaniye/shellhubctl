package tui

import (
	"testing"

	"github.com/znaniye/shellhubctl/internal/shellhub"
)

func TestDeviceColumnsTitles(t *testing.T) {
	m := devicesModel{width: 100}

	want := []string{"NAME", "STATUS", "ONLINE", "OS", "LAST SEEN"}

	cols := m.columns()
	if len(cols) != len(want) {
		t.Fatalf("columns() = %d columns, want %d", len(cols), len(want))
	}

	for i := range want {
		if cols[i].Title != want[i] {
			t.Errorf("columns()[%d].Title = %q, want %q", i, cols[i].Title, want[i])
		}
	}
}

func TestNameColumnGetsTheLeftoverWidth(t *testing.T) {
	wide := devicesModel{width: 120}

	cols := wide.columns()

	fixed := tableCellPadding
	for _, col := range cols[1:] {
		fixed += col.Width + tableCellPadding
	}

	if want := wide.width - fixed; cols[0].Width != want {
		t.Errorf("NAME width = %d, want the leftover %d", cols[0].Width, want)
	}

	narrow := devicesModel{width: 30}

	if got := narrow.columns()[0].Width; got != minNameWidth {
		t.Errorf("narrow NAME width = %d, want the %d floor", got, minNameWidth)
	}
}

func TestOSLabelPrefersPrettyName(t *testing.T) {
	info := shellhub.DeviceInfo{
		ID:         "ubuntu",
		PrettyName: "Ubuntu 22.04.3 LTS",
		Platform:   "native",
		Version:    "15.0.0",
		Arch:       "x86_64",
	}

	if got := osLabel(info); got != "Ubuntu 22.04.3 LTS · x86_64" {
		t.Errorf("osLabel = %q, want the distro with arch", got)
	}
}

func TestOSLabelFallsBackToID(t *testing.T) {
	info := shellhub.DeviceInfo{
		ID:       "nixos",
		Platform: "docker",
	}

	if got := osLabel(info); got != "nixos" {
		t.Errorf("osLabel = %q, want the os-release id", got)
	}
}

func TestOSLabelNeverAppendsAgentVersion(t *testing.T) {
	info := shellhub.DeviceInfo{
		ID:      "ubuntu",
		Version: "15.0.0",
	}

	if got := osLabel(info); got != "ubuntu" {
		t.Errorf("osLabel = %q, want the distro id without the agent version", got)
	}
}

func TestOSLabelEmptyInfo(t *testing.T) {
	if got := osLabel(shellhub.DeviceInfo{}); got != "-" {
		t.Errorf("osLabel(empty) = %q, want -", got)
	}

	if got := osLabel(shellhub.DeviceInfo{Platform: "native"}); got != "-" {
		t.Errorf("osLabel(platform only) = %q, want -", got)
	}

	if got := osLabel(shellhub.DeviceInfo{Version: "15.0.0"}); got != "-" {
		t.Errorf("osLabel(version only) = %q, want -", got)
	}
}

func TestDeviceRowShowsTheDistro(t *testing.T) {
	d := shellhub.Device{
		Name:   "pi-01",
		Status: "accepted",
		Online: true,
		Info: shellhub.DeviceInfo{
			ID:         "debian",
			PrettyName: "Debian GNU/Linux 12 (bookworm)",
			Platform:   "bundle",
			Version:    "15.0.0",
			Arch:       "arm64",
		},
	}

	row := deviceRow(d)
	if len(row) != 5 {
		t.Fatalf("deviceRow returned %d cells, want 5", len(row))
	}

	if row[3] != "Debian GNU/Linux 12 (bookworm) · arm64" {
		t.Errorf("OS cell = %q, want the distro with arch", row[3])
	}
}

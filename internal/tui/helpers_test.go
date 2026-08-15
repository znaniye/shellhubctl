package tui

import (
	"testing"
	"time"

	"github.com/znaniye/shellhubctl/internal/shellhub"
)

func TestOnlineLabel(t *testing.T) {
	if got := onlineLabel(true); got != "yes" {
		t.Errorf("onlineLabel(true) = %q, want yes", got)
	}

	if got := onlineLabel(false); got != "no" {
		t.Errorf("onlineLabel(false) = %q, want no", got)
	}
}

func TestOSLabel(t *testing.T) {
	tests := []struct {
		name string
		info shellhub.DeviceInfo
		want string
	}{
		{
			name: "pretty name and arch",
			info: shellhub.DeviceInfo{PrettyName: "Ubuntu 22.04.3 LTS", Arch: "arm64"},
			want: "Ubuntu 22.04.3 LTS · arm64",
		},
		{
			name: "pretty name only",
			info: shellhub.DeviceInfo{PrettyName: "NixOS 24.05 (Uakari)"},
			want: "NixOS 24.05 (Uakari)",
		},
		{
			name: "id fallback with arch",
			info: shellhub.DeviceInfo{ID: "debian", Arch: "arm64"},
			want: "debian · arm64",
		},
		{
			name: "unknown arch dropped",
			info: shellhub.DeviceInfo{PrettyName: "Raspberry Pi OS", Arch: "unknown"},
			want: "Raspberry Pi OS",
		},
		{
			name: "nothing known",
			info: shellhub.DeviceInfo{},
			want: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := osLabel(tt.info); got != tt.want {
				t.Errorf("osLabel(%+v) = %q, want %q", tt.info, got, tt.want)
			}
		})
	}
}

func TestFormatLastSeen(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		in   time.Time
		want string
	}{
		{name: "zero", in: time.Time{}, want: "never"},
		{name: "just now", in: now.Add(-10 * time.Second), want: "just now"},
		{name: "minutes", in: now.Add(-5 * time.Minute), want: "5 min ago"},
		{name: "hours", in: now.Add(-3 * time.Hour), want: "3 h ago"},
		{name: "days", in: now.Add(-48 * time.Hour), want: "2 d ago"},
		{name: "older", in: time.Date(2020, 5, 6, 7, 8, 9, 0, time.UTC), want: "2020-05-06"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatLastSeen(tt.in); got != tt.want {
				t.Errorf("formatLastSeen(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestDeviceCountFooter(t *testing.T) {
	if got := deviceCountFooter(100, 250); got != "100 of 250 devices" {
		t.Errorf("deviceCountFooter(100, 250) = %q", got)
	}
}

func TestDeviceRow(t *testing.T) {
	d := shellhub.Device{
		Name:   "raspberry",
		Status: "accepted",
		Online: true,
		Info:   shellhub.DeviceInfo{PrettyName: "Raspberry Pi OS", Arch: "arm64"},
	}

	row := deviceRow(d)
	if len(row) != 5 {
		t.Fatalf("deviceRow returned %d cells, want 5", len(row))
	}

	want := []string{"raspberry", "accepted", "yes", "Raspberry Pi OS · arm64", "never"}
	for i := range want {
		if row[i] != want[i] {
			t.Errorf("row[%d] = %q, want %q", i, row[i], want[i])
		}
	}
}

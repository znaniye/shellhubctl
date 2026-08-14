package tui

import (
	"testing"
	"time"

	"github.com/znaniye/shellhub-tui/internal/shellhub"
)

func TestOnlineLabel(t *testing.T) {
	if got := onlineLabel(true); got != "yes" {
		t.Errorf("onlineLabel(true) = %q, want yes", got)
	}

	if got := onlineLabel(false); got != "no" {
		t.Errorf("onlineLabel(false) = %q, want no", got)
	}
}

func TestPlatformLabel(t *testing.T) {
	tests := []struct {
		name string
		info shellhub.DeviceInfo
		want string
	}{
		{
			name: "platform and arch",
			info: shellhub.DeviceInfo{Platform: "linux", Arch: "arm64"},
			want: "linux · arm64",
		},
		{
			name: "platform only",
			info: shellhub.DeviceInfo{Platform: "linux"},
			want: "linux",
		},
		{
			name: "pretty name fallback",
			info: shellhub.DeviceInfo{PrettyName: "Raspberry Pi", Arch: "armv7"},
			want: "Raspberry Pi · armv7",
		},
		{
			name: "unknown arch dropped",
			info: shellhub.DeviceInfo{Platform: "linux", Arch: "unknown"},
			want: "linux",
		},
		{
			name: "nothing known",
			info: shellhub.DeviceInfo{},
			want: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := platformLabel(tt.info); got != tt.want {
				t.Errorf("platformLabel(%+v) = %q, want %q", tt.info, got, tt.want)
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
		Info:   shellhub.DeviceInfo{Platform: "linux", Arch: "arm64"},
	}

	row := deviceRow(d)
	if len(row) != 5 {
		t.Fatalf("deviceRow returned %d cells, want 5", len(row))
	}

	want := []string{"raspberry", "accepted", "yes", "linux · arm64", "never"}
	for i := range want {
		if row[i] != want[i] {
			t.Errorf("row[%d] = %q, want %q", i, row[i], want[i])
		}
	}
}

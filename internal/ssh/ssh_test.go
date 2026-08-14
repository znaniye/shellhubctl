package ssh

import (
	"reflect"
	"testing"
)

func TestSSHID(t *testing.T) {
	cfg := Config{
		User:      "root",
		Namespace: "alpha",
		Device:    "dev1",
		Host:      "cloud.shellhub.io",
	}

	got := cfg.SSHID()

	want := "root@alpha.dev1@cloud.shellhub.io"
	if got != want {
		t.Errorf("SSHID = %q, want %q", got, want)
	}
}

func TestCommandUsesTheDefaultSSHPort(t *testing.T) {
	cfg := Config{
		User:      "root",
		Namespace: "alpha",
		Device:    "dev1",
		Host:      "cloud.shellhub.io",
	}

	name, args := cfg.Command()

	if name != "ssh" {
		t.Errorf("name = %q, want ssh", name)
	}

	want := []string{"root@alpha.dev1@cloud.shellhub.io"}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("args = %v, want %v", args, want)
	}
}

func TestValidate(t *testing.T) {
	valid := Config{
		User:      "root",
		Namespace: "alpha",
		Device:    "dev1",
		Host:      "cloud.shellhub.io",
	}

	if err := valid.Validate(); err != nil {
		t.Errorf("Validate(valid) = %v, want nil", err)
	}

	for _, field := range []string{"User", "Namespace", "Device", "Host"} {
		t.Run(field, func(t *testing.T) {
			cfg := valid

			switch field {
			case "User":
				cfg.User = ""
			case "Namespace":
				cfg.Namespace = ""
			case "Device":
				cfg.Device = ""
			case "Host":
				cfg.Host = ""
			}

			if err := cfg.Validate(); err == nil {
				t.Errorf("Validate with empty %s = nil, want error", field)
			}
		})
	}
}

package ssh

import (
	"errors"
	"strings"
)

type Config struct {
	User      string
	Namespace string
	Device    string
	Host      string
}

func (c Config) SSHID() string {
	return strings.Join([]string{c.User, "@", c.Namespace, ".", c.Device, "@", c.Host}, "")
}

func (c Config) Command() (string, []string) {
	return "ssh", []string{c.SSHID()}
}

func (c Config) Validate() error {
	switch {
	case c.User == "":
		return errors.New("ssh: device user is empty")
	case c.Namespace == "":
		return errors.New("ssh: namespace is empty")
	case c.Device == "":
		return errors.New("ssh: device name is empty")
	case c.Host == "":
		return errors.New("ssh: gateway host is empty")
	default:
		return nil
	}
}

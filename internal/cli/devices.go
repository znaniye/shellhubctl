package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/znaniye/shellhubctl/internal/auth"
	"github.com/znaniye/shellhubctl/internal/selector"
	"github.com/znaniye/shellhubctl/internal/shellhub"
	"github.com/znaniye/shellhubctl/internal/ssh"
)

type deviceJSON struct {
	shellhub.Device
	SSHID string   `json:"sshid"`
	Tags  []string `json:"tags"`
}

func newDevicesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "devices",
		Short: "List namespace devices as JSON",
		RunE:  runDevices,
	}

	cmd.Flags().String("status", "accepted", "device status to list")
	cmd.Flags().String("ssh-user", resolveSSHUser(""), "SSH user for the sshid field (default $SHELLHUB_SSH_USER, then root)")
	cmd.Flags().StringArray("name", nil, "regex to match against device names (repeatable)")
	cmd.Flags().StringArray("distro", nil, "regex to match against the OS pretty name (repeatable)")
	cmd.Flags().StringArray("tag", nil, "exact tag to require (repeatable)")
	cmd.Flags().Bool("online", false, "only list online devices")

	return cmd
}

func resolveSSHUser(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}

	if env := os.Getenv("SHELLHUB_SSH_USER"); env != "" {
		return env
	}

	return "root"
}

func runDevices(cmd *cobra.Command, _ []string) error {
	serverFlag, err := cmd.Flags().GetString("server")
	if err != nil {
		return err
	}

	key, err := auth.APIKey(cmd.Context())
	if err != nil {
		return err
	}

	client, err := shellhub.New(resolveServer(serverFlag))
	if err != nil {
		return err
	}

	client.SetAPIKey(key)

	opts, err := devicesSelector(cmd)
	if err != nil {
		return err
	}

	matcher, err := opts.Compile()
	if err != nil {
		return err
	}

	status, err := cmd.Flags().GetString("status")
	if err != nil {
		return err
	}

	sshUser, err := cmd.Flags().GetString("ssh-user")
	if err != nil {
		return err
	}

	devices, err := client.ListAllDevices(cmd.Context(), shellhub.DeviceListOptions{
		Status:  status,
		SortBy:  "name",
		OrderBy: "asc",
	})
	if err != nil {
		return mapDevicesError(err)
	}

	filtered := matcher.Apply(devices)

	data, err := buildDevicesJSON(filtered, sshUser, client.Host())
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))

	return err
}

func devicesSelector(cmd *cobra.Command) (selector.Options, error) {
	names, err := cmd.Flags().GetStringArray("name")
	if err != nil {
		return selector.Options{}, err
	}

	distros, err := cmd.Flags().GetStringArray("distro")
	if err != nil {
		return selector.Options{}, err
	}

	tags, err := cmd.Flags().GetStringArray("tag")
	if err != nil {
		return selector.Options{}, err
	}

	online, err := cmd.Flags().GetBool("online")
	if err != nil {
		return selector.Options{}, err
	}

	return selector.Options{
		Names:   names,
		Distros: distros,
		Tags:    tags,
		Online:  online,
	}, nil
}

func buildDevicesJSON(devices []shellhub.Device, sshUser, host string) ([]byte, error) {
	out := make([]deviceJSON, 0, len(devices))

	for _, device := range devices {
		cfg := ssh.Config{
			User:      sshUser,
			Namespace: device.Namespace,
			Device:    device.Name,
			Host:      host,
		}

		if err := cfg.Validate(); err != nil {
			name := device.Name
			if name == "" {
				name = device.UID
			}

			return nil, fmt.Errorf("devices: cannot build sshid for %q: %w", name, err)
		}

		out = append(out, deviceJSON{
			Device: device,
			SSHID:  cfg.SSHID(),
			Tags:   device.TagNames(),
		})
	}

	return json.MarshalIndent(out, "", "  ")
}

func mapDevicesError(err error) error {
	switch {
	case errors.Is(err, shellhub.ErrUnauthorized):
		return errors.New("devices: API key rejected: unknown, expired or revoked")
	case errors.Is(err, shellhub.ErrForbidden):
		return errors.New("devices: API key lacks permission for this operation")
	default:
		return err
	}
}

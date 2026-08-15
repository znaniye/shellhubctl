package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/znaniye/shellhubctl/internal/auth"
	"github.com/znaniye/shellhubctl/internal/shellhub"
	"github.com/znaniye/shellhubctl/internal/tui"
)

var version = "dev"

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "shellhubctl",
		Short:         "Terminal UI for ShellHub",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			flag, err := cmd.Flags().GetString("server")
			if err != nil {
				return err
			}

			server := resolveServer(flag)

			client, err := shellhub.New(server)
			if err != nil {
				return fmt.Errorf("shellhub: %w", err)
			}

			store, err := auth.NewStore()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			sess, err := store.Load()
			if err != nil && !errors.Is(err, auth.ErrNoSession) {
				return fmt.Errorf("auth: %w", err)
			}

			if sess == nil || !sess.Valid(server) {
				sess = nil
			} else {
				client.SetToken(sess.Token)
			}

			return tui.Run(cmd.Context(), tui.Options{
				Client:  client,
				Store:   store,
				Session: sess,
			})
		},
	}

	cmd.PersistentFlags().String("server", "", "ShellHub server URL (default $SHELLHUB_URL, then https://cloud.shellhub.io)")

	cmd.AddCommand(newLogoutCmd())

	return cmd
}

func resolveServer(flag string) string {
	if flag != "" {
		return flag
	}

	if env := os.Getenv("SHELLHUB_URL"); env != "" {
		return env
	}

	return "https://cloud.shellhub.io"
}

func Execute(ctx context.Context) error {
	err := newRootCmd().ExecuteContext(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
	}

	return err
}

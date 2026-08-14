package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/znaniye/shellhub-tui/internal/auth"
)

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the locally stored session",
		Long: `Remove the locally stored session.

The ShellHub API has no token revocation endpoint: the token keeps working on
the server until it expires (72h). Logout is purely local.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			store, err := auth.NewStore()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			if err := store.Clear(); err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "session removed locally; the token stays valid on the server until it expires (72h)")

			return nil
		},
	}
}

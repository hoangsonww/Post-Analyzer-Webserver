package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login <username> <password>",
		Short: "Log in and save the JWT for later CLI commands",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := clientFromFlags()
			token, role, err := c.Login(args[0], args[1])
			if err != nil {
				return err
			}
			if err := SaveToken(token); err != nil {
				return fmt.Errorf("login succeeded but failed to save token: %w", err)
			}
			fmt.Println(ok(fmt.Sprintf("Logged in as %s (role: %s)", bold(args[0]), yellow(role))))
			fmt.Println(dim("Token saved to ~/.post-analyzer/token"))
			return nil
		},
	}
	return cmd
}

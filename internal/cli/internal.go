package cli

import (
	"context"
	"errors"

	"github.com/irangarcia/agentwho/internal/enforce"
	"github.com/irangarcia/agentwho/internal/execution"
	"github.com/spf13/cobra"
)

func (a *app) internalCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "internal", Hidden: true}
	execCmd := &cobra.Command{
		Use: "exec <claude|codex> [arguments...]", Hidden: true, DisableFlagParsing: true,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := getPaths()
			if err != nil {
				return err
			}
			err = execution.Run(context.Background(), execution.DefaultRequest(args[0], args[1:], p))
			if errors.Is(err, enforce.ErrRefused) {
				return silent(err)
			}
			return err
		},
	}
	cmd.AddCommand(execCmd, a.shellUseCmd())
	return cmd
}

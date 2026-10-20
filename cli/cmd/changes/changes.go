// Package changes exposes read-only change context and verification facts.
package changes

import (
	"github.com/Tencent/WeKnora/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// NewCmd registers existing-read operations; it never submits execution.
func NewCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{Use: "changes", Short: "Read change context and verification facts"}
	cmd.AddCommand(NewCmdContext(f))
	runs := &cobra.Command{Use: "runs", Short: "Read existing verification runs"}
	runs.AddCommand(NewCmdRunGet(f))
	cmd.AddCommand(runs)
	return cmd
}

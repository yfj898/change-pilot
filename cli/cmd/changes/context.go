package changes

import (
	"context"
	"fmt"
	"github.com/Tencent/WeKnora/cli/internal/cmdutil"
	"github.com/Tencent/WeKnora/cli/internal/iostreams"
	sdk "github.com/Tencent/WeKnora/client"
	"github.com/spf13/cobra"
)

// ContextOptions is intentionally empty: the source is the persisted change.
type ContextOptions struct{}

// ContextService is the one SDK read this command consumes.
type ContextService interface {
	GetChangeContext(context.Context, string) (*sdk.ChangeContext, error)
}

// NewCmdContext builds a read-only context command with the standard envelope.
func NewCmdContext(f *cmdutil.Factory) *cobra.Command {
	opts := &ContextOptions{}
	cmd := &cobra.Command{Use: "context <change-id>", Short: "Read a change's versioned context", Args: cobra.ExactArgs(1), Example: `  weknora changes context <change-id> --format json`, RunE: func(c *cobra.Command, args []string) error {
		fopts, err := cmdutil.CheckFormatFlag(c)
		if err != nil {
			return err
		}
		fopts.ResolveDefault(iostreams.IO.IsStdoutTTY())
		svc, err := f.Client()
		if err != nil {
			return err
		}
		return runContext(c.Context(), opts, fopts, svc, args[0])
	}}
	cmdutil.AddFormatFlag(cmd, "change", "sources", "snapshots", "current_snapshot_id")
	cmdutil.SetAgentHelp(cmd, cmdutil.AgentHelp{UsedFor: "Read one authorized change and its exact versioned source content without changing it.", RequiredFlags: []string{"<change-id> (positional)"}, Examples: []string{"weknora changes context <change-id> --format json"}, Output: "Standard envelope.data contains change (revision, input_revision and code commits), sources (fixed Git references and content), immutable snapshots and current_snapshot_id."})
	return cmd
}
func runContext(ctx context.Context, _ *ContextOptions, fopts *cmdutil.FormatOptions, svc ContextService, id string) error {
	out, err := svc.GetChangeContext(ctx, id)
	if err != nil {
		return cmdutil.WrapHTTP(err, "get change context %q", id)
	}
	if fopts.WantsJSON() {
		return fopts.Emit(iostreams.IO.Out, out, nil)
	}
	fmt.Fprintf(iostreams.IO.Out, "%s\nchange: %s\ninput_revision: %d\ntarget_commit: %s\n", out.Change.Title, out.Change.ID, out.Change.InputRevision, out.Change.Input.TargetCommit)
	for _, source := range out.Sources {
		fmt.Fprintf(iostreams.IO.Out, "\n%s@%s\n%s\n", source.Reference.Path, source.Reference.Commit, source.Content)
	}
	return nil
}

package changes

import (
	"context"
	"fmt"
	"github.com/Tencent/WeKnora/cli/internal/cmdutil"
	"github.com/Tencent/WeKnora/cli/internal/iostreams"
	sdk "github.com/Tencent/WeKnora/client"
	"github.com/spf13/cobra"
)

// RunGetOptions is empty because this command reads one existing run.
type RunGetOptions struct{}

// RunGetService is the narrow SDK contract for run reads.
type RunGetService interface {
	GetChangeVerificationRun(context.Context, string, string) (*sdk.ChangeRunView, error)
}

// NewCmdRunGet does not wait, submit, or retry a mutation.
func NewCmdRunGet(f *cmdutil.Factory) *cobra.Command {
	opts := &RunGetOptions{}
	cmd := &cobra.Command{Use: "get <change-id> <run-id>", Short: "Read one existing verification run", Args: cobra.ExactArgs(2), Example: `  weknora changes runs get <change-id> <run-id> --format json`, RunE: func(c *cobra.Command, args []string) error {
		fopts, err := cmdutil.CheckFormatFlag(c)
		if err != nil {
			return err
		}
		fopts.ResolveDefault(iostreams.IO.IsStdoutTTY())
		svc, err := f.Client()
		if err != nil {
			return err
		}
		return runGet(c.Context(), opts, fopts, svc, args[0], args[1])
	}}
	cmdutil.AddFormatFlag(cmd, "link", "run", "applicable", "execution_error")
	cmdutil.SetAgentHelp(cmd, cmdutil.AgentHelp{UsedFor: "Read one existing verification run. Does not submit, retry or wait for work.", RequiredFlags: []string{"<change-id> (positional)", "<run-id> (positional)"}, Examples: []string{"weknora changes runs get <change-id> <run-id> --format json"}, Output: "Standard envelope.data contains link, applicable, execution_error and optional run with exact commit, task status, generation and attempts. A successful read can contain a failed verification; inspect run.status and attempts[].verdict."})
	return cmd
}
func runGet(ctx context.Context, _ *RunGetOptions, fopts *cmdutil.FormatOptions, svc RunGetService, changeID, runID string) error {
	out, err := svc.GetChangeVerificationRun(ctx, changeID, runID)
	if err != nil {
		return cmdutil.WrapHTTP(err, "get change verification run %q", runID)
	}
	if fopts.WantsJSON() {
		return fopts.Emit(iostreams.IO.Out, out, nil)
	}
	fmt.Fprintf(iostreams.IO.Out, "run: %s\nsubmission: %s\napplicable: %t\n", out.Link.RunID, out.Link.SubmissionState, out.Applicable)
	if out.ExecutionError != "" {
		fmt.Fprintf(iostreams.IO.Out, "execution_error: %s\n", out.ExecutionError)
	}
	if out.Run != nil {
		fmt.Fprintf(iostreams.IO.Out, "status: %s\ntarget_commit: %s\ngeneration: %d\n", out.Run.Status, out.Run.TargetCommit, out.Run.Generation)
		for _, attempt := range out.Run.Attempts {
			fmt.Fprintf(iostreams.IO.Out, "attempt %d: %s\n", attempt.Generation, attempt.Verdict)
			for _, check := range attempt.Checks {
				fmt.Fprintf(iostreams.IO.Out, "  %s: %s %s\n", check.ID, check.Status, check.Error)
			}
		}
	}
	return nil
}

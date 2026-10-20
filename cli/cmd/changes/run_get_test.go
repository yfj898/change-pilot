package changes

import (
	"context"
	"encoding/json"
	"github.com/Tencent/WeKnora/cli/internal/cmdutil"
	"github.com/Tencent/WeKnora/cli/internal/iostreams"
	sdk "github.com/Tencent/WeKnora/client"
	"testing"
)

type runFake struct{ changeID, runID string }

func (f *runFake) GetChangeVerificationRun(_ context.Context, c, r string) (*sdk.ChangeRunView, error) {
	f.changeID = c
	f.runID = r
	return &sdk.ChangeRunView{Link: &sdk.ChangeRunLink{RunID: r, SubmissionState: "accepted"}, Applicable: false, Run: &sdk.VerificationRun{VerificationRequest: sdk.VerificationRequest{RunID: r}, Status: "failed", Generation: 2, Attempts: []sdk.VerificationReport{{Generation: 2, Verdict: "incomplete"}}}}, nil
}
func TestRunGetPreservesHistoricalFailure(t *testing.T) {
	out, _ := iostreams.SetForTest(t)
	f := &runFake{}
	if err := runGet(context.Background(), &RunGetOptions{}, &cmdutil.FormatOptions{Mode: cmdutil.FormatJSON}, f, "change", "run"); err != nil {
		t.Fatal(err)
	}
	var env struct {
		OK   bool              `json:"ok"`
		Data sdk.ChangeRunView `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if !env.OK || env.Data.Applicable || env.Data.Run.Status != "failed" || env.Data.Run.Attempts[0].Verdict != "incomplete" || f.changeID != "change" || f.runID != "run" {
		t.Fatalf("execution facts changed: %s", out.String())
	}
}

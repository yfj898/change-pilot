package changes

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Tencent/WeKnora/cli/internal/cmdutil"
	"github.com/Tencent/WeKnora/cli/internal/iostreams"
	sdk "github.com/Tencent/WeKnora/client"
	"testing"
)

type contextFake struct {
	out *sdk.ChangeContext
	err error
	id  string
}

func (f *contextFake) GetChangeContext(_ context.Context, id string) (*sdk.ChangeContext, error) {
	f.id = id
	return f.out, f.err
}
func TestContextReadUsesStandardEnvelope(t *testing.T) {
	out, _ := iostreams.SetForTest(t)
	f := &contextFake{out: &sdk.ChangeContext{Change: &sdk.ChangeRequest{ID: "change", Revision: 5, InputRevision: 2}, Sources: []sdk.ResolvedChangeReference{}, Snapshots: []*sdk.ChangeSnapshot{}}}
	if err := runContext(context.Background(), &ContextOptions{}, &cmdutil.FormatOptions{Mode: cmdutil.FormatJSON}, f, "change"); err != nil {
		t.Fatal(err)
	}
	var env struct {
		OK   bool              `json:"ok"`
		Data sdk.ChangeContext `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if !env.OK || env.Data.Change.InputRevision != 2 || f.id != "change" {
		t.Fatalf("wrong envelope: %s", out.String())
	}
}
func TestContextFailureDoesNotEmitSuccess(t *testing.T) {
	out, _ := iostreams.SetForTest(t)
	err := runContext(context.Background(), &ContextOptions{}, &cmdutil.FormatOptions{Mode: cmdutil.FormatJSON}, &contextFake{err: errors.New("permission source unavailable")}, "change")
	if err == nil || out.Len() != 0 {
		t.Fatalf("failure emitted success: %v %s", err, out.String())
	}
}

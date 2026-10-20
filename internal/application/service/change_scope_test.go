package service

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	cloudclient "github.com/Tencent/WeKnora/internal/integration/taskflowclient"
	"github.com/Tencent/WeKnora/internal/types"
)

type scopedChangeKnowledge struct {
	documents  map[string]string
	workspaces map[string]uint64
}

func (k *scopedChangeKnowledge) GetKnowledgeByIDOnly(_ context.Context, id string) (*types.Knowledge, error) {
	kbID, ok := k.documents[id]
	if !ok {
		return nil, apperrors.NewNotFoundError("knowledge is unavailable")
	}
	return &types.Knowledge{ID: id, KnowledgeBaseID: kbID}, nil
}
func (k *scopedChangeKnowledge) GetKnowledgeBaseByIDOnly(_ context.Context, id string) (*types.KnowledgeBase, error) {
	return &types.KnowledgeBase{ID: id, TenantID: k.workspaces[id]}, nil
}

func TestChangeReadCapabilityDoesNotGrantWritesOrUnscopedSources(t *testing.T) {
	s, ctx, input := changeTestService()
	c, err := s.Create(ctx, CreateChangeRequest{Title: "Change", Input: input})
	if err != nil {
		t.Fatal(err)
	}
	readonly := types.WithTenantAPIKeyScope(ctx, types.TenantAPIKeyScope{ScopeType: types.APIKeyScopeTenant, Capabilities: types.StringArray{"retrieve"}})
	_, err = s.Context(readonly, c.ID)
	requireChangeStatus(t, err, http.StatusForbidden)
	readonly = types.WithTenantAPIKeyScope(ctx, types.TenantAPIKeyScope{ScopeType: types.APIKeyScopeTenant, Capabilities: types.StringArray{"read_changes"}})
	if _, err = s.Context(readonly, c.ID); err != nil {
		t.Fatal(err)
	}
	title := "no machine write"
	_, err = s.Patch(readonly, c.ID, PatchChangeRequest{ExpectedRevision: 1, Title: &title})
	requireChangeStatus(t, err, http.StatusForbidden)
	bounded := types.WithTenantAPIKeyScope(ctx, types.TenantAPIKeyScope{ScopeType: types.APIKeyScopeTenant, Capabilities: types.StringArray{"read_changes"}, KnowledgeBaseIDs: types.StringArray{"kb"}})
	_, err = s.Context(bounded, c.ID)
	requireChangeStatus(t, err, http.StatusForbidden)
}

func TestChangeRunAndArtifactAuthorizeTheAssociatedSnapshot(t *testing.T) {
	for _, history := range []bool{false, true} {
		for _, linked := range []bool{false, true} {
			t.Run(fmt.Sprintf("historical=%t/linked=%t", history, linked), func(t *testing.T) {
				s, ctx, input := changeTestService()
				knowledge := &scopedChangeKnowledge{documents: map[string]string{"doc-a": "kb-a", "doc-b": "kb-b"}, workspaces: map[string]uint64{"kb-a": 1, "kb-b": 1}}
				s.knowledge, s.kb = knowledge, knowledge
				if linked {
					input.References[0].KnowledgeID = "doc-b"
				}
				v := &testChangeVerifier{runs: map[string]*cloudclient.VerificationRun{}}
				s.verification = v
				c, err := s.Create(ctx, CreateChangeRequest{Title: "Change", Input: input})
				if err != nil {
					t.Fatal(err)
				}
				snapshot, err := s.Confirm(ctx, c.ID, c.Revision)
				if err != nil {
					t.Fatal(err)
				}
				runID := "1523d950-4e5c-49a1-8c0c-15f3929ba3e1"
				_, err = s.SubmitRun(ctx, c.ID, snapshot.ID, runID, 2)
				if err != nil {
					t.Fatal(err)
				}
				v.runs[runID].Attempts = []cloudclient.VerificationReport{{Generation: 1, Verdict: "passed", Checks: []cloudclient.VerificationCheck{{ID: "test", Required: true, Status: "passed", ArtifactID: "log"}}}}
				v.runs[runID].Status = "success"
				if history {
					input.References[0].KnowledgeID = "doc-a"
					c, err = s.Patch(ctx, c.ID, PatchChangeRequest{ExpectedRevision: 2, Input: &input})
					if err != nil {
						t.Fatal(err)
					}
					advice := "current advice only uses kb-a"
					c, err = s.PrepareDraft(ctx, c.ID, "", &advice, c.Revision)
					if err != nil {
						t.Fatal(err)
					}
				}
				bounded := types.WithTenantAPIKeyScope(ctx, types.TenantAPIKeyScope{ScopeType: types.APIKeyScopeTenant, Capabilities: types.StringArray{"read_changes"}, KnowledgeBaseIDs: types.StringArray{"kb-a"}})
				if history {
					if _, err = s.Context(bounded, c.ID); err != nil {
						t.Fatalf("current authorized context should remain readable: %v", err)
					}
				}
				getsBefore := v.gets
				_, err = s.Run(bounded, c.ID, runID)
				requireChangeStatus(t, err, http.StatusForbidden)
				_, err = s.Artifact(bounded, c.ID, runID, 1, "log")
				requireChangeStatus(t, err, http.StatusForbidden)
				if v.gets != getsBefore || v.artifacts != 0 {
					t.Fatal("out-of-scope snapshot reached the service credential transport")
				}
				views, err := s.Runs(bounded, c.ID)
				if err != nil || len(views) != 1 || views[0].Run != nil || views[0].ExecutionError == "" {
					t.Fatalf("history metadata lost its explicit access error: %+v %v", views, err)
				}
				if _, err = s.Run(ctx, c.ID, runID); err != nil {
					t.Fatalf("workspace member should read registered Git sources: %v", err)
				}
				if linked {
					delete(knowledge.documents, "doc-b")
					_, err = s.Run(ctx, c.ID, runID)
					requireChangeStatus(t, err, http.StatusNotFound)
					_, err = s.Artifact(ctx, c.ID, runID, 1, "log")
					requireChangeStatus(t, err, http.StatusNotFound)
					if !history {
						_, err = s.Handoff(ctx, c.ID, runID, "accepted", "self-verification", 2)
						requireChangeStatus(t, err, http.StatusNotFound)
					}
				}
			})
		}
	}
}

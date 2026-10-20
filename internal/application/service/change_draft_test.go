package service

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
)

type adviceModels struct {
	finishReason string
	calls        *int
}

func (m adviceModels) GetChatModel(_ context.Context, id string) (chat.Chat, error) {
	if id != "configured" {
		return nil, fmt.Errorf("model %s unavailable", id)
	}
	return adviceChat{finishReason: m.finishReason, calls: m.calls}, nil
}

type adviceChat struct {
	finishReason string
	calls        *int
}

func (m adviceChat) Chat(context.Context, []chat.Message, *chat.ChatOptions) (*types.ChatResponse, error) {
	if m.calls != nil {
		*m.calls++
	}
	return &types.ChatResponse{Content: "Review compatibility. A model-supplied nonexistent/path is only untrusted advice.", FinishReason: m.finishReason}, nil
}

func TestChangeAdviceRejectsLengthWithoutOverwritingDraft(t *testing.T) {
	for _, reason := range []string{"length", "stop"} {
		t.Run(reason, func(t *testing.T) {
			s, ctx, input := changeTestService()
			c, err := s.Create(ctx, CreateChangeRequest{Title: "Change", Input: input})
			if err != nil {
				t.Fatal(err)
			}
			original := "original developer-reviewed advice"
			c, err = s.PrepareDraft(ctx, c.ID, "", &original, c.Revision)
			if err != nil {
				t.Fatal(err)
			}
			before, err := s.repo.Get(ctx, c.TenantID, c.ID)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			s.models = adviceModels{finishReason: reason, calls: &calls}
			updated, err := s.PrepareDraft(ctx, c.ID, "configured", nil, c.Revision)
			if reason == "length" {
				requireChangeStatus(t, err, http.StatusServiceUnavailable)
				if updated != nil || !strings.Contains(err.Error(), "finish_reason=length") {
					t.Fatalf("truncation was not reported: %+v %v", updated, err)
				}
				if details := err.(*apperrors.AppError).Details; !reflect.DeepEqual(details, map[string]any{"finish_reason": "length", "max_tokens": changeAdviceMaxTokens}) {
					t.Fatalf("missing truncation cause or output budget: %+v", details)
				}
				after, readErr := s.repo.Get(ctx, c.TenantID, c.ID)
				if readErr != nil || !reflect.DeepEqual(before, after) {
					t.Fatalf("truncated advice changed the stored draft or revision: %+v %v", after, readErr)
				}
			} else {
				if err != nil || updated.Revision != c.Revision+1 || updated.InputRevision != c.InputRevision {
					t.Fatalf("normal stop was rejected: %+v %v", updated, err)
				}
				view, readErr := s.Context(ctx, c.ID)
				if readErr != nil || view.Change.Draft.ModelID != "configured" || view.Change.Draft.Content == original {
					t.Fatalf("normal stop did not save generated advice: %+v %v", view, readErr)
				}
			}
			if calls != 1 {
				t.Fatalf("model calls=%d; want one, without automatic retry", calls)
			}
		})
	}
}
func (adviceChat) ChatStream(context.Context, []chat.Message, *chat.ChatOptions) (<-chan types.StreamResponse, error) {
	return nil, fmt.Errorf("not used")
}
func (adviceChat) GetModelName() string { return "deterministic advice test" }
func (adviceChat) GetModelID() string   { return "configured" }
func TestChangeAdviceCannotConfirmOrAdoptReferences(t *testing.T) {
	s, ctx, input := changeTestService()
	c, _ := s.Create(ctx, CreateChangeRequest{Title: "Change", Input: input})
	_, err := s.PrepareDraft(ctx, c.ID, "", nil, 1)
	requireChangeStatus(t, err, http.StatusServiceUnavailable)
	s.models = adviceModels{}
	updated, err := s.PrepareDraft(ctx, c.ID, "configured", nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if updated.InputRevision != 1 || updated.Stage != "draft" || len(updated.Input.References) != 1 || updated.Input.References[0].Path != "docs/rules.md" || updated.Draft != nil {
		t.Fatal("model text changed adopted input or confirmation")
	}
	context, err := s.Context(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if context.CurrentSnapshotID != "" || len(context.Snapshots) != 0 {
		t.Fatal("model advice confirmed a snapshot")
	}
	if context.Change.Draft == nil || !strings.Contains(context.Change.Draft.Content, "nonexistent/path") {
		t.Fatal("authorized context did not return advice")
	}
	human := "edited by a developer"
	updated, err = s.PrepareDraft(ctx, c.ID, "configured", &human, 2)
	if err != nil {
		t.Fatal(err)
	}
	context, err = s.Context(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Draft != nil || context.Change.Draft.ModelID != "" || context.Change.Draft.Content != human || updated.InputRevision != 1 {
		t.Fatal("human edit attribution or applicability incorrect")
	}
}

func TestChangeRevokedDraftDoesNotLockMetadataOrRecovery(t *testing.T) {
	for _, cause := range []string{"revoked", "deleted"} {
		for _, recovery := range []string{"clear", "regenerate"} {
			t.Run(cause+"/"+recovery, func(t *testing.T) {
				s, ctx, input := changeTestService()
				input.References[0].KnowledgeID = "doc"
				s.models = adviceModels{}
				c, err := s.Create(ctx, CreateChangeRequest{Title: "Change", Input: input})
				if err != nil {
					t.Fatal(err)
				}
				c, err = s.PrepareDraft(ctx, c.ID, "configured", nil, c.Revision)
				if err != nil {
					t.Fatal(err)
				}
				knowledge := s.knowledge.(*testChangeKnowledge)
				status := http.StatusForbidden
				if cause == "revoked" {
					knowledge.tenant = 2
				} else {
					knowledge.knowledgeErr = apperrors.NewNotFoundError("knowledge was deleted")
					status = http.StatusNotFound
				}
				_, err = s.Context(ctx, c.ID)
				requireChangeStatus(t, err, status)
				metadata, err := s.Get(ctx, c.ID)
				if err != nil || metadata.Draft != nil {
					t.Fatalf("metadata was blocked or returned advice: %+v %v", metadata, err)
				}
				list, total, err := s.List(ctx, 1, 20)
				if err != nil || total != 1 || len(list) != 1 || list[0].Draft != nil {
					t.Fatalf("metadata list returned advice: %+v %v", list, err)
				}
				title := "Repaired title"
				c, err = s.Patch(ctx, c.ID, PatchChangeRequest{ExpectedRevision: c.Revision, Title: &title, Input: &input})
				if err != nil || c.Draft != nil || c.InputRevision != 1 {
					t.Fatalf("metadata edit was blocked or leaked advice: %+v %v", c, err)
				}
				// The invalid knowledge link is removed through metadata editing.
				input.References[0].KnowledgeID = ""
				c, err = s.Patch(ctx, c.ID, PatchChangeRequest{ExpectedRevision: c.Revision, Input: &input})
				if err != nil || c.Draft != nil || c.InputRevision != 2 {
					t.Fatalf("repair failed: %+v %v", c, err)
				}
				_, err = s.Context(ctx, c.ID)
				requireChangeStatus(t, err, status) // historical draft still has its original source boundary
				before := c.Revision
				if recovery == "clear" {
					empty := ""
					c, err = s.PrepareDraft(ctx, c.ID, "", &empty, before)
				} else {
					c, err = s.PrepareDraft(ctx, c.ID, "configured", nil, before)
				}
				if err != nil || c.Draft != nil || c.Revision != before+1 || c.InputRevision != 2 {
					t.Fatalf("draft recovery failed: %+v %v", c, err)
				}
				view, err := s.Context(ctx, c.ID)
				if err != nil {
					t.Fatal(err)
				}
				if recovery == "clear" && view.Change.Draft != nil {
					t.Fatal("clear retained old advice")
				}
				if recovery == "regenerate" && (view.Change.Draft == nil || view.Change.Draft.SourceInput.References[0].KnowledgeID != "") {
					t.Fatal("regeneration retained old source permissions")
				}
			})
		}
	}
}

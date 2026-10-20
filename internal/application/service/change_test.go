package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/config"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
)

type memoryChangeRepo struct {
	records   map[string]*types.ChangeRequest
	snapshots []*types.ChangeSnapshot
	links     map[string]*types.ChangeRunLink
	handoffs  []*types.ChangeHandoff
}

func (r *memoryChangeRepo) Snapshot(_ context.Context, tenant uint64, id, snapshotID string) (*types.ChangeSnapshot, error) {
	for _, s := range r.snapshots {
		if s.TenantID == tenant && s.ChangeID == id && s.ID == snapshotID {
			return s, nil
		}
	}
	return nil, apperrors.NewNotFoundError("snapshot not found")
}
func (r *memoryChangeRepo) CreateRunIntent(_ context.Context, link *types.ChangeRunLink, expected int64) error {
	if r.records[link.ChangeID].Revision != expected {
		return apperrors.NewConflictError("stale")
	}
	if r.links == nil {
		r.links = map[string]*types.ChangeRunLink{}
	}
	if r.links[link.RunID] == nil {
		r.links[link.RunID] = link
	}
	return nil
}
func (r *memoryChangeRepo) RunLink(_ context.Context, tenant uint64, id string) (*types.ChangeRunLink, error) {
	if l := r.links[id]; l != nil && l.TenantID == tenant {
		copy := *l
		return &copy, nil
	}
	return nil, apperrors.NewNotFoundError("run link not found")
}
func (r *memoryChangeRepo) RunLinks(_ context.Context, tenant uint64, id string) ([]*types.ChangeRunLink, error) {
	out := []*types.ChangeRunLink{}
	for _, l := range r.links {
		if l.TenantID == tenant && l.ChangeID == id {
			out = append(out, l)
		}
	}
	return out, nil
}
func (r *memoryChangeRepo) UpdateRunSubmission(_ context.Context, tenant uint64, id, state, reason string) error {
	l := r.links[id]
	if l == nil || l.TenantID != tenant {
		return apperrors.NewNotFoundError("link not found")
	}
	l.SubmissionState = state
	l.SubmissionError = reason
	return nil
}
func (r *memoryChangeRepo) Handoff(_ context.Context, h *types.ChangeHandoff, expected int64) error {
	c := r.records[h.ChangeID]
	if c.Revision != expected || c.Stage != "active" {
		return apperrors.NewConflictError("stale")
	}
	c.Revision++
	if h.Conclusion == "accepted" {
		c.Stage = "closed"
	}
	r.handoffs = append(r.handoffs, h)
	return nil
}
func (r *memoryChangeRepo) Handoffs(_ context.Context, tenant uint64, id string) ([]*types.ChangeHandoff, error) {
	out := []*types.ChangeHandoff{}
	for _, h := range r.handoffs {
		if h.TenantID == tenant && h.ChangeID == id {
			out = append(out, h)
		}
	}
	return out, nil
}

func copyChange(c *types.ChangeRequest) *types.ChangeRequest {
	out := new(types.ChangeRequest)
	b, _ := json.Marshal(c)
	_ = json.Unmarshal(b, out)
	return out
}
func (r *memoryChangeRepo) Create(_ context.Context, c *types.ChangeRequest) error {
	r.records[c.ID] = copyChange(c)
	return nil
}
func (r *memoryChangeRepo) Get(_ context.Context, tenant uint64, id string) (*types.ChangeRequest, error) {
	c := r.records[id]
	if c == nil || c.TenantID != tenant {
		return nil, apperrors.NewNotFoundError("change not found")
	}
	return copyChange(c), nil
}
func (r *memoryChangeRepo) List(_ context.Context, tenant uint64, offset, limit int) ([]*types.ChangeRequest, int64, error) {
	out := []*types.ChangeRequest{}
	for _, c := range r.records {
		if c.TenantID == tenant {
			out = append(out, copyChange(c))
		}
	}
	return out, int64(len(out)), nil
}
func (r *memoryChangeRepo) UpdateCAS(_ context.Context, c *types.ChangeRequest, expected int64) error {
	if r.records[c.ID].Revision != expected {
		return apperrors.NewConflictError("stale")
	}
	r.records[c.ID] = copyChange(c)
	return nil
}
func (r *memoryChangeRepo) Confirm(_ context.Context, s *types.ChangeSnapshot, expected int64) error {
	c := r.records[s.ChangeID]
	if c.Revision != expected {
		return apperrors.NewConflictError("stale")
	}
	for _, old := range r.snapshots {
		if old.ChangeID == s.ChangeID && old.InputRevision == s.InputRevision {
			return apperrors.NewConflictError("already confirmed")
		}
	}
	b, _ := json.Marshal(s)
	out := new(types.ChangeSnapshot)
	_ = json.Unmarshal(b, out)
	r.snapshots = append(r.snapshots, out)
	c.Revision++
	c.Stage = "active"
	return nil
}
func (r *memoryChangeRepo) Snapshots(_ context.Context, tenant uint64, id string) ([]*types.ChangeSnapshot, error) {
	out := []*types.ChangeSnapshot{}
	for _, s := range r.snapshots {
		if s.TenantID == tenant && s.ChangeID == id {
			out = append(out, s)
		}
	}
	return out, nil
}

type validChangeSources struct{ err error }

func (v validChangeSources) Commit(context.Context, string, string) error { return v.err }
func (v validChangeSources) Read(context.Context, string, types.ChangeReference) (string, error) {
	return "document at fixed commit", v.err
}

type testChangeKnowledge struct {
	tenant       uint64
	knowledgeErr error
}

func (k *testChangeKnowledge) GetKnowledgeByIDOnly(context.Context, string) (*types.Knowledge, error) {
	if k.knowledgeErr != nil {
		return nil, k.knowledgeErr
	}
	return &types.Knowledge{ID: "doc", KnowledgeBaseID: "kb"}, nil
}
func (k *testChangeKnowledge) GetKnowledgeBaseByIDOnly(context.Context, string) (*types.KnowledgeBase, error) {
	return &types.KnowledgeBase{ID: "kb", TenantID: k.tenant}, nil
}

func changeTestBool(v bool) *bool { return &v }

const testChangeCommit = "1111111111111111111111111111111111111111"

func changeTestService() (*ChangeService, context.Context, types.ChangeInput) {
	p := config.ChangeProjectConfig{ID: "demo", TenantID: 1, RepositoryPath: "registered", BusinessPackID: "pack", BusinessPackRevision: "1", CheckProfileID: "verify-change", CheckProfileRevision: "1", CheckIDs: []string{"test"}, RequiredReferencePaths: []string{"docs/rules.md"}}
	k := &testChangeKnowledge{tenant: 1}
	s := &ChangeService{cfg: &config.Config{Tenant: &config.TenantConfig{EnableRBAC: changeTestBool(true)}, ChangeWorkspace: &config.ChangeWorkspaceConfig{Enabled: true, Projects: []config.ChangeProjectConfig{p}}}, repo: &memoryChangeRepo{records: map[string]*types.ChangeRequest{}}, sources: validChangeSources{}, knowledge: k, kb: k}
	ctx := types.WithCaller(context.Background(), types.Caller{TenantID: 1, UserID: "creator", Role: types.TenantRoleContributor})
	input := types.ChangeInput{Objective: "add a business state", ProjectID: "demo", BusinessPackID: "pack", BusinessPackRevision: "1", BaseCommit: testChangeCommit, TargetCommit: testChangeCommit, CheckPlan: types.ChangeCheckPlan{ProfileID: "verify-change", ProfileRevision: "1", CheckIDs: []string{"test"}}, References: []types.ChangeReference{{RepositoryID: "demo", Commit: testChangeCommit, Path: "docs/rules.md"}}}
	return s, ctx, input
}
func requireChangeStatus(t *testing.T, err error, status int) {
	t.Helper()
	var app *apperrors.AppError
	if !errors.As(err, &app) || app.HTTPCode != status {
		t.Fatalf("want HTTP %d; got %v", status, err)
	}
}
func TestChangeAuthorizationAndRevisions(t *testing.T) {
	s, ctx, input := changeTestService()
	c, err := s.Create(ctx, CreateChangeRequest{Title: "Change", Input: input})
	if err != nil {
		t.Fatal(err)
	}
	other := types.WithCaller(ctx, types.Caller{TenantID: 2, UserID: "creator", Role: types.TenantRoleOwner})
	_, err = s.Get(other, c.ID)
	requireChangeStatus(t, err, http.StatusNotFound)
	colleague := types.WithCaller(ctx, types.Caller{TenantID: 1, UserID: "colleague", Role: types.TenantRoleContributor})
	title := "Retitled"
	_, err = s.Patch(colleague, c.ID, PatchChangeRequest{ExpectedRevision: 1, Title: &title})
	requireChangeStatus(t, err, http.StatusForbidden)
	viewer := types.WithCaller(ctx, types.Caller{TenantID: 1, UserID: "creator", Role: types.TenantRoleViewer})
	_, err = s.Patch(viewer, c.ID, PatchChangeRequest{ExpectedRevision: 1, Title: &title})
	requireChangeStatus(t, err, http.StatusForbidden)
	updated, err := s.Patch(ctx, c.ID, PatchChangeRequest{ExpectedRevision: 1, Title: &title})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != 2 || updated.InputRevision != 1 {
		t.Fatalf("metadata changed applicability: %+v", updated)
	}
	_, err = s.Patch(ctx, c.ID, PatchChangeRequest{ExpectedRevision: 1, Title: &title})
	requireChangeStatus(t, err, http.StatusConflict)
	input.Objective = "different acceptance objective"
	updated, err = s.Patch(ctx, c.ID, PatchChangeRequest{ExpectedRevision: 2, Input: &input})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != 3 || updated.InputRevision != 2 {
		t.Fatal("verification input did not increment input_revision")
	}
	s.cfg.Tenant.EnableRBAC = changeTestBool(false)
	_, err = s.Patch(ctx, c.ID, PatchChangeRequest{ExpectedRevision: 3, Title: &title})
	requireChangeStatus(t, err, http.StatusForbidden)
}
func TestChangeConfirmationIsImmutableAndVersionBound(t *testing.T) {
	s, ctx, input := changeTestService()
	c, err := s.Create(ctx, CreateChangeRequest{Title: "Change", Input: input})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := s.Confirm(ctx, c.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Context(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.CurrentSnapshotID != snapshot.ID || result.Change.Stage != "active" || result.Change.Revision != 2 {
		t.Fatalf("bad confirmation: %+v", result)
	}
	title := "Retitled"
	if _, err = s.Patch(ctx, c.ID, PatchChangeRequest{ExpectedRevision: 2, Title: &title}); err != nil {
		t.Fatal(err)
	}
	result, err = s.Context(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.CurrentSnapshotID != snapshot.ID {
		t.Fatal("title edit invalidated snapshot")
	}
	input.TargetCommit = strings.Repeat("2", 40)
	if _, err = s.Patch(ctx, c.ID, PatchChangeRequest{ExpectedRevision: 3, Input: &input}); err != nil {
		t.Fatal(err)
	}
	result, err = s.Context(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.CurrentSnapshotID != "" {
		t.Fatal("old snapshot marked current")
	}
	if result.Snapshots[0].Input.TargetCommit != testChangeCommit {
		t.Fatal("history overwritten")
	}
	_, err = s.Confirm(ctx, c.ID, 3)
	requireChangeStatus(t, err, http.StatusConflict)
}
func TestChangeMissingEvidenceAndRevokedKnowledge(t *testing.T) {
	s, ctx, input := changeTestService()
	input.References = nil
	c, err := s.Create(ctx, CreateChangeRequest{Title: "Change", Input: input})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Confirm(ctx, c.ID, 1)
	requireChangeStatus(t, err, http.StatusBadRequest)
	input.References = []types.ChangeReference{{RepositoryID: "demo", Commit: testChangeCommit, Path: "docs/rules.md", KnowledgeID: "doc"}}
	c, err = s.Patch(ctx, c.ID, PatchChangeRequest{ExpectedRevision: 1, Input: &input})
	if err != nil {
		t.Fatal(err)
	}
	k := s.kb.(*testChangeKnowledge)
	k.tenant = 2
	_, err = s.Context(ctx, c.ID)
	requireChangeStatus(t, err, http.StatusForbidden)
	_, err = s.Confirm(ctx, c.ID, c.Revision)
	requireChangeStatus(t, err, http.StatusForbidden)
}
func TestGitChangeSourcesReadCommittedVersion(t *testing.T) {
	repo := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init")
	if err := os.WriteFile(filepath.Join(repo, "rules.md"), []byte("first\nsecond\nthird\n"), 0600); err != nil {
		t.Fatal(err)
	}
	git("add", "rules.md")
	git("-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-m", "source baseline")
	commit := git("rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(repo, "rules.md"), []byte("uncommitted changed source"), 0600); err != nil {
		t.Fatal(err)
	}
	s := gitChangeSources{}
	ref := types.ChangeReference{Commit: commit, Path: "rules.md", StartLine: 2, EndLine: 2}
	text, err := s.Read(context.Background(), repo, ref)
	if err != nil {
		t.Fatal(err)
	}
	if text != "second" {
		t.Fatalf("read working tree or wrong range: %q", text)
	}
	for _, p := range []string{"../rules.md", "/rules.md", "a\\rules.md", "a/../rules.md"} {
		ref.Path = p
		if _, err = s.Read(context.Background(), repo, ref); err == nil {
			t.Fatalf("accepted path %q", p)
		}
	}
}

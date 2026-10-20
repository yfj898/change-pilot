package service

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/application/access"
	"github.com/Tencent/WeKnora/internal/config"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
)

type changeKnowledgeLookup interface {
	GetKnowledgeByIDOnly(context.Context, string) (*types.Knowledge, error)
}
type changeKBLookup interface {
	GetKnowledgeBaseByIDOnly(context.Context, string) (*types.KnowledgeBase, error)
}

// ChangeService owns authorization, input versions, and confirmation invariants.
type ChangeService struct {
	cfg          *config.Config
	repo         interfaces.ChangeRepository
	knowledge    changeKnowledgeLookup
	kb           changeKBLookup
	sources      changeSourceReader
	verification changeVerifier
	models       changeModelLookup
}

type changeModelLookup interface {
	GetChatModel(context.Context, string) (chat.Chat, error)
}

// NewChangeService adds the workflow without replacing knowledge or identity services.
func NewChangeService(cfg *config.Config, repo interfaces.ChangeRepository, knowledge interfaces.KnowledgeService, kb interfaces.KnowledgeBaseService, models interfaces.ModelService) *ChangeService {
	return &ChangeService{cfg: cfg, repo: repo, knowledge: knowledge, kb: kb, sources: gitChangeSources{}, models: models}
}

// CreateChangeRequest excludes ownership and workspace fields: auth supplies them.
type CreateChangeRequest struct {
	Title      string            `json:"title"`
	AssigneeID string            `json:"assignee_id"`
	Notes      string            `json:"notes"`
	Input      types.ChangeInput `json:"input"`
}

// PatchChangeRequest separates edit concurrency from verification applicability.
type PatchChangeRequest struct {
	ExpectedRevision int64              `json:"expected_revision"`
	Title            *string            `json:"title,omitempty"`
	AssigneeID       *string            `json:"assignee_id,omitempty"`
	Notes            *string            `json:"notes,omitempty"`
	Input            *types.ChangeInput `json:"input,omitempty"`
}

func (s *ChangeService) caller(ctx context.Context, write bool) (types.Caller, error) {
	if s.cfg.ChangeWorkspace == nil || !s.cfg.ChangeWorkspace.Enabled {
		return types.Caller{}, apperrors.NewServiceUnavailableError("change workspace is disabled")
	}
	c := types.CallerFromContext(ctx)
	if scope, ok := types.TenantAPIKeyScopeFromContext(ctx); ok {
		if c.TenantID == 0 {
			return c, apperrors.NewUnauthorizedError("workspace identity is required")
		}
		if write || (!scope.FullAccess && !scope.HasCapability(types.APIKeyCapabilityReadChanges)) {
			return c, apperrors.NewForbiddenError("an explicit read_changes capability is required; machine writes are unavailable")
		}
		return c, nil
	}
	if c.TenantID == 0 || c.UserID == "" || types.IsSyntheticUserID(c.UserID) {
		return c, apperrors.NewUnauthorizedError("an authenticated workspace member is required")
	}
	if !c.Role.HasPermission(types.TenantRoleViewer) {
		return c, apperrors.NewForbiddenError("workspace viewer role is required")
	}
	if write {
		if s.cfg.Tenant == nil || !s.cfg.Tenant.IsRBACEnforced() {
			return c, apperrors.NewForbiddenError("enable enforced workspace RBAC before using change writes")
		}
		if !c.Role.HasPermission(types.TenantRoleContributor) {
			return c, apperrors.NewForbiddenError("workspace contributor role is required")
		}
	}
	return c, nil
}

func (s *ChangeService) editable(ctx context.Context, id string) (*types.ChangeRequest, types.Caller, error) {
	c, err := s.caller(ctx, true)
	if err != nil {
		return nil, c, err
	}
	change, err := s.repo.Get(ctx, c.TenantID, id)
	if err != nil {
		return nil, c, err
	}
	if change.CreatorID != c.UserID && !c.Role.HasPermission(types.TenantRoleAdmin) {
		return nil, c, apperrors.NewForbiddenError("only the creator or workspace admin may modify this change")
	}
	if change.Stage == "closed" {
		return nil, c, apperrors.NewConflictError("closed changes are immutable; create a follow-up change")
	}
	return change, c, nil
}

func (s *ChangeService) project(c types.Caller, input types.ChangeInput) (config.ChangeProjectConfig, error) {
	for _, p := range s.cfg.ChangeWorkspace.Projects {
		if p.ID == input.ProjectID && p.BusinessPackID == input.BusinessPackID && p.TenantID == c.TenantID {
			if p.BusinessPackRevision != input.BusinessPackRevision {
				return p, apperrors.NewConflictError("business pack version is not registered")
			}
			return p, nil
		}
	}
	return config.ChangeProjectConfig{}, apperrors.NewForbiddenError("project and business pack are not registered for this workspace")
}

// Projects returns only public mapping fields registered for this workspace.
func (s *ChangeService) Projects(ctx context.Context) ([]config.ChangeProjectConfig, error) {
	c, err := s.caller(ctx, false)
	if err != nil {
		return nil, err
	}
	projects := []config.ChangeProjectConfig{}
	for _, p := range s.cfg.ChangeWorkspace.Projects {
		if p.TenantID == c.TenantID {
			projects = append(projects, p)
		}
	}
	return projects, nil
}

// Create starts a draft; a draft may omit commits and evidence until confirmation.
func (s *ChangeService) Create(ctx context.Context, req CreateChangeRequest) (*types.ChangeRequest, error) {
	c, err := s.caller(ctx, true)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Title) == "" {
		return nil, apperrors.NewBadRequestError("title is required")
	}
	p, err := s.project(c, req.Input)
	if err != nil {
		return nil, err
	}
	if req.Input.CheckPlan.ProfileID == "" {
		req.Input.CheckPlan = types.ChangeCheckPlan{ProfileID: p.CheckProfileID, ProfileRevision: p.CheckProfileRevision, CheckIDs: append([]string{}, p.CheckIDs...)}
	}
	if err := s.validateInput(ctx, c, p, req.Input, false); err != nil {
		return nil, err
	}
	change := &types.ChangeRequest{ID: uuid.NewString(), TenantID: c.TenantID, CreatorID: c.UserID, Title: req.Title, AssigneeID: req.AssigneeID, Notes: req.Notes, Input: req.Input, Revision: 1, InputRevision: 1, Stage: "draft", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := s.repo.Create(ctx, change); err != nil {
		return nil, err
	}
	return change, nil
}

// changeMetadata excludes advice derived from source documents. Reading that
// body belongs to Context, which reauthorizes the draft's historical sources.
func changeMetadata(change *types.ChangeRequest) *types.ChangeRequest {
	metadata := *change
	metadata.Draft = nil
	return &metadata
}

// Get and List expose metadata only inside the caller's workspace. Unavailable
// historical advice must not prevent a creator from repairing the current input.
func (s *ChangeService) Get(ctx context.Context, id string) (*types.ChangeRequest, error) {
	c, err := s.caller(ctx, false)
	if err != nil {
		return nil, err
	}
	change, err := s.repo.Get(ctx, c.TenantID, id)
	if err != nil {
		return nil, err
	}
	return changeMetadata(change), nil
}
func (s *ChangeService) List(ctx context.Context, page, pageSize int) ([]*types.ChangeRequest, int64, error) {
	c, err := s.caller(ctx, false)
	if err != nil {
		return nil, 0, err
	}
	changes, total, err := s.repo.List(ctx, c.TenantID, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, err
	}
	for i, change := range changes {
		changes[i] = changeMetadata(change)
	}
	return changes, total, nil
}

// Patch never overwrites a stale edit and invalidates checks only for input changes.
func (s *ChangeService) Patch(ctx context.Context, id string, req PatchChangeRequest) (*types.ChangeRequest, error) {
	change, c, err := s.editable(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.ExpectedRevision != change.Revision {
		return nil, apperrors.NewConflictError("change revision is stale; reload before saving").WithDetails(map[string]any{"current_revision": change.Revision})
	}
	if req.Title == nil && req.AssigneeID == nil && req.Notes == nil && req.Input == nil {
		return nil, apperrors.NewBadRequestError("no fields to update")
	}
	if req.Title != nil {
		if strings.TrimSpace(*req.Title) == "" {
			return nil, apperrors.NewBadRequestError("title is required")
		}
		change.Title = *req.Title
	}
	if req.AssigneeID != nil {
		change.AssigneeID = *req.AssigneeID
	}
	if req.Notes != nil {
		change.Notes = *req.Notes
	}
	if req.Input != nil && !reflect.DeepEqual(change.Input, *req.Input) {
		p, err := s.project(c, *req.Input)
		if err != nil {
			return nil, err
		}
		if err := s.validateInput(ctx, c, p, *req.Input, false); err != nil {
			return nil, err
		}
		change.InputRevision++
		change.Input = *req.Input
	}
	change.Revision++
	change.UpdatedAt = time.Now()
	if err := s.repo.UpdateCAS(ctx, change, req.ExpectedRevision); err != nil {
		return nil, err
	}
	return changeMetadata(change), nil
}

// Confirm records only sources reread from fixed commits and an allowed check plan.
func (s *ChangeService) Confirm(ctx context.Context, id string, expected int64) (*types.ChangeSnapshot, error) {
	change, c, err := s.editable(ctx, id)
	if err != nil {
		return nil, err
	}
	if expected != change.Revision {
		return nil, apperrors.NewConflictError("change revision is stale; reload before confirming")
	}
	p, err := s.project(c, change.Input)
	if err != nil {
		return nil, err
	}
	if err := s.validateInput(ctx, c, p, change.Input, true); err != nil {
		return nil, err
	}
	snapshot := &types.ChangeSnapshot{ID: uuid.NewString(), TenantID: c.TenantID, ChangeID: id, Revision: change.Revision, InputRevision: change.InputRevision, Input: change.Input, ConfirmedBy: c.UserID, CreatedAt: time.Now()}
	if err := s.repo.Confirm(ctx, snapshot, expected); err != nil {
		return nil, err
	}
	return snapshot, nil
}

// Context resolves sources afresh, including knowledge authorization after revocation.
func (s *ChangeService) Context(ctx context.Context, id string) (*types.ChangeContext, error) {
	c, err := s.caller(ctx, false)
	if err != nil {
		return nil, err
	}
	change, err := s.repo.Get(ctx, c.TenantID, id)
	if err != nil {
		return nil, err
	}
	sources, err := s.sourcesForInput(ctx, c, change.Input)
	if err != nil {
		return nil, err
	}
	if change.Draft != nil {
		if _, err := s.sourcesForInput(ctx, c, change.Draft.SourceInput); err != nil {
			return nil, err
		}
	}
	result := &types.ChangeContext{Change: change, Sources: sources}
	result.Snapshots, err = s.repo.Snapshots(ctx, c.TenantID, id)
	if err != nil {
		return nil, err
	}
	for _, snapshot := range result.Snapshots {
		if snapshot.InputRevision == change.InputRevision {
			result.CurrentSnapshotID = snapshot.ID
			break
		}
	}
	return result, nil
}

func (s *ChangeService) sourcesForInput(ctx context.Context, c types.Caller, input types.ChangeInput) ([]types.ResolvedChangeReference, error) {
	p, err := s.project(c, input)
	if err != nil {
		return nil, err
	}
	sources := []types.ResolvedChangeReference{}
	for _, ref := range input.References {
		content, err := s.resolveReference(ctx, c, p, ref)
		if err != nil {
			return nil, err
		}
		sources = append(sources, types.ResolvedChangeReference{Reference: ref, Content: content})
	}
	return sources, nil
}

var changeCommitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

func (s *ChangeService) validateInput(ctx context.Context, c types.Caller, p config.ChangeProjectConfig, input types.ChangeInput, confirm bool) error {
	if strings.TrimSpace(input.Objective) == "" {
		return apperrors.NewBadRequestError("objective is required")
	}
	for _, commit := range []string{input.BaseCommit, input.TargetCommit} {
		if commit != "" && !changeCommitPattern.MatchString(commit) {
			return apperrors.NewBadRequestError("commits must be full lowercase 40-character Git commit IDs")
		}
	}
	if confirm {
		if len(input.References) == 0 {
			return apperrors.NewBadRequestError("at least one versioned source is required before confirmation")
		}
		if input.BaseCommit == "" || input.TargetCommit == "" {
			return apperrors.NewBadRequestError("base and target commit are required before confirmation")
		}
		if err := s.sources.Commit(ctx, p.RepositoryPath, input.BaseCommit); err != nil {
			return apperrors.NewBadRequestError(fmt.Sprintf("base commit is unavailable: %v", err))
		}
		if err := s.sources.Commit(ctx, p.RepositoryPath, input.TargetCommit); err != nil {
			return apperrors.NewBadRequestError(fmt.Sprintf("target commit is unavailable: %v", err))
		}
	}
	plan := input.CheckPlan
	if plan.ProfileID != p.CheckProfileID || plan.ProfileRevision != p.CheckProfileRevision {
		return apperrors.NewBadRequestError("check profile/version must match the registered project")
	}
	seen := map[string]bool{}
	for _, id := range plan.CheckIDs {
		if seen[id] {
			return apperrors.NewBadRequestError("duplicate check ID")
		}
		seen[id] = true
		found := false
		for _, allowed := range p.CheckIDs {
			if id == allowed {
				found = true
				break
			}
		}
		if !found {
			return apperrors.NewBadRequestError("check ID is not registered for this project: " + id)
		}
	}
	for _, id := range p.CheckIDs {
		if !seen[id] {
			return apperrors.NewBadRequestError("required check is missing: " + id)
		}
	}
	paths := map[string]bool{}
	for _, ref := range input.References {
		if _, err := s.resolveReference(ctx, c, p, ref); err != nil {
			return err
		}
		paths[ref.Path] = true
	}
	if confirm {
		for _, path := range p.RequiredReferencePaths {
			if !paths[path] {
				return apperrors.NewBadRequestError("required versioned source is missing: " + path)
			}
		}
	}
	return nil
}

func (s *ChangeService) resolveReference(ctx context.Context, c types.Caller, p config.ChangeProjectConfig, ref types.ChangeReference) (string, error) {
	if scope, ok := types.TenantAPIKeyScopeFromContext(ctx); ok && scope.IsKnowledgeBaseRestricted() && ref.KnowledgeID == "" {
		return "", apperrors.NewForbiddenError("KB-restricted consumers require a linked authorized knowledge document for every source")
	}
	if ref.RepositoryID != p.ID {
		return "", apperrors.NewForbiddenError("reference repository is outside the registered project")
	}
	if !changeCommitPattern.MatchString(ref.Commit) {
		return "", apperrors.NewBadRequestError("reference requires a full Git commit ID")
	}
	if ref.KnowledgeID != "" {
		knowledge, err := s.knowledge.GetKnowledgeByIDOnly(ctx, ref.KnowledgeID)
		if err != nil {
			return "", err
		}
		kb, err := s.kb.GetKnowledgeBaseByIDOnly(ctx, knowledge.KnowledgeBaseID)
		if err != nil {
			return "", err
		}
		if kb.TenantID != c.TenantID {
			return "", apperrors.NewForbiddenError("cross-workspace references are not supported")
		}
		if _, err := access.ResolveKB(ctx, access.KBRequest{Caller: c}, kb, types.OrgRoleViewer, nil, nil); err != nil {
			return "", apperrors.NewForbiddenError(err.Error())
		}
	}
	content, err := s.sources.Read(ctx, p.RepositoryPath, ref)
	if err != nil {
		return "", apperrors.NewBadRequestError(fmt.Sprintf("source %s@%s is unavailable: %v", ref.Path, ref.Commit, err))
	}
	return content, nil
}

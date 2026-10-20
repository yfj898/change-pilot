package client

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// ChangeReference identifies an exact, reviewable source in a registered repository.
// Content is resolved on read, never persisted as an authorization-bypassing excerpt.
type ChangeReference struct {
	RepositoryID string `json:"repository_id"`
	Commit       string `json:"commit"`
	Path         string `json:"path"`
	StartLine    int    `json:"start_line,omitempty"`
	EndLine      int    `json:"end_line,omitempty"`
	KnowledgeID  string `json:"knowledge_id,omitempty"`
}

// ChangeCheckPlan contains registered check identifiers, not executable commands.
type ChangeCheckPlan struct {
	ProfileID       string   `json:"profile_id"`
	ProfileRevision string   `json:"profile_revision"`
	CheckIDs        []string `json:"check_ids"`
}

// ChangeInput is the verification-relevant portion of a change.
type ChangeInput struct {
	Objective            string            `json:"objective"`
	ProjectID            string            `json:"project_id"`
	BusinessPackID       string            `json:"business_pack_id"`
	BusinessPackRevision string            `json:"business_pack_revision"`
	BaseCommit           string            `json:"base_commit"`
	TargetCommit         string            `json:"target_commit"`
	References           []ChangeReference `json:"references"`
	CheckPlan            ChangeCheckPlan   `json:"check_plan"`
}

// ChangeRequest keeps edit concurrency separate from verification applicability.
type ChangeRequest struct {
	ID            string      `json:"id"`
	TenantID      uint64      `json:"tenant_id"`
	CreatorID     string      `json:"creator_id"`
	Title         string      `json:"title"`
	AssigneeID    string      `json:"assignee_id"`
	Notes         string      `json:"notes"`
	Input         ChangeInput `json:"input"`
	Revision      int64       `json:"revision"`
	InputRevision int64       `json:"input_revision"`
	// Draft is included only in the authorized Context response, never metadata or edit responses.
	Draft     *ChangeDraft `json:"draft,omitempty"`
	Stage     string       `json:"stage"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// ChangeSnapshot is immutable after confirmation.
type ChangeSnapshot struct {
	ID            string      `json:"id"`
	TenantID      uint64      `json:"tenant_id"`
	ChangeID      string      `json:"change_id"`
	Revision      int64       `json:"revision"`
	InputRevision int64       `json:"input_revision"`
	Input         ChangeInput `json:"input"`
	ConfirmedBy   string      `json:"confirmed_by"`
	CreatedAt     time.Time   `json:"created_at"`
}

// ChangeDraft is editable advice, never adopted evidence or an execution instruction.
type ChangeDraft struct {
	Content       string      `json:"content"`
	ModelID       string      `json:"model_id,omitempty"`
	InputRevision int64       `json:"input_revision"`
	SourceInput   ChangeInput `json:"source_input"`
	AuthorID      string      `json:"author_id"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

// ResolvedChangeReference returns source read from the exact Git object.
type ResolvedChangeReference struct {
	Reference ChangeReference `json:"reference"`
	Content   string          `json:"content"`
}

// ChangeContext is shared by the web UI, CLI, and read-only tools.
type ChangeContext struct {
	Change            *ChangeRequest            `json:"change"`
	Sources           []ResolvedChangeReference `json:"sources"`
	Snapshots         []*ChangeSnapshot         `json:"snapshots"`
	CurrentSnapshotID string                    `json:"current_snapshot_id,omitempty"`
}

// ChangeRunLink records submission intent; execution facts belong to TaskFlow.
type ChangeRunLink struct {
	RunID           string    `json:"run_id"`
	TenantID        uint64    `json:"tenant_id"`
	ChangeID        string    `json:"change_id"`
	SnapshotID      string    `json:"snapshot_id"`
	SubmittedBy     string    `json:"submitted_by"`
	SubmissionState string    `json:"submission_state"`
	SubmissionError string    `json:"submission_error,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ChangeHandoff preserves the human conclusion separately from machine results.
type ChangeHandoff struct {
	ID         string    `json:"id"`
	TenantID   uint64    `json:"tenant_id"`
	ChangeID   string    `json:"change_id"`
	SnapshotID string    `json:"snapshot_id"`
	RunID      string    `json:"run_id"`
	Conclusion string    `json:"conclusion"`
	Notes      string    `json:"notes"`
	ActorID    string    `json:"actor_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type VerificationRequest struct {
	RunID                string   `json:"run_id"`
	ProjectID            string   `json:"project_id"`
	TargetCommit         string   `json:"target_commit"`
	CheckProfileID       string   `json:"check_profile_id"`
	CheckProfileRevision string   `json:"check_profile_revision"`
	CheckIDs             []string `json:"check_ids"`
	ExternalSnapshotID   string   `json:"external_snapshot_id"`
}
type VerificationCheck struct {
	ID         string `json:"id"`
	Required   bool   `json:"required"`
	Status     string `json:"status"`
	ExitCode   *int   `json:"exit_code,omitempty"`
	ArtifactID string `json:"artifact_id,omitempty"`
	Error      string `json:"error,omitempty"`
}
type VerificationReport struct {
	Generation uint64              `json:"generation"`
	StartedAt  time.Time           `json:"started_at"`
	FinishedAt *time.Time          `json:"finished_at,omitempty"`
	Verdict    string              `json:"verdict"`
	Error      string              `json:"error,omitempty"`
	Checks     []VerificationCheck `json:"checks"`
}
type VerificationRun struct {
	VerificationRequest
	TaskID     uint64               `json:"task_id"`
	Status     string               `json:"status"`
	Generation uint64               `json:"generation"`
	LastError  string               `json:"last_error,omitempty"`
	Attempts   []VerificationReport `json:"attempts"`
	CreatedAt  time.Time            `json:"created_at"`
}

// ChangeRunView combines submission intent with authoritative TaskFlow facts.
type ChangeRunView struct {
	Link           *ChangeRunLink   `json:"link"`
	Run            *VerificationRun `json:"run,omitempty"`
	Applicable     bool             `json:"applicable"`
	ExecutionError string           `json:"execution_error,omitempty"`
}

// GetChangeContext reads one authorized change and its versioned source content.
func (c *Client) GetChangeContext(ctx context.Context, id string) (*ChangeContext, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1/changes/"+url.PathEscape(id)+"/context", nil, nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		Success bool          `json:"success"`
		Data    ChangeContext `json:"data"`
	}
	if err := parseResponse(resp, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

// GetChangeVerificationRun reads an existing run without submitting or retrying work.
func (c *Client) GetChangeVerificationRun(ctx context.Context, changeID, runID string) (*ChangeRunView, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1/changes/"+url.PathEscape(changeID)+"/runs/"+url.PathEscape(runID), nil, nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		Success bool          `json:"success"`
		Data    ChangeRunView `json:"data"`
	}
	if err := parseResponse(resp, &out); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

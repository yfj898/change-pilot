package service

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	cloudclient "github.com/Tencent/WeKnora/internal/integration/taskflowclient"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
)

type changeVerifier interface {
	SubmitVerification(context.Context, cloudclient.VerificationRequest) (*cloudclient.VerificationRun, error)
	GetVerificationRun(context.Context, string) (*cloudclient.VerificationRun, error)
	GetVerificationArtifact(context.Context, string, uint64, string) ([]byte, error)
}

// ChangeRunView joins local submission intent with authoritative execution facts.
type ChangeRunView struct {
	Link           *types.ChangeRunLink         `json:"link"`
	Run            *cloudclient.VerificationRun `json:"run,omitempty"`
	Applicable     bool                         `json:"applicable"`
	ExecutionError string                       `json:"execution_error,omitempty"`
}

func (s *ChangeService) verifier() (changeVerifier, error) {
	if s.verification != nil {
		return s.verification, nil
	}
	cfg := s.cfg.ChangeWorkspace
	if cfg.TaskFlowEndpoint == "" || cfg.TaskFlowTokenEnv == "" {
		return nil, apperrors.NewServiceUnavailableError("TaskFlow endpoint and token environment name are not configured")
	}
	client, err := cloudclient.New(cfg.TaskFlowEndpoint, func(context.Context) (string, error) {
		token := os.Getenv(cfg.TaskFlowTokenEnv)
		if strings.TrimSpace(token) == "" {
			return "", fmt.Errorf("TaskFlow credential environment is empty: %s", cfg.TaskFlowTokenEnv)
		}
		return token, nil
	}, nil)
	if err != nil {
		return nil, apperrors.NewServiceUnavailableError(err.Error())
	}
	return client, nil
}
func changeNotFound(err error) bool {
	var app *apperrors.AppError
	return stderrors.As(err, &app) && app.HTTPCode == http.StatusNotFound
}
func taskflowMissing(err error) bool {
	var api *cloudclient.APIError
	return stderrors.As(err, &api) && api.StatusCode == http.StatusNotFound
}
func executionRequest(snapshot *types.ChangeSnapshot, runID string) cloudclient.VerificationRequest {
	in := snapshot.Input
	return cloudclient.VerificationRequest{RunID: runID, ProjectID: in.ProjectID, TargetCommit: in.TargetCommit, CheckProfileID: in.CheckPlan.ProfileID, CheckProfileRevision: in.CheckPlan.ProfileRevision, CheckIDs: in.CheckPlan.CheckIDs, ExternalSnapshotID: snapshot.ID}
}
func runMatches(run *cloudclient.VerificationRun, snapshot *types.ChangeSnapshot, runID string) bool {
	want := executionRequest(snapshot, runID)
	return run != nil && run.RunID == want.RunID && run.ProjectID == want.ProjectID && run.TargetCommit == want.TargetCommit && run.CheckProfileID == want.CheckProfileID && run.CheckProfileRevision == want.CheckProfileRevision && sameChangeChecks(run.CheckIDs, want.CheckIDs) && run.ExternalSnapshotID == want.ExternalSnapshotID
}

func sameChangeChecks(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[string]bool{}
	for _, id := range a {
		seen[id] = true
	}
	for _, id := range b {
		if !seen[id] {
			return false
		}
	}
	return true
}

// SubmitRun persists intent before HTTP; the same UUID resolves unknown outcomes.
func (s *ChangeService) SubmitRun(ctx context.Context, id, snapshotID, runID string, expected int64) (*ChangeRunView, error) {
	change, c, err := s.editable(ctx, id)
	if err != nil {
		return nil, err
	}
	parsed, err := uuid.Parse(runID)
	if err != nil || parsed.String() != runID {
		return nil, apperrors.NewBadRequestError("run_id must be a canonical UUID generated before submission")
	}
	snapshot, err := s.repo.Snapshot(ctx, c.TenantID, id, snapshotID)
	if err != nil {
		return nil, err
	}
	if _, err := s.sourcesForInput(ctx, c, snapshot.Input); err != nil {
		return nil, err
	}
	client, err := s.verifier()
	if err != nil {
		return nil, err
	}
	link, err := s.repo.RunLink(ctx, c.TenantID, runID)
	if err != nil && !changeNotFound(err) {
		return nil, err
	}
	if link != nil {
		if link.ChangeID != id || link.SnapshotID != snapshotID {
			return nil, apperrors.NewConflictError("run UUID already belongs to another change or snapshot")
		}
		actual, getErr := client.GetVerificationRun(ctx, runID)
		if getErr == nil {
			return s.recordAccepted(ctx, change, snapshot, link, actual)
		}
		if !taskflowMissing(getErr) {
			return nil, getErr
		}
	} else {
		if change.Revision != expected || snapshot.InputRevision != change.InputRevision {
			return nil, apperrors.NewConflictError("confirm the current inputs before creating a run")
		}
		p, err := s.project(c, snapshot.Input)
		if err != nil {
			return nil, err
		}
		if err = s.validateInput(ctx, c, p, snapshot.Input, true); err != nil {
			return nil, err
		}
		link = &types.ChangeRunLink{RunID: runID, TenantID: c.TenantID, ChangeID: id, SnapshotID: snapshotID, SubmittedBy: c.UserID, SubmissionState: "pending", CreatedAt: time.Now(), UpdatedAt: time.Now()}
		if err := s.repo.CreateRunIntent(ctx, link, expected); err != nil {
			return nil, err
		}
		persisted, err := s.repo.RunLink(ctx, c.TenantID, runID)
		if err != nil {
			return nil, err
		}
		link = persisted
		if link.ChangeID != id || link.SnapshotID != snapshotID {
			return nil, apperrors.NewConflictError("run UUID is already used")
		}
	}
	if snapshot.InputRevision != change.InputRevision {
		return nil, apperrors.NewConflictError("unaccepted historical input cannot start new execution; create a run for the current snapshot")
	}
	actual, submitErr := client.SubmitVerification(ctx, executionRequest(snapshot, runID))
	if submitErr != nil {
		state := "unknown"
		var api *cloudclient.APIError
		if stderrors.As(submitErr, &api) && api.StatusCode >= 400 && api.StatusCode < 500 {
			state = "rejected"
		}
		if err := s.repo.UpdateRunSubmission(ctx, c.TenantID, runID, state, submitErr.Error()); err != nil {
			return nil, fmt.Errorf("persist submission outcome: %w (TaskFlow: %v)", err, submitErr)
		}
		link.SubmissionState = state
		link.SubmissionError = submitErr.Error()
		if state == "rejected" {
			return nil, apperrors.NewServiceUnavailableError("TaskFlow rejected verification: " + submitErr.Error()).WithDetails(map[string]any{"run_id": runID, "submission_state": state})
		}
		return &ChangeRunView{Link: link, Applicable: true, ExecutionError: submitErr.Error()}, nil
	}
	return s.recordAccepted(ctx, change, snapshot, link, actual)
}
func (s *ChangeService) recordAccepted(ctx context.Context, change *types.ChangeRequest, snapshot *types.ChangeSnapshot, link *types.ChangeRunLink, run *cloudclient.VerificationRun) (*ChangeRunView, error) {
	if !runMatches(run, snapshot, link.RunID) {
		return nil, apperrors.NewConflictError("TaskFlow run fields do not match the confirmed snapshot")
	}
	if err := s.repo.UpdateRunSubmission(ctx, link.TenantID, link.RunID, "accepted", ""); err != nil {
		return nil, err
	}
	link.SubmissionState = "accepted"
	link.SubmissionError = ""
	current, err := s.Get(ctx, change.ID)
	if err != nil {
		return nil, err
	}
	return &ChangeRunView{Link: link, Run: run, Applicable: current.InputRevision == snapshot.InputRevision}, nil
}

// Run reads execution facts; an unavailable upstream remains an explicit per-run error.
func (s *ChangeService) Run(ctx context.Context, id, runID string) (*ChangeRunView, error) {
	change, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	link, err := s.repo.RunLink(ctx, change.TenantID, runID)
	if err != nil {
		return nil, err
	}
	if link.ChangeID != id {
		return nil, apperrors.NewNotFoundError("run link not found")
	}
	snapshot, err := s.repo.Snapshot(ctx, change.TenantID, id, link.SnapshotID)
	if err != nil {
		return nil, err
	}
	if _, err := s.sourcesForInput(ctx, types.CallerFromContext(ctx), snapshot.Input); err != nil {
		return nil, err
	}
	view := &ChangeRunView{Link: link, Applicable: change.InputRevision == snapshot.InputRevision}
	client, err := s.verifier()
	if err != nil {
		view.ExecutionError = err.Error()
		return view, nil
	}
	actual, err := client.GetVerificationRun(ctx, runID)
	if err != nil {
		view.ExecutionError = err.Error()
		return view, nil
	}
	if !runMatches(actual, snapshot, runID) {
		return nil, apperrors.NewConflictError("TaskFlow run fields do not match the confirmed snapshot")
	}
	view.Run = actual
	return view, nil
}

// Runs keeps all attempts visible, including failures and historical inputs.
func (s *ChangeService) Runs(ctx context.Context, id string) ([]*ChangeRunView, error) {
	change, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	links, err := s.repo.RunLinks(ctx, change.TenantID, id)
	if err != nil {
		return nil, err
	}
	out := []*ChangeRunView{}
	for _, link := range links {
		view, err := s.Run(ctx, id, link.RunID)
		if err != nil {
			// Keep local association metadata visible without forwarding a report
			// whose historical sources the caller cannot currently read.
			var app *apperrors.AppError
			if stderrors.As(err, &app) && (app.HTTPCode == http.StatusForbidden || app.HTTPCode == http.StatusNotFound) {
				snapshot, snapshotErr := s.repo.Snapshot(ctx, change.TenantID, id, link.SnapshotID)
				if snapshotErr != nil {
					return nil, snapshotErr
				}
				out = append(out, &ChangeRunView{Link: link, Applicable: change.InputRevision == snapshot.InputRevision, ExecutionError: err.Error()})
				continue
			}
			return nil, err
		}
		out = append(out, view)
	}
	return out, nil
}

// Artifact checks the change/run association before asking the authenticated SDK.
func (s *ChangeService) Artifact(ctx context.Context, id, runID string, generation uint64, artifact string) ([]byte, error) {
	view, err := s.Run(ctx, id, runID)
	if err != nil {
		return nil, err
	}
	if view.Run == nil {
		return nil, apperrors.NewServiceUnavailableError(view.ExecutionError)
	}
	found := false
	for _, attempt := range view.Run.Attempts {
		if attempt.Generation == generation {
			for _, check := range attempt.Checks {
				if check.ArtifactID == artifact {
					found = true
				}
			}
		}
	}
	if !found {
		return nil, apperrors.NewNotFoundError("artifact is not registered for this run generation")
	}
	client, err := s.verifier()
	if err != nil {
		return nil, err
	}
	return client.GetVerificationArtifact(ctx, runID, generation, artifact)
}

// Handoff can close only the current confirmed input with a complete passing report.
func (s *ChangeService) Handoff(ctx context.Context, id, runID, conclusion, notes string, expected int64) (*types.ChangeHandoff, error) {
	c, err := s.caller(ctx, true)
	if err != nil {
		return nil, err
	}
	change, err := s.repo.Get(ctx, c.TenantID, id)
	if err != nil {
		return nil, err
	}
	if change.Stage != "active" || change.Revision != expected {
		return nil, apperrors.NewConflictError("handoff requires the current active revision")
	}
	if conclusion != "accepted" && conclusion != "rejected" {
		return nil, apperrors.NewBadRequestError("conclusion must be accepted or rejected")
	}
	if strings.TrimSpace(notes) == "" {
		return nil, apperrors.NewBadRequestError("handoff notes are required")
	}
	view, err := s.Run(ctx, id, runID)
	if err != nil {
		return nil, err
	}
	if !view.Applicable {
		return nil, apperrors.NewConflictError("historical verification is not applicable to current inputs")
	}
	if conclusion == "accepted" {
		if view.Run == nil || view.Run.Status != "success" {
			return nil, apperrors.NewConflictError("acceptance requires a successful TaskFlow run")
		}
		passed := false
		for _, attempt := range view.Run.Attempts {
			if attempt.Generation == view.Run.Generation && attempt.Verdict == "passed" {
				passed = true
				present := map[string]bool{}
				for _, check := range attempt.Checks {
					if check.Required && check.Status == "passed" {
						present[check.ID] = true
					}
					if check.Required && check.Status != "passed" {
						passed = false
					}
				}
				for _, required := range view.Run.CheckIDs {
					if !present[required] {
						passed = false
					}
				}
			}
		}
		if !passed {
			return nil, apperrors.NewConflictError("acceptance requires every mandatory check in the current generation to pass")
		}
		snapshot, err := s.repo.Snapshot(ctx, c.TenantID, id, view.Link.SnapshotID)
		if err != nil {
			return nil, err
		}
		p, err := s.project(c, snapshot.Input)
		if err != nil {
			return nil, err
		}
		if err = s.validateInput(ctx, c, p, snapshot.Input, true); err != nil {
			return nil, err
		}
	}
	h := &types.ChangeHandoff{ID: uuid.NewString(), TenantID: c.TenantID, ChangeID: id, SnapshotID: view.Link.SnapshotID, RunID: runID, Conclusion: conclusion, Notes: notes, ActorID: c.UserID, CreatedAt: time.Now()}
	if err := s.repo.Handoff(ctx, h, expected); err != nil {
		return nil, err
	}
	return h, nil
}
func (s *ChangeService) Handoffs(ctx context.Context, id string) ([]*types.ChangeHandoff, error) {
	change, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.repo.Handoffs(ctx, change.TenantID, id)
}

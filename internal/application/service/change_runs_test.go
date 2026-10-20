package service

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	cloudclient "github.com/Tencent/WeKnora/internal/integration/taskflowclient"
)

type testChangeVerifier struct {
	runs         map[string]*cloudclient.VerificationRun
	submits      int
	loseResponse bool
	gets         int
	artifacts    int
}

func (v *testChangeVerifier) SubmitVerification(_ context.Context, in cloudclient.VerificationRequest) (*cloudclient.VerificationRun, error) {
	v.submits++
	run := &cloudclient.VerificationRun{VerificationRequest: in, Status: "pending", Generation: 1, Attempts: []cloudclient.VerificationReport{}}
	v.runs[in.RunID] = run
	if v.loseResponse {
		v.loseResponse = false
		return nil, fmt.Errorf("response connection lost after acceptance")
	}
	return run, nil
}
func (v *testChangeVerifier) GetVerificationRun(_ context.Context, id string) (*cloudclient.VerificationRun, error) {
	v.gets++
	if r := v.runs[id]; r != nil {
		return r, nil
	}
	return nil, &cloudclient.APIError{StatusCode: http.StatusNotFound, Message: "run not found"}
}
func (v *testChangeVerifier) GetVerificationArtifact(context.Context, string, uint64, string) ([]byte, error) {
	v.artifacts++
	return []byte("actual log"), nil
}

func TestChangeRunUnknownOutcomeRecoveryAndHandoff(t *testing.T) {
	s, ctx, input := changeTestService()
	v := &testChangeVerifier{runs: map[string]*cloudclient.VerificationRun{}, loseResponse: true}
	s.verification = v
	c, err := s.Create(ctx, CreateChangeRequest{Title: "Change", Input: input})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := s.Confirm(ctx, c.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	runID := "d05fb8d9-655c-4c20-8b2d-b2594c5a4dc3"
	view, err := s.SubmitRun(ctx, c.ID, snapshot.ID, runID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if view.Link.SubmissionState != "unknown" || view.Run != nil {
		t.Fatalf("unknown outcome hidden: %+v", view)
	}
	view, err = s.SubmitRun(ctx, c.ID, snapshot.ID, runID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if v.submits != 1 || view.Run == nil || view.Link.SubmissionState != "accepted" {
		t.Fatalf("recovery duplicated execution: %+v submits %d", view, v.submits)
	}
	_, err = s.Handoff(ctx, c.ID, runID, "accepted", "reviewed", 2)
	requireChangeStatus(t, err, http.StatusConflict)
	v.runs[runID].Status = "failed"
	v.runs[runID].Attempts = []cloudclient.VerificationReport{{Generation: 1, Verdict: "failed", Checks: []cloudclient.VerificationCheck{{ID: "test", Required: true, Status: "failed"}}}}
	_, err = s.Handoff(ctx, c.ID, runID, "accepted", "reviewed", 2)
	requireChangeStatus(t, err, http.StatusConflict)
	v.runs[runID].Status = "success"
	v.runs[runID].Attempts[0].Verdict = "passed"
	v.runs[runID].Attempts[0].Checks[0].Status = "passed"
	if _, err = s.Handoff(ctx, c.ID, runID, "accepted", "self verification recorded explicitly", 2); err != nil {
		t.Fatal(err)
	}
	closed, err := s.Get(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if closed.Stage != "closed" {
		t.Fatal("acceptance did not close")
	}
}
func TestChangeHistoricalRunCannotAccept(t *testing.T) {
	s, ctx, input := changeTestService()
	v := &testChangeVerifier{runs: map[string]*cloudclient.VerificationRun{}}
	s.verification = v
	c, _ := s.Create(ctx, CreateChangeRequest{Title: "Change", Input: input})
	snapshot, _ := s.Confirm(ctx, c.ID, 1)
	runID := "e05fb8d9-655c-4c20-8b2d-b2594c5a4dc3"
	if _, err := s.SubmitRun(ctx, c.ID, snapshot.ID, runID, 2); err != nil {
		t.Fatal(err)
	}
	input.TargetCommit = "2222222222222222222222222222222222222222"
	if _, err := s.Patch(ctx, c.ID, PatchChangeRequest{ExpectedRevision: 2, Input: &input}); err != nil {
		t.Fatal(err)
	}
	view, err := s.Run(ctx, c.ID, runID)
	if err != nil {
		t.Fatal(err)
	}
	if view.Applicable {
		t.Fatal("old run is current")
	}
	_, err = s.Handoff(ctx, c.ID, runID, "accepted", "reviewed", 3)
	requireChangeStatus(t, err, http.StatusConflict)
}

func TestChangeUnknownHistoricalRunDoesNotBlockANewSnapshot(t *testing.T) {
	s, ctx, input := changeTestService()
	v := &testChangeVerifier{runs: map[string]*cloudclient.VerificationRun{}, loseResponse: true}
	s.verification = v
	c, err := s.Create(ctx, CreateChangeRequest{Title: "Change", Input: input})
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.Confirm(ctx, c.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	aID := "d05fb8d9-655c-4c20-8b2d-b2594c5a4dc3"
	view, err := s.SubmitRun(ctx, c.ID, a.ID, aID, 2)
	if err != nil || view.Link.SubmissionState != "unknown" {
		t.Fatalf("want unknown A: %+v %v", view, err)
	}
	actualA := v.runs[aID]
	input.TargetCommit = "2222222222222222222222222222222222222222"
	c, err = s.Patch(ctx, c.ID, PatchChangeRequest{ExpectedRevision: 2, Input: &input})
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Confirm(ctx, c.ID, c.Revision)
	if err != nil {
		t.Fatal(err)
	}
	// A is no longer accepted upstream. Recovery may query it, not execute stale input.
	delete(v.runs, aID)
	_, err = s.SubmitRun(ctx, c.ID, a.ID, aID, 4)
	requireChangeStatus(t, err, http.StatusConflict)
	if v.submits != 1 {
		t.Fatal("historical recovery restarted A")
	}
	bID := "e05fb8d9-655c-4c20-8b2d-b2594c5a4dc3"
	view, err = s.SubmitRun(ctx, c.ID, b.ID, bID, 4)
	if err != nil || !view.Applicable || view.Link.SnapshotID != b.ID || v.submits != 2 {
		t.Fatalf("B was blocked by A: %+v %v", view, err)
	}
	v.runs[aID] = actualA
	view, err = s.SubmitRun(ctx, c.ID, a.ID, aID, 4)
	if err != nil || view.Applicable || view.Link.SnapshotID != a.ID || v.submits != 2 {
		t.Fatalf("accepted historical A was not reconciled: %+v %v", view, err)
	}
}

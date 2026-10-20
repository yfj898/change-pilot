package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChangeReadSDKPreservesFacts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer member" {
			t.Errorf("request changed: %s %s", r.Method, r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/changes/change/context":
			w.Write([]byte(`{"success":true,"data":{"change":{"id":"change","revision":5,"input_revision":2},"sources":[],"snapshots":[],"current_snapshot_id":"snapshot"}}`))
		case "/api/v1/changes/change/runs/run":
			w.Write([]byte(`{"success":true,"data":{"link":{"run_id":"run","submission_state":"accepted"},"applicable":false,"run":{"run_id":"run","status":"failed","generation":2,"target_commit":"exact","attempts":[{"generation":2,"verdict":"failed","checks":[{"id":"test","required":true,"status":"failed","exit_code":1}]}]}}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL, WithBearerToken("member"))
	view, err := c.GetChangeContext(context.Background(), "change")
	if err != nil {
		t.Fatal(err)
	}
	if view.Change.Revision != 5 || view.Change.InputRevision != 2 || view.CurrentSnapshotID != "snapshot" {
		t.Fatalf("context changed: %+v", view)
	}
	run, err := c.GetChangeVerificationRun(context.Background(), "change", "run")
	if err != nil {
		t.Fatal(err)
	}
	if run.Applicable || run.Run.Status != "failed" || run.Run.Attempts[0].Checks[0].Status != "failed" {
		t.Fatalf("failure changed: %+v", run)
	}
}

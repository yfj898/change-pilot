package repository

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// This test executes the shipped SQLite migration, not AutoMigrate.
func TestChangeRepositoryCASAndSnapshots(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "changes.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	_, sourceFile, _, _ := runtime.Caller(0)
	for _, name := range []string{"000042_change_workspace.up.sql", "000043_change_runs.up.sql", "000044_change_draft.up.sql"} {
		bytes, err := os.ReadFile(filepath.Join(filepath.Dir(sourceFile), "../../../migrations/sqlite", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Exec(string(bytes)).Error; err != nil {
			t.Fatal(err)
		}
	}
	repo := NewChangeRepository(db)
	ctx := context.Background()
	c := &types.ChangeRequest{ID: "change", TenantID: 1, CreatorID: "creator", Title: "Original", Revision: 1, InputRevision: 1, Stage: "draft", CreatedAt: time.Now(), UpdatedAt: time.Now(), Input: types.ChangeInput{Objective: "source", References: []types.ChangeReference{}, CheckPlan: types.ChangeCheckPlan{CheckIDs: []string{"test"}}}}
	if err := repo.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(ctx, 2, c.ID); err == nil {
		t.Fatal("cross-workspace read succeeded")
	}
	c.Title = "Updated"
	c.Revision = 2
	if err := repo.UpdateCAS(ctx, c, 1); err != nil {
		t.Fatal(err)
	}
	c.Title = "Stale"
	c.Revision = 2
	if err := repo.UpdateCAS(ctx, c, 1); err == nil {
		t.Fatal("stale CAS overwrote record")
	}
	s := &types.ChangeSnapshot{ID: "snapshot", TenantID: 1, ChangeID: c.ID, Revision: 2, InputRevision: 1, Input: c.Input, ConfirmedBy: "creator", CreatedAt: time.Now()}
	if err := repo.Confirm(ctx, s, 2); err != nil {
		t.Fatal(err)
	}
	if err := repo.Confirm(ctx, &types.ChangeSnapshot{ID: "duplicate", TenantID: 1, ChangeID: c.ID, Revision: 3, InputRevision: 1, Input: c.Input, ConfirmedBy: "creator", CreatedAt: time.Now()}, 3); err == nil {
		t.Fatal("duplicate input confirmation succeeded")
	}
	current, err := repo.Get(ctx, 1, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Title != "Updated" || current.Revision != 3 || current.Stage != "active" {
		t.Fatalf("CAS/rollback mismatch: %+v", current)
	}
	history, err := repo.Snapshots(ctx, 1, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].Input.Objective != "source" {
		t.Fatalf("snapshot did not round trip: %+v", history)
	}
	other, err := repo.Snapshots(ctx, 2, c.ID)
	if err != nil || len(other) != 0 {
		t.Fatalf("cross-workspace snapshots: %+v %v", other, err)
	}
	link := &types.ChangeRunLink{RunID: "run", TenantID: 1, ChangeID: c.ID, SnapshotID: s.ID, SubmittedBy: "creator", SubmissionState: "pending", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.CreateRunIntent(ctx, link, 2); err == nil {
		t.Fatal("stale revision created intent")
	}
	if err := repo.CreateRunIntent(ctx, link, 3); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRunIntent(ctx, link, 3); err != nil {
		t.Fatal(err)
	}
	links, err := repo.RunLinks(ctx, 1, c.ID)
	if err != nil || len(links) != 1 {
		t.Fatalf("duplicate intent: %v %v", links, err)
	}
	if err := repo.UpdateRunSubmission(ctx, 1, "run", "unknown", "response lost"); err != nil {
		t.Fatal(err)
	}
	saved, err := repo.RunLink(ctx, 1, "run")
	if err != nil || saved.SubmissionState != "unknown" {
		t.Fatalf("lost submission state: %v %v", saved, err)
	}
	if _, err := repo.RunLink(ctx, 2, "run"); err == nil {
		t.Fatal("cross-workspace run read")
	}
	h := &types.ChangeHandoff{ID: "handoff", TenantID: 1, ChangeID: c.ID, SnapshotID: s.ID, RunID: "run", Conclusion: "accepted", Notes: "reviewed", ActorID: "reviewer", CreatedAt: time.Now()}
	if err := repo.Handoff(ctx, h, 2); err == nil {
		t.Fatal("stale handoff closed a change")
	}
	if err := repo.Handoff(ctx, h, 3); err != nil {
		t.Fatal(err)
	}
	current, err = repo.Get(ctx, 1, c.ID)
	if err != nil || current.Stage != "closed" || current.Revision != 4 {
		t.Fatalf("handoff did not atomically close: %+v %v", current, err)
	}
}

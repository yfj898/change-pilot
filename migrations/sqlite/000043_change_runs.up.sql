CREATE TABLE change_run_links (
 run_id TEXT PRIMARY KEY,
 tenant_id INTEGER NOT NULL,
 change_id TEXT NOT NULL REFERENCES change_requests(id),
 snapshot_id TEXT NOT NULL REFERENCES change_snapshots(id),
 submitted_by TEXT NOT NULL,
 submission_state TEXT NOT NULL CHECK(submission_state IN ('pending','accepted','unknown','rejected')),
 submission_error TEXT NOT NULL DEFAULT '',
 created_at DATETIME NOT NULL,
 updated_at DATETIME NOT NULL
);
CREATE INDEX idx_change_run_links_change ON change_run_links(tenant_id,change_id,created_at DESC);
CREATE TABLE change_handoffs (
 id TEXT PRIMARY KEY,
 tenant_id INTEGER NOT NULL,
 change_id TEXT NOT NULL REFERENCES change_requests(id),
 snapshot_id TEXT NOT NULL REFERENCES change_snapshots(id),
 run_id TEXT NOT NULL REFERENCES change_run_links(run_id),
 conclusion TEXT NOT NULL CHECK(conclusion IN ('accepted','rejected')),
 notes TEXT NOT NULL,
 actor_id TEXT NOT NULL,
 created_at DATETIME NOT NULL
);
CREATE INDEX idx_change_handoffs_change ON change_handoffs(tenant_id,change_id,created_at DESC);

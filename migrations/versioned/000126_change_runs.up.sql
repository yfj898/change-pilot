CREATE TABLE change_run_links (
 run_id VARCHAR(36) PRIMARY KEY,
 tenant_id BIGINT NOT NULL,
 change_id VARCHAR(36) NOT NULL REFERENCES change_requests(id),
 snapshot_id VARCHAR(36) NOT NULL REFERENCES change_snapshots(id),
 submitted_by VARCHAR(64) NOT NULL,
 submission_state VARCHAR(20) NOT NULL CHECK(submission_state IN ('pending','accepted','unknown','rejected')),
 submission_error TEXT NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL,
 updated_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_change_run_links_change ON change_run_links(tenant_id,change_id,created_at DESC);
CREATE TABLE change_handoffs (
 id VARCHAR(36) PRIMARY KEY,
 tenant_id BIGINT NOT NULL,
 change_id VARCHAR(36) NOT NULL REFERENCES change_requests(id),
 snapshot_id VARCHAR(36) NOT NULL REFERENCES change_snapshots(id),
 run_id VARCHAR(36) NOT NULL REFERENCES change_run_links(run_id),
 conclusion VARCHAR(20) NOT NULL CHECK(conclusion IN ('accepted','rejected')),
 notes TEXT NOT NULL,
 actor_id VARCHAR(64) NOT NULL,
 created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_change_handoffs_change ON change_handoffs(tenant_id,change_id,created_at DESC);

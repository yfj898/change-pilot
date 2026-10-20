CREATE TABLE change_requests (
 id TEXT PRIMARY KEY,
 tenant_id INTEGER NOT NULL,
 creator_id TEXT NOT NULL,
 title TEXT NOT NULL,
 assignee_id TEXT NOT NULL DEFAULT '',
 notes TEXT NOT NULL DEFAULT '',
 input TEXT NOT NULL,
 revision INTEGER NOT NULL,
 input_revision INTEGER NOT NULL,
 stage TEXT NOT NULL CHECK (stage IN ('draft','active','closed')),
 created_at DATETIME NOT NULL,
 updated_at DATETIME NOT NULL
);
CREATE INDEX idx_change_requests_tenant_created ON change_requests(tenant_id,created_at DESC,id DESC);
CREATE TABLE change_snapshots (
 id TEXT PRIMARY KEY,
 tenant_id INTEGER NOT NULL,
 change_id TEXT NOT NULL REFERENCES change_requests(id),
 revision INTEGER NOT NULL,
 input_revision INTEGER NOT NULL,
 input TEXT NOT NULL,
 confirmed_by TEXT NOT NULL,
 created_at DATETIME NOT NULL,
 UNIQUE(tenant_id,change_id,input_revision)
);
CREATE INDEX idx_change_snapshots_change ON change_snapshots(tenant_id,change_id,created_at DESC);

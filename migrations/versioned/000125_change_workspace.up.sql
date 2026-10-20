CREATE TABLE change_requests (
 id VARCHAR(36) PRIMARY KEY,
 tenant_id BIGINT NOT NULL,
 creator_id VARCHAR(64) NOT NULL,
 title TEXT NOT NULL,
 assignee_id VARCHAR(64) NOT NULL DEFAULT '',
 notes TEXT NOT NULL DEFAULT '',
 input TEXT NOT NULL,
 revision BIGINT NOT NULL,
 input_revision BIGINT NOT NULL,
 stage VARCHAR(20) NOT NULL CHECK (stage IN ('draft','active','closed')),
 created_at TIMESTAMPTZ NOT NULL,
 updated_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_change_requests_tenant_created ON change_requests(tenant_id,created_at DESC,id DESC);
CREATE TABLE change_snapshots (
 id VARCHAR(36) PRIMARY KEY,
 tenant_id BIGINT NOT NULL,
 change_id VARCHAR(36) NOT NULL REFERENCES change_requests(id),
 revision BIGINT NOT NULL,
 input_revision BIGINT NOT NULL,
 input TEXT NOT NULL,
 confirmed_by VARCHAR(64) NOT NULL,
 created_at TIMESTAMPTZ NOT NULL,
 UNIQUE(tenant_id,change_id,input_revision)
);
CREATE INDEX idx_change_snapshots_change ON change_snapshots(tenant_id,change_id,created_at DESC);

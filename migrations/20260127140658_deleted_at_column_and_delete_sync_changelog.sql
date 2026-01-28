-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

ALTER TABLE texts
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

ALTER TABLE credentials
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

ALTER TABLE cards
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

ALTER TABLE binaries
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Drop sync_changelog related indexes and table
DROP INDEX IF EXISTS idx_sync_user_created;
DROP INDEX IF EXISTS idx_sync_user_version;
DROP TABLE IF EXISTS sync_changelog;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Recreate sync_changelog table and its indexes
CREATE TABLE IF NOT EXISTS sync_changelog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    operation VARCHAR(10) NOT NULL,
    data JSONB,
    version BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sync_user_created ON sync_changelog(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_sync_user_version ON sync_changelog(user_id, version);

-- Drop deleted_at columns
ALTER TABLE users
    DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE texts
    DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE credentials
    DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE cards
    DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE binaries
    DROP COLUMN IF EXISTS deleted_at;
-- +goose StatementEnd

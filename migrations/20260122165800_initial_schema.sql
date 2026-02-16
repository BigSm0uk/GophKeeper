-- +goose Up
-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255),
    hashed_password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create texts table
CREATE TABLE IF NOT EXISTS texts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    metadata TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create credentials table
CREATE TABLE IF NOT EXISTS credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    login VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL,
    url VARCHAR(2048),
    metadata TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create cards table
CREATE TABLE IF NOT EXISTS cards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    card_number VARCHAR(255) NOT NULL,
    cardholder_name VARCHAR(255) NOT NULL,
    expiry_date VARCHAR(10) NOT NULL,
    cvv VARCHAR(4) NOT NULL,
    bank_name VARCHAR(255),
    metadata TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create binaries table
CREATE TABLE IF NOT EXISTS binaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL CHECK (size > 0),
    content_type VARCHAR(255) NOT NULL,
    metadata TEXT,
    storage_path VARCHAR(2048) NOT NULL,
    checksum VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create sync_changelog table
CREATE TABLE IF NOT EXISTS sync_changelog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,  -- 'credentials', 'texts', 'binaries', 'cards'
    entity_id UUID NOT NULL,
    operation VARCHAR(10) NOT NULL,    -- 'create', 'update', 'delete'
    data JSONB,                        
    version BIGINT NOT NULL,           
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_texts_user_id ON texts(user_id);
CREATE INDEX IF NOT EXISTS idx_credentials_user_id ON credentials(user_id);
CREATE INDEX IF NOT EXISTS idx_cards_user_id ON cards(user_id);
CREATE INDEX IF NOT EXISTS idx_binaries_user_id ON binaries(user_id);
CREATE INDEX IF NOT EXISTS idx_binaries_checksum ON binaries(checksum);
CREATE INDEX IF NOT EXISTS idx_binaries_user_checksum ON binaries(user_id, checksum);
-- Create index on username for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

-- Create indexes for sync_changelog table
CREATE INDEX IF NOT EXISTS idx_sync_user_created ON sync_changelog(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_sync_user_version ON sync_changelog(user_id, version);

-- Create partial index on email for users who have email set
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE email IS NOT NULL;

-- +goose Down
-- Drop indexes first
DROP INDEX IF EXISTS idx_sync_user_created;
DROP INDEX IF EXISTS idx_sync_user_version;
DROP INDEX IF EXISTS idx_texts_user_id;
DROP INDEX IF EXISTS idx_credentials_user_id;
DROP INDEX IF EXISTS idx_cards_user_id;
DROP INDEX IF EXISTS idx_binaries_user_id;
DROP INDEX IF EXISTS idx_binaries_checksum;
DROP INDEX IF EXISTS idx_binaries_user_checksum;
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_users_email;

-- Drop tables in reverse order due to foreign key constraints
DROP TABLE IF EXISTS sync_changelog;
DROP TABLE IF EXISTS binaries;
DROP TABLE IF EXISTS cards;
DROP TABLE IF EXISTS credentials;
DROP TABLE IF EXISTS texts;
DROP TABLE IF EXISTS users;

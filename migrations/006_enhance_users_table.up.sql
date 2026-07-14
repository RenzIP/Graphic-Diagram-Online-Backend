-- 006_enhance_users_table.up.sql

ALTER TABLE user_profiles
    ADD COLUMN IF NOT EXISTS username TEXT,
    ADD COLUMN IF NOT EXISTS email TEXT,
    ADD COLUMN IF NOT EXISTS password TEXT,
    ADD COLUMN IF NOT EXISTS provider TEXT DEFAULT 'local',
    ADD COLUMN IF NOT EXISTS provider_id TEXT,
    ADD COLUMN IF NOT EXISTS role TEXT DEFAULT 'user',
    ADD COLUMN IF NOT EXISTS status TEXT DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS remember_token TEXT,
    ADD COLUMN IF NOT EXISTS last_login TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT now();

-- Map existing full_name to name
ALTER TABLE user_profiles ADD COLUMN IF NOT EXISTS name TEXT;
UPDATE user_profiles SET name = full_name WHERE name IS NULL AND full_name IS NOT NULL;

-- Add avatar alias for avatar_url
ALTER TABLE user_profiles ADD COLUMN IF NOT EXISTS avatar TEXT;
UPDATE user_profiles SET avatar = avatar_url WHERE avatar IS NULL AND avatar_url IS NOT NULL;

-- Add constraints
ALTER TABLE user_profiles ADD CONSTRAINT user_profiles_email_unique UNIQUE (email);
ALTER TABLE user_profiles ADD CONSTRAINT user_profiles_username_unique UNIQUE (username);

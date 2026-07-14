-- 006_enhance_users_table.down.sql

ALTER TABLE user_profiles DROP CONSTRAINT IF EXISTS user_profiles_email_unique;
ALTER TABLE user_profiles DROP CONSTRAINT IF EXISTS user_profiles_username_unique;

ALTER TABLE user_profiles
    DROP COLUMN IF EXISTS username,
    DROP COLUMN IF EXISTS email,
    DROP COLUMN IF EXISTS password,
    DROP COLUMN IF EXISTS provider,
    DROP COLUMN IF EXISTS provider_id,
    DROP COLUMN IF EXISTS role,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS email_verified_at,
    DROP COLUMN IF EXISTS remember_token,
    DROP COLUMN IF EXISTS last_login,
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS name,
    DROP COLUMN IF EXISTS avatar;

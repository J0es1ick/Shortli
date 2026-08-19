ALTER TABLE user_info ADD COLUMN IF NOT EXISTS role VARCHAR(16);

UPDATE user_info
SET role = CASE WHEN is_admin THEN 'owner' ELSE 'user' END
WHERE role IS NULL;

ALTER TABLE user_info ALTER COLUMN role SET DEFAULT 'user';
ALTER TABLE user_info ALTER COLUMN role SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'user_info_role_check'
    ) THEN
        ALTER TABLE user_info
            ADD CONSTRAINT user_info_role_check
            CHECK (role IN ('owner', 'admin', 'support', 'user'));
    END IF;
END $$;

UPDATE user_info SET is_admin = (role <> 'user');

CREATE INDEX IF NOT EXISTS idx_user_info_role ON user_info(role);

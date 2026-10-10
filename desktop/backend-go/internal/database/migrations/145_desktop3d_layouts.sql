-- 145_desktop3d_layouts.sql
-- User-owned saved layouts for the experimental 3D Desktop surface.

CREATE TABLE IF NOT EXISTS desktop3d_layouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(40) NOT NULL DEFAULT 'custom' CHECK (type IN ('custom')),
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    modules JSONB NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(modules) = 'array'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE desktop3d_layouts
    ALTER COLUMN user_id TYPE VARCHAR(255) USING user_id::text,
    ALTER COLUMN name TYPE VARCHAR(255),
    ALTER COLUMN type SET DEFAULT 'custom',
    ALTER COLUMN modules SET DEFAULT '[]'::jsonb;

ALTER TABLE desktop3d_layouts
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE desktop3d_layouts
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC',
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'desktop3d_layouts_user_id_fkey'
          AND conrelid = 'desktop3d_layouts'::regclass
    ) THEN
        ALTER TABLE desktop3d_layouts
            ADD CONSTRAINT desktop3d_layouts_user_id_fkey
            FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_desktop3d_layouts_user
    ON desktop3d_layouts(user_id, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_desktop3d_layouts_one_active
    ON desktop3d_layouts(user_id)
    WHERE is_active = TRUE;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'desktop3d_layouts_type_check'
          AND conrelid = 'desktop3d_layouts'::regclass
    ) THEN
        ALTER TABLE desktop3d_layouts
            ADD CONSTRAINT desktop3d_layouts_type_check
            CHECK (type IN ('custom'));
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'desktop3d_layouts_modules_check'
          AND conrelid = 'desktop3d_layouts'::regclass
    ) THEN
        ALTER TABLE desktop3d_layouts
            ADD CONSTRAINT desktop3d_layouts_modules_check
            CHECK (jsonb_typeof(modules) = 'array');
    END IF;
END $$;

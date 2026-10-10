-- Desktop 3D Layout Storage
-- Allows users to save and load custom 3D Desktop layouts

CREATE TABLE IF NOT EXISTS desktop3d_layouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL DEFAULT 'custom', -- 'default' or 'custom'
    is_active BOOLEAN DEFAULT false,
    modules JSONB NOT NULL DEFAULT '[]'::jsonb, -- Array of ModulePosition objects
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT chk_layout_type CHECK (type IN ('default', 'custom'))
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_desktop3d_layouts_user_id
    ON desktop3d_layouts(user_id);

CREATE INDEX IF NOT EXISTS idx_desktop3d_layouts_user_active
    ON desktop3d_layouts(user_id, is_active)
    WHERE is_active = true;

CREATE INDEX IF NOT EXISTS idx_desktop3d_layouts_created
    ON desktop3d_layouts(created_at DESC);

-- Comments for documentation
COMMENT ON TABLE desktop3d_layouts IS
    'Stores custom 3D Desktop layouts with module positions, rotations, and scales';

COMMENT ON COLUMN desktop3d_layouts.modules IS
    'JSON array of {module_id, position: {x, y, z}, rotation: {x, y, z}, scale}';

COMMENT ON COLUMN desktop3d_layouts.type IS
    'Layout type: "default" for the system 5-ring geodesic layout, "custom" for user-created layouts';

COMMENT ON COLUMN desktop3d_layouts.is_active IS
    'Indicates if this is the currently active layout for the user';

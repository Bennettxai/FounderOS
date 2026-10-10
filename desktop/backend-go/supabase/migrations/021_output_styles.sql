-- ================================================
-- Migration 018: Output Styles System
-- Description: Output style templates and user preferences
-- Author: Pedro
-- Date: 2025-12-31
-- ================================================

-- ================================================
-- OUTPUT STYLES TABLE
-- Style templates for AI responses
-- ================================================
CREATE TABLE IF NOT EXISTS output_styles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Style Identity
    name VARCHAR(100) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    icon VARCHAR(50),

    -- Style Configuration
    style_type VARCHAR(50) NOT NULL,          -- 'prose', 'structured', 'code', 'mixed'

    -- Formatting Rules
    use_headers BOOLEAN DEFAULT TRUE,
    use_bullets BOOLEAN DEFAULT TRUE,
    use_numbered_lists BOOLEAN DEFAULT FALSE,
    use_paragraphs BOOLEAN DEFAULT TRUE,
    use_code_blocks BOOLEAN DEFAULT FALSE,
    use_tables BOOLEAN DEFAULT FALSE,
    use_blockquotes BOOLEAN DEFAULT FALSE,

    -- Length & Density
    verbosity VARCHAR(20) DEFAULT 'balanced',  -- 'concise', 'balanced', 'detailed', 'comprehensive'
    max_paragraphs INTEGER,
    max_bullets_per_section INTEGER,

    -- Tone
    tone VARCHAR(50) DEFAULT 'professional',   -- 'casual', 'professional', 'formal', 'friendly', 'technical'

    -- System Prompt Addition
    style_instructions TEXT NOT NULL,

    -- Block Mapping (how to convert output to blocks)
    block_mapping JSONB DEFAULT '{}',

    is_system BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    sort_order INTEGER DEFAULT 0,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for output_styles
CREATE INDEX IF NOT EXISTS idx_output_styles_name ON output_styles(name);
CREATE INDEX IF NOT EXISTS idx_output_styles_active ON output_styles(is_active, sort_order);

-- ================================================
-- USER OUTPUT PREFERENCES TABLE
-- User-specific style preferences
-- ================================================
CREATE TABLE IF NOT EXISTS user_output_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Default style
    default_style_id UUID REFERENCES output_styles(id),

    -- Context-specific overrides
    style_overrides JSONB DEFAULT '{}',       -- {"focus_mode:deep": "style_id", "agent:analyst": "style_id"}

    -- Custom instructions
    custom_instructions TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id)
);

-- Indexes for user_output_preferences
CREATE INDEX IF NOT EXISTS idx_user_output_prefs ON user_output_preferences(user_id);

-- ================================================
-- SEED DATA: 8 Pre-defined Output Styles
-- ================================================
INSERT INTO output_styles (name, display_name, description, icon, style_type, use_headers, use_bullets, use_numbered_lists, use_paragraphs, use_code_blocks, use_tables, use_blockquotes, verbosity, tone, style_instructions, block_mapping, is_system, sort_order)
VALUES
    -- 1. Conversational (ChatGPT-like casual)
    ('conversational', 'Conversational', 'Natural, flowing conversation like talking to a friend', 'message-circle', 'prose',
     FALSE, FALSE, FALSE, TRUE, FALSE, FALSE, FALSE,
     'balanced', 'friendly',
     'Respond in a natural, conversational way. Use flowing paragraphs instead of bullet points or lists. Write as if having a friendly conversation. Avoid formal structure - just communicate naturally and clearly. Use "I" and "you" freely. Keep responses warm and approachable.',
     '{"paragraph": "text", "emphasis": "text"}',
     TRUE, 1),

    -- 2. Professional (Business communication)
    ('professional', 'Professional', 'Clear, structured business communication', 'briefcase', 'structured',
     TRUE, TRUE, FALSE, TRUE, FALSE, FALSE, FALSE,
     'balanced', 'professional',
     'Respond in a professional business style. Use clear structure with headers for main sections. Use bullet points for lists of items or key points. Keep language formal but accessible. Be direct and actionable.',
     '{"h2": "heading", "bullet": "bullet_list", "paragraph": "text"}',
     TRUE, 2),

    -- 3. Technical (Developer documentation)
    ('technical', 'Technical', 'Precise technical documentation with code examples', 'code', 'code',
     TRUE, TRUE, FALSE, TRUE, TRUE, TRUE, FALSE,
     'detailed', 'technical',
     'Respond in a technical documentation style. Include code examples where relevant. Use precise technical terminology. Structure with clear headers. Use tables for comparisons or specifications. Include type definitions and API signatures when applicable.',
     '{"h2": "heading", "code": "code", "table": "table", "bullet": "bullet_list"}',
     TRUE, 3),

    -- 4. Executive Summary (Brief, high-level)
    ('executive', 'Executive Summary', 'Brief, high-level summaries for quick consumption', 'zap', 'structured',
     FALSE, TRUE, FALSE, TRUE, FALSE, FALSE, FALSE,
     'concise', 'formal',
     'Provide brief, executive-level summaries. Lead with the key takeaway or recommendation. Use 3-5 bullet points maximum for supporting details. Avoid technical jargon. Focus on business impact and actionable insights. Keep total response under 200 words.',
     '{"bullet": "bullet_list", "paragraph": "text"}',
     TRUE, 4),

    -- 5. Detailed Analysis (Comprehensive reports)
    ('detailed', 'Detailed Analysis', 'Comprehensive, in-depth analysis with full context', 'file-text', 'structured',
     TRUE, TRUE, TRUE, TRUE, FALSE, TRUE, FALSE,
     'comprehensive', 'professional',
     'Provide comprehensive, detailed analysis. Use clear section headers. Include numbered lists for sequential steps or rankings. Use tables for data comparisons. Provide context and background. Include considerations, trade-offs, and recommendations. Be thorough but organized.',
     '{"h2": "heading", "h3": "subheading", "numbered": "numbered_list", "bullet": "bullet_list", "table": "table", "paragraph": "text"}',
     TRUE, 5),

    -- 6. Creative (Engaging, story-like)
    ('creative', 'Creative', 'Engaging, narrative style for creative content', 'sparkles', 'prose',
     FALSE, FALSE, FALSE, TRUE, FALSE, FALSE, TRUE,
     'detailed', 'casual',
     'Write in an engaging, creative style. Use narrative techniques - metaphors, vivid descriptions, storytelling. Create flow and rhythm in the writing. Engage the reader emotionally. Avoid dry, factual presentation. Make the content memorable and enjoyable to read.',
     '{"paragraph": "text", "blockquote": "quote"}',
     TRUE, 6),

    -- 7. Step-by-Step (Tutorial/how-to)
    ('tutorial', 'Step-by-Step', 'Clear tutorial style with numbered steps', 'list-ordered', 'structured',
     TRUE, FALSE, TRUE, TRUE, TRUE, FALSE, FALSE,
     'detailed', 'friendly',
     'Write in a clear tutorial style. Use numbered steps for any process or procedure. Include code examples with explanations. Add tips or notes for important points. Anticipate common questions or issues. Make sure each step is clear and actionable.',
     '{"h2": "heading", "numbered": "numbered_list", "code": "code", "note": "callout", "paragraph": "text"}',
     TRUE, 7),

    -- 8. Q&A (Direct answers)
    ('qa', 'Q&A', 'Direct question-and-answer format', 'help-circle', 'mixed',
     FALSE, FALSE, FALSE, TRUE, FALSE, FALSE, FALSE,
     'concise', 'friendly',
     'Answer directly and concisely. Start with the direct answer to the question. Then provide brief supporting context if needed. Avoid unnecessary preamble. If multiple questions, address each clearly. Use simple, clear language.',
     '{"paragraph": "text"}',
     TRUE, 8)

ON CONFLICT (name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    description = EXCLUDED.description,
    icon = EXCLUDED.icon,
    style_type = EXCLUDED.style_type,
    use_headers = EXCLUDED.use_headers,
    use_bullets = EXCLUDED.use_bullets,
    use_numbered_lists = EXCLUDED.use_numbered_lists,
    use_paragraphs = EXCLUDED.use_paragraphs,
    use_code_blocks = EXCLUDED.use_code_blocks,
    use_tables = EXCLUDED.use_tables,
    use_blockquotes = EXCLUDED.use_blockquotes,
    verbosity = EXCLUDED.verbosity,
    tone = EXCLUDED.tone,
    style_instructions = EXCLUDED.style_instructions,
    block_mapping = EXCLUDED.block_mapping,
    updated_at = NOW();

-- ================================================
-- TRIGGERS
-- ================================================
CREATE OR REPLACE FUNCTION update_output_styles_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_output_styles_updated_at ON output_styles;
CREATE TRIGGER trigger_output_styles_updated_at
    BEFORE UPDATE ON output_styles
    FOR EACH ROW
    EXECUTE FUNCTION update_output_styles_updated_at();

DROP TRIGGER IF EXISTS trigger_user_output_prefs_updated_at ON user_output_preferences;
CREATE TRIGGER trigger_user_output_prefs_updated_at
    BEFORE UPDATE ON user_output_preferences
    FOR EACH ROW
    EXECUTE FUNCTION update_output_styles_updated_at();

-- ================================================
-- COMMENTS
-- ================================================
COMMENT ON TABLE output_styles IS 'AI response style templates';
COMMENT ON TABLE user_output_preferences IS 'User-specific output style preferences';
COMMENT ON COLUMN output_styles.style_type IS 'Type: prose, structured, code, mixed';
COMMENT ON COLUMN output_styles.verbosity IS 'Level: concise, balanced, detailed, comprehensive';
COMMENT ON COLUMN output_styles.block_mapping IS 'Maps markdown elements to block types for document integration';

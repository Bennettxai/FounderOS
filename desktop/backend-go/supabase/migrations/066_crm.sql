-- Migration 041: CRM Enhancement
-- Companies, Pipelines, Deals, and CRM Activities
-- Extends the existing clients table with full CRM capabilities

-- ============================================================================
-- COMPANIES TABLE
-- ============================================================================

-- Note: This complements the existing "clients" table
-- clients = contacts/individuals, companies = organizations

CREATE TABLE IF NOT EXISTS companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Basic info
    name VARCHAR(255) NOT NULL,
    legal_name VARCHAR(255),
    industry VARCHAR(100),
    company_size VARCHAR(50),  -- '1-10', '11-50', '51-200', '201-500', '500+'

    -- Contact info
    website VARCHAR(500),
    email VARCHAR(255),
    phone VARCHAR(50),

    -- Address
    address_line1 VARCHAR(255),
    address_line2 VARCHAR(255),
    city VARCHAR(100),
    state VARCHAR(100),
    postal_code VARCHAR(20),
    country VARCHAR(100),

    -- Business details
    annual_revenue DECIMAL(15,2),
    currency VARCHAR(10) DEFAULT 'USD',
    fiscal_year_end VARCHAR(20),
    tax_id VARCHAR(100),

    -- Social/online
    linkedin_url VARCHAR(500),
    twitter_handle VARCHAR(100),

    -- Relationship
    owner_id VARCHAR(255),  -- Account owner/sales rep
    lifecycle_stage VARCHAR(50),  -- 'lead', 'prospect', 'customer', 'partner', 'churned'
    lead_source VARCHAR(100),

    -- Scoring
    health_score INT,  -- 0-100
    engagement_score INT,

    -- Logo
    logo_url TEXT,

    -- Custom fields
    custom_fields JSONB DEFAULT '{}',

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- External sync
    external_id VARCHAR(255),  -- HubSpot, Salesforce, etc.
    external_source VARCHAR(50),
    last_synced_at TIMESTAMPTZ,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_companies_user ON companies(user_id);
CREATE INDEX IF NOT EXISTS idx_companies_name ON companies(user_id, name);
CREATE INDEX IF NOT EXISTS idx_companies_industry ON companies(user_id, industry);
CREATE INDEX IF NOT EXISTS idx_companies_owner ON companies(owner_id);
CREATE INDEX IF NOT EXISTS idx_companies_lifecycle ON companies(user_id, lifecycle_stage);
CREATE INDEX IF NOT EXISTS idx_companies_external ON companies(external_source, external_id);

-- ============================================================================
-- CONTACT-COMPANY RELATIONSHIP
-- ============================================================================

-- Links clients (contacts) to companies
CREATE TABLE IF NOT EXISTS contact_company_relations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contact_id UUID NOT NULL,  -- References clients table
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

    -- Role at company
    job_title VARCHAR(255),
    department VARCHAR(100),
    role_type VARCHAR(50),  -- 'primary', 'billing', 'technical', 'decision_maker'

    -- Status
    is_primary BOOLEAN DEFAULT FALSE,  -- Primary contact for the company
    is_active BOOLEAN DEFAULT TRUE,

    -- Dates
    started_at DATE,
    ended_at DATE,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(contact_id, company_id)
);

CREATE INDEX IF NOT EXISTS idx_contact_company_contact ON contact_company_relations(contact_id);
CREATE INDEX IF NOT EXISTS idx_contact_company_company ON contact_company_relations(company_id);

-- ============================================================================
-- PIPELINES TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS pipelines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Pipeline info
    name VARCHAR(255) NOT NULL,
    description TEXT,
    pipeline_type VARCHAR(50) DEFAULT 'sales',  -- 'sales', 'hiring', 'projects', 'custom'

    -- Settings
    currency VARCHAR(10) DEFAULT 'USD',
    is_default BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,

    -- Display
    color VARCHAR(50),
    icon VARCHAR(50),

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pipelines_user ON pipelines(user_id);
CREATE INDEX IF NOT EXISTS idx_pipelines_default ON pipelines(user_id, is_default) WHERE is_default = TRUE;

-- ============================================================================
-- PIPELINE STAGES TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS pipeline_stages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_id UUID NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,

    -- Stage info
    name VARCHAR(100) NOT NULL,
    description TEXT,

    -- Position in pipeline
    position INT NOT NULL DEFAULT 0,

    -- Probability of closing (0-100)
    probability INT DEFAULT 0,

    -- Stage type
    stage_type VARCHAR(50) DEFAULT 'open',  -- 'open', 'won', 'lost'

    -- Rotting (stale deals)
    rotting_days INT,  -- Days before deal is considered stale

    -- Display
    color VARCHAR(50),

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(pipeline_id, name)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_stages_pipeline ON pipeline_stages(pipeline_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_stages_position ON pipeline_stages(pipeline_id, position);

-- ============================================================================
-- DEALS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS deals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Pipeline/stage
    pipeline_id UUID NOT NULL REFERENCES pipelines(id),
    stage_id UUID NOT NULL REFERENCES pipeline_stages(id),

    -- Deal info
    name VARCHAR(500) NOT NULL,
    description TEXT,

    -- Value
    amount DECIMAL(15,2),
    currency VARCHAR(10) DEFAULT 'USD',

    -- Probability (can override stage probability)
    probability INT,

    -- Expected close
    expected_close_date DATE,
    actual_close_date DATE,

    -- Assignment
    owner_id VARCHAR(255),

    -- Related entities
    company_id UUID REFERENCES companies(id),
    primary_contact_id UUID,  -- References clients table

    -- Status
    status VARCHAR(50) DEFAULT 'open',  -- 'open', 'won', 'lost'
    lost_reason VARCHAR(255),

    -- Priority
    priority VARCHAR(20) DEFAULT 'medium',  -- 'low', 'medium', 'high', 'urgent'

    -- Source
    lead_source VARCHAR(100),

    -- Scoring
    deal_score INT,

    -- Custom fields
    custom_fields JSONB DEFAULT '{}',

    -- Timestamps
    stage_entered_at TIMESTAMPTZ DEFAULT NOW(),  -- When entered current stage
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_deals_user ON deals(user_id);
CREATE INDEX IF NOT EXISTS idx_deals_pipeline ON deals(pipeline_id);
CREATE INDEX IF NOT EXISTS idx_deals_stage ON deals(stage_id);
CREATE INDEX IF NOT EXISTS idx_deals_company ON deals(company_id);
CREATE INDEX IF NOT EXISTS idx_deals_owner ON deals(owner_id);
CREATE INDEX IF NOT EXISTS idx_deals_status ON deals(user_id, status);
CREATE INDEX IF NOT EXISTS idx_deals_close_date ON deals(expected_close_date);
CREATE INDEX IF NOT EXISTS idx_deals_amount ON deals(user_id, amount DESC);

-- ============================================================================
-- CRM ACTIVITIES TABLE
-- ============================================================================

-- Activity types for CRM
DO $$ BEGIN
    CREATE TYPE crm_activity_type AS ENUM (
        'call',
        'email',
        'meeting',
        'note',
        'task',
        'demo',
        'proposal_sent',
        'contract_sent',
        'follow_up',
        'linkedin_message',
        'other'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS crm_activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- What type of activity
    activity_type crm_activity_type NOT NULL,

    -- Subject/title
    subject VARCHAR(500) NOT NULL,

    -- Details
    description TEXT,
    outcome TEXT,  -- What was the result

    -- Related entities (polymorphic)
    deal_id UUID REFERENCES deals(id) ON DELETE SET NULL,
    company_id UUID REFERENCES companies(id) ON DELETE SET NULL,
    contact_id UUID,  -- References clients table

    -- Participants
    participants JSONB DEFAULT '[]',
    -- Format: [{"type": "contact", "id": "...", "name": "..."}]

    -- Timing
    activity_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    duration_minutes INT,

    -- For calls
    call_direction VARCHAR(20),  -- 'inbound', 'outbound'
    call_disposition VARCHAR(50),  -- 'answered', 'voicemail', 'no_answer', 'busy'
    call_recording_url TEXT,

    -- For emails
    email_direction VARCHAR(20),  -- 'sent', 'received'
    email_message_id VARCHAR(255),

    -- For meetings
    meeting_location VARCHAR(255),
    meeting_url TEXT,

    -- Assignment
    owner_id VARCHAR(255),
    completed_by VARCHAR(255),

    -- Status
    is_completed BOOLEAN DEFAULT FALSE,
    completed_at TIMESTAMPTZ,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_crm_activities_user ON crm_activities(user_id);
CREATE INDEX IF NOT EXISTS idx_crm_activities_deal ON crm_activities(deal_id);
CREATE INDEX IF NOT EXISTS idx_crm_activities_company ON crm_activities(company_id);
CREATE INDEX IF NOT EXISTS idx_crm_activities_contact ON crm_activities(contact_id);
CREATE INDEX IF NOT EXISTS idx_crm_activities_date ON crm_activities(activity_date DESC);
CREATE INDEX IF NOT EXISTS idx_crm_activities_type ON crm_activities(activity_type);
CREATE INDEX IF NOT EXISTS idx_crm_activities_pending ON crm_activities(is_completed, activity_date) WHERE is_completed = FALSE;

-- ============================================================================
-- DEAL STAGE HISTORY (For funnel analytics)
-- ============================================================================

CREATE TABLE IF NOT EXISTS deal_stage_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_id UUID NOT NULL REFERENCES deals(id) ON DELETE CASCADE,

    -- Stage transition
    from_stage_id UUID REFERENCES pipeline_stages(id),
    to_stage_id UUID NOT NULL REFERENCES pipeline_stages(id),

    -- When and who
    changed_by VARCHAR(255),
    changed_at TIMESTAMPTZ DEFAULT NOW(),

    -- Duration in previous stage (seconds)
    duration_seconds INT,

    -- Snapshot of deal value at time of change
    deal_amount DECIMAL(15,2)
);

CREATE INDEX IF NOT EXISTS idx_deal_stage_history_deal ON deal_stage_history(deal_id);
CREATE INDEX IF NOT EXISTS idx_deal_stage_history_time ON deal_stage_history(changed_at DESC);

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Auto-update timestamps
DO $$
DECLARE
    tables TEXT[] := ARRAY[
        'companies', 'contact_company_relations', 'pipelines',
        'pipeline_stages', 'deals', 'crm_activities'
    ];
    t TEXT;
BEGIN
    FOREACH t IN ARRAY tables
    LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS %I_updated_at ON %I', t, t);
        EXECUTE format('CREATE TRIGGER %I_updated_at BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION update_custom_updated_at()', t, t);
    END LOOP;
END $$;

-- Track deal stage changes
CREATE OR REPLACE FUNCTION track_deal_stage_change()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.stage_id IS DISTINCT FROM NEW.stage_id THEN
        INSERT INTO deal_stage_history (
            deal_id, from_stage_id, to_stage_id,
            duration_seconds, deal_amount
        ) VALUES (
            NEW.id,
            OLD.stage_id,
            NEW.stage_id,
            EXTRACT(EPOCH FROM (NOW() - OLD.stage_entered_at))::INT,
            NEW.amount
        );

        -- Update stage_entered_at
        NEW.stage_entered_at := NOW();
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS deals_stage_change ON deals;
CREATE TRIGGER deals_stage_change
    BEFORE UPDATE ON deals
    FOR EACH ROW
    EXECUTE FUNCTION track_deal_stage_change();

-- ============================================================================
-- SEED DEFAULT PIPELINE
-- ============================================================================

-- Function to create default sales pipeline for a user
CREATE OR REPLACE FUNCTION create_default_pipeline(p_user_id VARCHAR(255))
RETURNS UUID AS $$
DECLARE
    v_pipeline_id UUID;
BEGIN
    -- Check if user already has a default pipeline
    SELECT id INTO v_pipeline_id
    FROM pipelines
    WHERE user_id = p_user_id AND is_default = TRUE;

    IF v_pipeline_id IS NOT NULL THEN
        RETURN v_pipeline_id;
    END IF;

    -- Create default pipeline
    INSERT INTO pipelines (user_id, name, description, is_default)
    VALUES (p_user_id, 'Sales Pipeline', 'Default sales pipeline', TRUE)
    RETURNING id INTO v_pipeline_id;

    -- Create default stages
    INSERT INTO pipeline_stages (pipeline_id, name, position, probability, stage_type, color) VALUES
        (v_pipeline_id, 'Lead', 0, 10, 'open', '#94A3B8'),
        (v_pipeline_id, 'Qualified', 1, 25, 'open', '#60A5FA'),
        (v_pipeline_id, 'Proposal', 2, 50, 'open', '#A78BFA'),
        (v_pipeline_id, 'Negotiation', 3, 75, 'open', '#FBBF24'),
        (v_pipeline_id, 'Won', 4, 100, 'won', '#34D399'),
        (v_pipeline_id, 'Lost', 5, 0, 'lost', '#F87171');

    RETURN v_pipeline_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE companies IS 'Organizations/companies in CRM';
COMMENT ON TABLE contact_company_relations IS 'Links contacts to companies with roles';
COMMENT ON TABLE pipelines IS 'Sales or other pipelines';
COMMENT ON TABLE pipeline_stages IS 'Stages within pipelines';
COMMENT ON TABLE deals IS 'Deals/opportunities in pipelines';
COMMENT ON TABLE crm_activities IS 'CRM activities: calls, emails, meetings, etc.';
COMMENT ON TABLE deal_stage_history IS 'History of deal stage transitions for analytics';
COMMENT ON FUNCTION create_default_pipeline IS 'Creates default sales pipeline for new users';

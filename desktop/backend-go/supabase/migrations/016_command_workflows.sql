-- Migration 015: Command Workflows
-- Enables multi-step command sequences with dependencies and parallel execution

-- Workflow definition table
CREATE TABLE IF NOT EXISTS command_workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Workflow metadata
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    trigger VARCHAR(100) NOT NULL, -- e.g., "/deploy" or "/release"

    -- Execution settings
    execution_mode VARCHAR(50) DEFAULT 'sequential', -- sequential, parallel, smart
    stop_on_failure BOOLEAN DEFAULT TRUE,
    timeout_seconds INTEGER DEFAULT 300,

    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    is_system BOOLEAN DEFAULT FALSE,

    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Workflow steps table
CREATE TABLE IF NOT EXISTS workflow_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES command_workflows(id) ON DELETE CASCADE,

    -- Step metadata
    name VARCHAR(255) NOT NULL,
    description TEXT,
    step_order INTEGER NOT NULL,

    -- Step action
    action_type VARCHAR(50) NOT NULL, -- command, agent, tool, condition, wait

    -- For action_type = 'command'
    command_trigger VARCHAR(100),
    command_args TEXT,

    -- For action_type = 'agent'
    target_agent_id UUID,
    prompt_template TEXT,

    -- For action_type = 'tool'
    tool_name VARCHAR(100),
    tool_params JSONB DEFAULT '{}',

    -- For action_type = 'condition'
    condition_expression TEXT, -- e.g., "{{previous.success}} == true"
    on_true_step UUID, -- step to go to if true
    on_false_step UUID, -- step to go to if false

    -- For action_type = 'wait'
    wait_seconds INTEGER DEFAULT 0,

    -- Dependencies
    depends_on UUID[], -- array of step IDs that must complete first
    can_parallel BOOLEAN DEFAULT FALSE,

    -- Error handling
    on_failure VARCHAR(50) DEFAULT 'stop', -- stop, continue, retry, skip
    max_retries INTEGER DEFAULT 0,
    retry_delay_seconds INTEGER DEFAULT 5,

    -- Context
    input_mapping JSONB DEFAULT '{}', -- map input from previous steps
    output_key VARCHAR(100), -- key to store output in context

    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Workflow execution logs
CREATE TABLE IF NOT EXISTS workflow_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES command_workflows(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,

    -- Execution context
    conversation_id UUID,
    initial_input TEXT,
    context JSONB DEFAULT '{}',

    -- Status
    status VARCHAR(50) DEFAULT 'pending', -- pending, running, completed, failed, cancelled
    current_step_id UUID,

    -- Results
    result JSONB DEFAULT '{}',
    error_message TEXT,

    -- Timing
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Step execution logs
CREATE TABLE IF NOT EXISTS workflow_step_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    step_id UUID NOT NULL REFERENCES workflow_steps(id) ON DELETE CASCADE,

    -- Status
    status VARCHAR(50) DEFAULT 'pending', -- pending, running, completed, failed, skipped
    attempt_number INTEGER DEFAULT 1,

    -- Results
    input JSONB DEFAULT '{}',
    output JSONB DEFAULT '{}',
    error_message TEXT,

    -- Timing
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms FLOAT,

    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_workflows_user ON command_workflows(user_id);
CREATE INDEX IF NOT EXISTS idx_workflows_trigger ON command_workflows(trigger);
CREATE INDEX IF NOT EXISTS idx_workflow_steps_workflow ON workflow_steps(workflow_id);
CREATE INDEX IF NOT EXISTS idx_workflow_steps_order ON workflow_steps(workflow_id, step_order);
CREATE INDEX IF NOT EXISTS idx_workflow_executions_workflow ON workflow_executions(workflow_id);
CREATE INDEX IF NOT EXISTS idx_workflow_executions_user ON workflow_executions(user_id);
CREATE INDEX IF NOT EXISTS idx_workflow_executions_status ON workflow_executions(status);
CREATE INDEX IF NOT EXISTS idx_step_executions_execution ON workflow_step_executions(execution_id);

-- Unique constraint for workflow triggers per user
CREATE UNIQUE INDEX IF NOT EXISTS idx_workflows_user_trigger ON command_workflows(user_id, trigger) WHERE is_active = TRUE;

-- Comments
COMMENT ON TABLE command_workflows IS 'Multi-step command workflows for complex automation';
COMMENT ON TABLE workflow_steps IS 'Individual steps within a workflow';
COMMENT ON TABLE workflow_executions IS 'Execution history and status of workflow runs';
COMMENT ON TABLE workflow_step_executions IS 'Execution history of individual steps';

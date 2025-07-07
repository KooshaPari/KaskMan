-- Persistent R&D Platform Database Schema
-- PostgreSQL 15+ with extensions for JSON, UUID, and full-text search

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- ============================================================================
-- PROJECT MANAGEMENT TABLES
-- ============================================================================

-- Projects table - main project lifecycle management
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL, -- research, development, analysis, maintenance
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, active, completed, paused, cancelled
    priority INTEGER NOT NULL DEFAULT 5, -- 1-10 scale
    objectives JSONB NOT NULL DEFAULT '[]',
    success_criteria JSONB NOT NULL DEFAULT '[]',
    resources_required JSONB NOT NULL DEFAULT '{}',
    estimated_duration INTERVAL,
    actual_duration INTERVAL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_by UUID,
    assigned_to UUID[],
    metadata JSONB DEFAULT '{}'
);

-- Project tasks - granular task management
CREATE TABLE project_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    parent_task_id UUID REFERENCES project_tasks(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL, -- code, test, research, design, review
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    priority INTEGER NOT NULL DEFAULT 5,
    estimated_time INTERVAL,
    actual_time INTERVAL,
    dependencies UUID[] DEFAULT '{}',
    outputs JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    assigned_agent VARCHAR(100),
    metadata JSONB DEFAULT '{}'
);

-- Project artifacts - files, documents, code generated
CREATE TABLE project_artifacts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    task_id UUID REFERENCES project_tasks(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- code, document, data, model, config
    path TEXT NOT NULL,
    size_bytes BIGINT,
    content_hash VARCHAR(64),
    version INTEGER NOT NULL DEFAULT 1,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(100),
    metadata JSONB DEFAULT '{}'
);

-- ============================================================================
-- LEARNING AND KNOWLEDGE MANAGEMENT TABLES
-- ============================================================================

-- Knowledge base - persistent learning storage
CREATE TABLE knowledge_base (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    category VARCHAR(100) NOT NULL, -- research, patterns, solutions, errors
    topic VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    source VARCHAR(255), -- project_id, external_url, agent_name
    confidence_score DECIMAL(3,2) DEFAULT 0.5, -- 0.0-1.0
    usage_count INTEGER DEFAULT 0,
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    tags TEXT[] DEFAULT '{}',
    embeddings VECTOR(1536), -- OpenAI embeddings
    metadata JSONB DEFAULT '{}'
);

-- Learning models - AI model configurations and weights
CREATE TABLE learning_models (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL, -- classification, regression, clustering, generation
    architecture VARCHAR(100) NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    training_data_source TEXT,
    hyperparameters JSONB NOT NULL DEFAULT '{}',
    metrics JSONB NOT NULL DEFAULT '{}',
    model_path TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    performance_history JSONB DEFAULT '[]',
    metadata JSONB DEFAULT '{}'
);

-- Learning outcomes - results from completed projects
CREATE TABLE learning_outcomes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    outcome_type VARCHAR(50) NOT NULL, -- success, failure, partial, insight
    description TEXT NOT NULL,
    impact_score DECIMAL(3,2) DEFAULT 0.0, -- -1.0 to 1.0
    lessons_learned TEXT[],
    patterns_identified JSONB DEFAULT '[]',
    recommendations TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    validated_at TIMESTAMP WITH TIME ZONE,
    applied_count INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}'
);

-- ============================================================================
-- SYSTEM OPERATIONS TABLES
-- ============================================================================

-- Agents - AI agent instances and configurations
CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL, -- researcher, coder, analyst, architect
    status VARCHAR(50) NOT NULL DEFAULT 'idle', -- idle, busy, offline, error
    capabilities TEXT[] NOT NULL DEFAULT '{}',
    current_task_id UUID REFERENCES project_tasks(id),
    performance_metrics JSONB DEFAULT '{}',
    configuration JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_active_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    total_tasks_completed INTEGER DEFAULT 0,
    success_rate DECIMAL(3,2) DEFAULT 0.0,
    metadata JSONB DEFAULT '{}'
);

-- System metrics - operational metrics and performance data
CREATE TABLE system_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    metric_name VARCHAR(100) NOT NULL,
    metric_type VARCHAR(50) NOT NULL, -- counter, gauge, histogram, summary
    value DECIMAL(15,6) NOT NULL,
    unit VARCHAR(20),
    labels JSONB DEFAULT '{}',
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    source VARCHAR(100) NOT NULL,
    metadata JSONB DEFAULT '{}'
);

-- System events - audit trail and event logging
CREATE TABLE system_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type VARCHAR(50) NOT NULL,
    event_name VARCHAR(255) NOT NULL,
    description TEXT,
    severity VARCHAR(20) NOT NULL DEFAULT 'info', -- debug, info, warning, error, critical
    source VARCHAR(100) NOT NULL,
    user_id UUID,
    project_id UUID REFERENCES projects(id),
    task_id UUID REFERENCES project_tasks(id),
    agent_id UUID REFERENCES agents(id),
    payload JSONB DEFAULT '{}',
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    correlation_id UUID,
    metadata JSONB DEFAULT '{}'
);

-- ============================================================================
-- MEMORY AND SESSIONS TABLES
-- ============================================================================

-- Memory entries - persistent memory system
CREATE TABLE memory_entries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key VARCHAR(500) NOT NULL UNIQUE,
    value JSONB NOT NULL,
    type VARCHAR(50) NOT NULL DEFAULT 'general',
    access_count INTEGER DEFAULT 0,
    last_accessed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}'
);

-- Sessions - user and agent sessions
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_type VARCHAR(50) NOT NULL, -- user, agent, system
    user_id UUID,
    agent_id UUID REFERENCES agents(id),
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_activity_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ended_at TIMESTAMP WITH TIME ZONE,
    context JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}'
);

-- ============================================================================
-- INDEXES FOR PERFORMANCE
-- ============================================================================

-- Projects indexes
CREATE INDEX idx_projects_status ON projects(status);
CREATE INDEX idx_projects_type ON projects(type);
CREATE INDEX idx_projects_priority ON projects(priority);
CREATE INDEX idx_projects_created_at ON projects(created_at);
CREATE INDEX idx_projects_metadata ON projects USING GIN(metadata);

-- Tasks indexes
CREATE INDEX idx_tasks_project_id ON project_tasks(project_id);
CREATE INDEX idx_tasks_status ON project_tasks(status);
CREATE INDEX idx_tasks_type ON project_tasks(type);
CREATE INDEX idx_tasks_assigned_agent ON project_tasks(assigned_agent);
CREATE INDEX idx_tasks_dependencies ON project_tasks USING GIN(dependencies);

-- Knowledge base indexes
CREATE INDEX idx_knowledge_category ON knowledge_base(category);
CREATE INDEX idx_knowledge_topic ON knowledge_base(topic);
CREATE INDEX idx_knowledge_confidence ON knowledge_base(confidence_score);
CREATE INDEX idx_knowledge_usage ON knowledge_base(usage_count);
CREATE INDEX idx_knowledge_tags ON knowledge_base USING GIN(tags);
CREATE INDEX idx_knowledge_content_fts ON knowledge_base USING GIN(to_tsvector('english', content));

-- System metrics indexes
CREATE INDEX idx_metrics_name_time ON system_metrics(metric_name, timestamp);
CREATE INDEX idx_metrics_source ON system_metrics(source);
CREATE INDEX idx_metrics_labels ON system_metrics USING GIN(labels);

-- Events indexes
CREATE INDEX idx_events_type ON system_events(event_type);
CREATE INDEX idx_events_timestamp ON system_events(timestamp);
CREATE INDEX idx_events_source ON system_events(source);
CREATE INDEX idx_events_severity ON system_events(severity);
CREATE INDEX idx_events_correlation ON system_events(correlation_id);

-- Memory indexes
CREATE INDEX idx_memory_key ON memory_entries(key);
CREATE INDEX idx_memory_type ON memory_entries(type);
CREATE INDEX idx_memory_access_count ON memory_entries(access_count);
CREATE INDEX idx_memory_tags ON memory_entries USING GIN(tags);

-- ============================================================================
-- VIEWS FOR COMMON QUERIES
-- ============================================================================

-- Active projects with task counts
CREATE VIEW active_projects_summary AS
SELECT 
    p.id,
    p.name,
    p.type,
    p.status,
    p.priority,
    p.created_at,
    COUNT(t.id) as total_tasks,
    COUNT(CASE WHEN t.status = 'completed' THEN 1 END) as completed_tasks,
    COUNT(CASE WHEN t.status = 'pending' THEN 1 END) as pending_tasks,
    COUNT(CASE WHEN t.status = 'in_progress' THEN 1 END) as active_tasks
FROM projects p
LEFT JOIN project_tasks t ON p.id = t.project_id
WHERE p.status IN ('active', 'pending')
GROUP BY p.id, p.name, p.type, p.status, p.priority, p.created_at;

-- Agent performance summary
CREATE VIEW agent_performance_summary AS
SELECT 
    a.id,
    a.name,
    a.type,
    a.status,
    a.total_tasks_completed,
    a.success_rate,
    COUNT(t.id) as current_tasks,
    a.last_active_at
FROM agents a
LEFT JOIN project_tasks t ON a.id::text = t.assigned_agent AND t.status IN ('pending', 'in_progress')
GROUP BY a.id, a.name, a.type, a.status, a.total_tasks_completed, a.success_rate, a.last_active_at;

-- System health metrics
CREATE VIEW system_health_metrics AS
SELECT 
    'total_projects' as metric,
    COUNT(*)::text as value,
    NOW() as timestamp
FROM projects
UNION ALL
SELECT 
    'active_projects' as metric,
    COUNT(*)::text as value,
    NOW() as timestamp
FROM projects WHERE status = 'active'
UNION ALL
SELECT 
    'total_agents' as metric,
    COUNT(*)::text as value,
    NOW() as timestamp
FROM agents
UNION ALL
SELECT 
    'active_agents' as metric,
    COUNT(*)::text as value,
    NOW() as timestamp
FROM agents WHERE status IN ('idle', 'busy');
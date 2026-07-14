CREATE TABLE market_segments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    name text NOT NULL,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('draft', 'active', 'archived')),
    definition jsonb NOT NULL DEFAULT '{}',
    version integer NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name)
);

CREATE TABLE icp_definitions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    market_segment_id uuid NOT NULL REFERENCES market_segments(id),
    name text NOT NULL,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('draft', 'active', 'retired')),
    definition jsonb NOT NULL DEFAULT '{}',
    version integer NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, market_segment_id, name, version)
);

CREATE TABLE leads (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    market_segment_id uuid NOT NULL REFERENCES market_segments(id),
    icp_definition_id uuid REFERENCES icp_definitions(id),
    name text NOT NULL,
    status text NOT NULL DEFAULT 'discovered' CHECK (status IN ('discovered', 'enriched', 'qualified', 'proof_requested', 'proof_ready', 'approved_for_outreach', 'contacted', 'replied', 'meeting', 'proposal', 'won', 'lost', 'suppressed')),
    score integer NOT NULL DEFAULT 0 CHECK (score BETWEEN 0 AND 100),
    version integer NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE lead_evidence (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    lead_id uuid NOT NULL REFERENCES leads(id),
    kind text NOT NULL,
    summary text NOT NULL,
    confidence integer NOT NULL CHECK (confidence BETWEEN 0 AND 100),
    source_ref text NOT NULL DEFAULT '',
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE contacts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    lead_id uuid NOT NULL REFERENCES leads(id),
    channel text NOT NULL CHECK (channel IN ('email', 'phone', 'web', 'custom')),
    value text NOT NULL,
    normalized_value text NOT NULL,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'invalid', 'suppressed')),
    consent_status text NOT NULL DEFAULT 'unknown' CHECK (consent_status IN ('unknown', 'opted_in', 'opted_out')),
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, channel, normalized_value)
);

CREATE TABLE contact_sources (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    contact_id uuid NOT NULL REFERENCES contacts(id),
    source_type text NOT NULL,
    source_ref text NOT NULL,
    evidence jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE proof_templates (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    name text NOT NULL,
    proof_type text NOT NULL CHECK (proof_type IN ('report', 'sample', 'comparison', 'prototype', 'analysis', 'audit', 'simulation', 'document', 'media', 'custom')),
    workflow_version_id uuid NOT NULL REFERENCES workflow_definitions(id),
    input_schema jsonb NOT NULL,
    output_schema jsonb NOT NULL,
    access_policy jsonb NOT NULL DEFAULT '{}',
    retention_days integer NOT NULL DEFAULT 30 CHECK (retention_days BETWEEN 1 AND 3650),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('draft', 'active', 'retired')),
    version integer NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name, version)
);

CREATE TABLE deals (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    lead_id uuid NOT NULL REFERENCES leads(id),
    name text NOT NULL,
    customer_id text NOT NULL,
    status text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'proposal', 'won', 'lost', 'cancelled')),
    currency char(3) NOT NULL,
    value_minor bigint NOT NULL DEFAULT 0 CHECK (value_minor >= 0),
    version integer NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    closed_at timestamptz
);

ALTER TABLE quotes
    ADD COLUMN growth_deal_id uuid REFERENCES deals(id);

CREATE TABLE proof_requests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    lead_id uuid NOT NULL REFERENCES leads(id),
    deal_id uuid REFERENCES deals(id),
    template_id uuid NOT NULL REFERENCES proof_templates(id),
    status text NOT NULL DEFAULT 'requested' CHECK (status IN ('requested', 'processing', 'review', 'ready', 'rejected', 'expired', 'deleted')),
    input jsonb NOT NULL DEFAULT '{}',
    requested_by text NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE proof_instances (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    proof_request_id uuid NOT NULL REFERENCES proof_requests(id),
    status text NOT NULL CHECK (status IN ('generated', 'approved', 'rejected', 'deleted')),
    result jsonb NOT NULL DEFAULT '{}',
    artifact_ref text NOT NULL DEFAULT '',
    review_rationale text NOT NULL DEFAULT '',
    generated_by text NOT NULL,
    reviewed_by text,
    created_at timestamptz NOT NULL DEFAULT now(),
    reviewed_at timestamptz,
    expires_at timestamptz NOT NULL,
    UNIQUE (tenant_id, proof_request_id)
);

CREATE TABLE campaigns (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    market_segment_id uuid REFERENCES market_segments(id),
    name text NOT NULL,
    channel text NOT NULL CHECK (channel IN ('email', 'phone', 'web', 'custom')),
    purpose text NOT NULL,
    status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'pending_approval', 'approved', 'rejected', 'active', 'paused', 'completed', 'cancelled')),
    version integer NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name)
);

CREATE TABLE campaign_steps (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    campaign_id uuid NOT NULL REFERENCES campaigns(id),
    position integer NOT NULL CHECK (position > 0),
    kind text NOT NULL CHECK (kind IN ('message', 'wait', 'condition', 'proof_request', 'manual_task')),
    definition jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, campaign_id, position)
);

CREATE TABLE campaign_approvals (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    campaign_id uuid NOT NULL REFERENCES campaigns(id),
    campaign_version integer NOT NULL CHECK (campaign_version > 0),
    decision text NOT NULL CHECK (decision IN ('approved', 'rejected')),
    rationale text NOT NULL,
    reviewed_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, campaign_id, campaign_version)
);

CREATE TABLE suppression_entries (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    subject_type text NOT NULL CHECK (subject_type IN ('lead', 'contact', 'address', 'domain')),
    subject_id text NOT NULL,
    subject_key char(64) NOT NULL,
    channel text NOT NULL CHECK (channel IN ('all', 'email', 'phone', 'web', 'custom')),
    reason text NOT NULL CHECK (reason IN ('do_not_contact', 'opt_out', 'bounce', 'complaint', 'manual', 'risk')),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'released')),
    source_ref text NOT NULL DEFAULT '',
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    released_at timestamptz
);

CREATE UNIQUE INDEX suppression_active_unique
    ON suppression_entries (tenant_id, subject_key, channel)
    WHERE status='active';

CREATE TABLE send_quotas (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    channel text NOT NULL CHECK (channel IN ('email', 'phone', 'web', 'custom')),
    period_start date NOT NULL,
    period_end date NOT NULL,
    limit_count integer NOT NULL CHECK (limit_count > 0),
    reserved_count integer NOT NULL DEFAULT 0 CHECK (reserved_count >= 0),
    consumed_count integer NOT NULL DEFAULT 0 CHECK (consumed_count >= 0),
    CHECK (period_end >= period_start),
    CHECK (reserved_count + consumed_count <= limit_count),
    UNIQUE (tenant_id, channel, period_start, period_end)
);

CREATE TABLE outreach_messages (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    campaign_id uuid NOT NULL REFERENCES campaigns(id),
    campaign_step_id uuid NOT NULL REFERENCES campaign_steps(id),
    lead_id uuid NOT NULL REFERENCES leads(id),
    contact_id uuid REFERENCES contacts(id),
    status text NOT NULL CHECK (status IN ('planned', 'blocked', 'sent', 'delivered', 'replied', 'bounced', 'complained', 'cancelled')),
    content jsonb NOT NULL DEFAULT '{}',
    block_reason text NOT NULL DEFAULT '',
    external_message_id text,
    idempotency_key text NOT NULL,
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, idempotency_key)
);

CREATE TABLE conversations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    lead_id uuid NOT NULL REFERENCES leads(id),
    deal_id uuid REFERENCES deals(id),
    channel text NOT NULL CHECK (channel IN ('email', 'phone', 'web', 'custom')),
    status text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    last_message_at timestamptz
);

CREATE TABLE conversation_messages (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    conversation_id uuid NOT NULL REFERENCES conversations(id),
    direction text NOT NULL CHECK (direction IN ('inbound', 'outbound', 'system')),
    status text NOT NULL CHECK (status IN ('received', 'draft', 'planned', 'sent', 'failed')),
    content jsonb NOT NULL DEFAULT '{}',
    idempotency_key text NOT NULL,
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, idempotency_key)
);

CREATE TABLE experiments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    name text NOT NULL,
    entity_type text NOT NULL CHECK (entity_type IN ('market_segment', 'lead', 'proof', 'campaign', 'deal')),
    entity_id uuid NOT NULL,
    hypothesis text NOT NULL,
    status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'running', 'completed', 'cancelled')),
    allocation_basis_points integer NOT NULL CHECK (allocation_basis_points BETWEEN 0 AND 10000),
    metrics_definition jsonb NOT NULL DEFAULT '{}',
    result jsonb NOT NULL DEFAULT '{}',
    version integer NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name)
);

CREATE INDEX idx_leads_segment_status ON leads (tenant_id, market_segment_id, status, updated_at DESC);
CREATE INDEX idx_lead_evidence_created ON lead_evidence (tenant_id, lead_id, created_at);
CREATE INDEX idx_proof_requests_status ON proof_requests (tenant_id, status, updated_at DESC);
CREATE INDEX idx_campaigns_status ON campaigns (tenant_id, status, updated_at DESC);
CREATE INDEX idx_outreach_messages_status ON outreach_messages (tenant_id, status, created_at DESC);
CREATE INDEX idx_conversations_lead ON conversations (tenant_id, lead_id, updated_at DESC);
CREATE INDEX idx_deals_status ON deals (tenant_id, status, updated_at DESC);
CREATE INDEX idx_quotes_growth_deal ON quotes (tenant_id, growth_deal_id) WHERE growth_deal_id IS NOT NULL;

DO $$
DECLARE
    table_name text;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'market_segments', 'icp_definitions', 'leads', 'lead_evidence', 'contacts',
        'contact_sources', 'proof_templates', 'deals', 'proof_requests', 'proof_instances',
        'campaigns', 'campaign_steps', 'campaign_approvals', 'suppression_entries',
        'send_quotas', 'outreach_messages', 'conversations', 'conversation_messages',
        'experiments'
    ] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', table_name);
        EXECUTE format(
            'CREATE POLICY tenant_isolation ON %I USING (tenant_id = NULLIF(current_setting(''app.tenant_id'', true), '''')::uuid) WITH CHECK (tenant_id = NULLIF(current_setting(''app.tenant_id'', true), '''')::uuid)',
            table_name
        );
    END LOOP;
END $$;

INSERT INTO feature_definitions (key, description, default_enabled) VALUES
    ('growth.core', 'Product-agnostic segment, lead, proof, campaign, deal, and experiment control plane', true),
    ('growth.outreach_planning', 'Approved and suppression-checked outreach planning', true),
    ('growth.outbound_delivery', 'External outbound message delivery adapters', false)
ON CONFLICT (key) DO UPDATE SET description=EXCLUDED.description, default_enabled=EXCLUDED.default_enabled;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO opportunity_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO opportunity_app;

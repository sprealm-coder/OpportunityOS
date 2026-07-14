ALTER TABLE sources
    ADD COLUMN status text NOT NULL DEFAULT 'active' CHECK (status IN ('active','paused','retired')),
    ADD COLUMN version integer NOT NULL DEFAULT 1 CHECK (version > 0),
    ADD COLUMN created_by text NOT NULL DEFAULT 'system',
    ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();
CREATE UNIQUE INDEX sources_tenant_name_unique ON sources (tenant_id,name);

ALTER TABLE signals
    ADD COLUMN external_id text,
    ADD COLUMN status text NOT NULL DEFAULT 'imported' CHECK (status IN ('imported','normalized','promoted','rejected')),
    ADD COLUMN normalized jsonb NOT NULL DEFAULT '{}',
    ADD COLUMN occurred_at timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN imported_by text NOT NULL DEFAULT 'system',
    ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();
CREATE UNIQUE INDEX signals_source_external_unique ON signals (tenant_id,source_id,external_id) WHERE external_id IS NOT NULL;
CREATE INDEX signals_tenant_status_created ON signals (tenant_id,status,created_at DESC);

CREATE TABLE opportunity_signals (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    opportunity_id uuid NOT NULL REFERENCES opportunities(id),
    signal_id uuid NOT NULL REFERENCES signals(id),
    linked_by text NOT NULL,
    linked_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id,opportunity_id,signal_id),
    UNIQUE (tenant_id,signal_id)
);

ALTER TABLE opportunities
    ADD COLUMN outcome_feedback_count integer NOT NULL DEFAULT 0 CHECK (outcome_feedback_count >= 0),
    ADD COLUMN outcome_metrics jsonb NOT NULL DEFAULT '{}',
    ADD COLUMN outcome_updated_at timestamptz;

ALTER TABLE outcome_feedback
    ADD COLUMN execution_order_id uuid REFERENCES execution_orders(id),
    ADD COLUMN status text NOT NULL DEFAULT 'accepted' CHECK (status IN ('accepted','superseded','reversed')),
    ADD COLUMN evidence jsonb NOT NULL DEFAULT '{}',
    ADD COLUMN idempotency_key text,
    ADD COLUMN version integer NOT NULL DEFAULT 1 CHECK (version > 0),
    ADD COLUMN created_by text NOT NULL DEFAULT 'system',
    ADD COLUMN validated_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE outcome_feedback ALTER COLUMN order_id SET NOT NULL;
CREATE UNIQUE INDEX outcome_feedback_idempotency_unique ON outcome_feedback (tenant_id,idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE UNIQUE INDEX outcome_feedback_order_version_unique ON outcome_feedback (tenant_id,opportunity_id,order_id,version);
CREATE INDEX outcome_feedback_opportunity_created ON outcome_feedback (tenant_id,opportunity_id,created_at DESC);

ALTER TABLE workflow_runs
    ADD COLUMN execution_order_id uuid REFERENCES execution_orders(id),
    ADD COLUMN created_by text NOT NULL DEFAULT 'system',
    ADD COLUMN started_at timestamptz,
    ADD COLUMN completed_at timestamptz,
    ADD COLUMN last_error jsonb NOT NULL DEFAULT '{}';
CREATE UNIQUE INDEX workflow_run_execution_unique ON workflow_runs (tenant_id,execution_order_id) WHERE execution_order_id IS NOT NULL;
CREATE INDEX workflow_run_status_updated ON workflow_runs (tenant_id,status,updated_at DESC);

ALTER TABLE workflow_step_runs
    ADD COLUMN execution_order_id uuid REFERENCES execution_orders(id),
    ADD COLUMN node_type text NOT NULL DEFAULT 'realtime_call',
    ADD COLUMN max_attempts integer NOT NULL DEFAULT 3 CHECK (max_attempts > 0),
    ADD COLUMN locked_by text,
    ADD COLUMN locked_until timestamptz,
    ADD COLUMN next_retry_at timestamptz,
    ADD COLUMN last_error jsonb NOT NULL DEFAULT '{}';
CREATE INDEX workflow_step_leaseable ON workflow_step_runs (tenant_id,status,next_retry_at,locked_until)
    WHERE status IN ('pending','retry_wait','leased');

CREATE TABLE adapter_identities (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    name text NOT NULL,
    key_id text NOT NULL,
    provider_endpoint_id uuid NOT NULL REFERENCES provider_endpoints(id),
    secret_ref text NOT NULL,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active','rotating','disabled')),
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (key_id),
    UNIQUE (tenant_id,name),
    UNIQUE (tenant_id,provider_endpoint_id,key_id)
);

CREATE TABLE adapter_result_receipts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    adapter_identity_id uuid NOT NULL REFERENCES adapter_identities(id),
    execution_order_id uuid NOT NULL REFERENCES execution_orders(id),
    workflow_step_id uuid NOT NULL REFERENCES workflow_step_runs(id),
    external_event_id text NOT NULL,
    nonce text NOT NULL,
    signature_digest char(64) NOT NULL,
    result_status text NOT NULL CHECK (result_status IN ('submitted','processing','succeeded','failed','unknown')),
    payload jsonb NOT NULL,
    received_at timestamptz NOT NULL DEFAULT now(),
    processed_at timestamptz,
    UNIQUE (tenant_id,adapter_identity_id,external_event_id),
    UNIQUE (tenant_id,adapter_identity_id,nonce)
);
CREATE INDEX adapter_receipt_execution_created ON adapter_result_receipts (tenant_id,execution_order_id,received_at DESC);

CREATE TABLE outbox_replays (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    outbox_event_id uuid NOT NULL REFERENCES outbox_events(id),
    reason text NOT NULL,
    previous_retry_count integer NOT NULL,
    requested_by text NOT NULL,
    requested_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE operational_alerts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    alert_type text NOT NULL,
    severity text NOT NULL CHECK (severity IN ('info','warning','critical')),
    status text NOT NULL DEFAULT 'open' CHECK (status IN ('open','acknowledged','resolved')),
    object_type text NOT NULL,
    object_id text NOT NULL,
    message text NOT NULL,
    details jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    acknowledged_by text,
    acknowledged_at timestamptz,
    resolved_at timestamptz
);
CREATE UNIQUE INDEX operational_alert_active_unique ON operational_alerts (tenant_id,alert_type,object_type,object_id) WHERE status IN ('open','acknowledged');
CREATE INDEX operational_alert_status_created ON operational_alerts (tenant_id,status,created_at DESC);

DO $$
DECLARE table_name text;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'opportunity_signals','adapter_identities','adapter_result_receipts','outbox_replays','operational_alerts'
    ] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', table_name);
        EXECUTE format('CREATE POLICY tenant_isolation ON %I USING (tenant_id = NULLIF(current_setting(''app.tenant_id'', true), '''')::uuid) WITH CHECK (tenant_id = NULLIF(current_setting(''app.tenant_id'', true), '''')::uuid)', table_name);
    END LOOP;
END $$;

INSERT INTO feature_definitions (key,description,default_enabled) VALUES
    ('intelligence.signal_ingress','Source, Signal, deduplication, normalization, and Opportunity lineage',true),
    ('workflow.persistent_leases','Persistent workflow step leases and retry scheduling',true),
    ('execution.trusted_adapter_ingress','Signed and replay-protected Adapter result ingestion',true),
    ('analytics.outcome_feedback','Validated commercial outcome feedback and Opportunity projection',true),
    ('operations.outbox_replay','Audited Outbox dead-letter replay and operational alerts',true)
ON CONFLICT (key) DO UPDATE SET description=EXCLUDED.description,default_enabled=EXCLUDED.default_enabled;

GRANT SELECT,INSERT,UPDATE,DELETE ON ALL TABLES IN SCHEMA public TO opportunity_app;
GRANT USAGE,SELECT ON ALL SEQUENCES IN SCHEMA public TO opportunity_app;

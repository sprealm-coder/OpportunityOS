CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text NOT NULL,
    display_name text NOT NULL,
    password_hash text NOT NULL,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (email)
);

CREATE TABLE auth_sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    user_id uuid NOT NULL REFERENCES users(id),
    token_hash char(64) NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE outbox_events
    ADD COLUMN locked_by text,
    ADD COLUMN locked_until timestamptz,
    ADD COLUMN published_at timestamptz,
    ADD COLUMN last_error text;

ALTER TABLE inbox_events
    ADD COLUMN received_at timestamptz NOT NULL DEFAULT now();

CREATE INDEX idx_auth_sessions_active
    ON auth_sessions (token_hash, expires_at)
    WHERE revoked_at IS NULL;

CREATE INDEX idx_outbox_leaseable
    ON outbox_events (next_retry_at, occurred_at)
    WHERE processed_at IS NULL AND dead_letter_reason IS NULL;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'opportunity_app') THEN
        CREATE ROLE opportunity_app NOLOGIN;
    END IF;
END $$;

GRANT USAGE ON SCHEMA public TO opportunity_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO opportunity_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO opportunity_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO opportunity_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO opportunity_app;

ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON tenants;
CREATE POLICY tenant_isolation ON tenants
    USING (id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

DO $$
DECLARE
    table_name text;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'brands', 'memberships', 'sources', 'signals', 'opportunities',
        'opportunity_evidence', 'opportunity_reviews', 'incubation_projects',
        'business_blueprints', 'capabilities', 'providers', 'provider_endpoints',
        'products', 'product_versions', 'skus', 'sku_versions',
        'workflow_definitions', 'workflow_runs', 'workflow_step_runs',
        'metering_definitions', 'price_books', 'price_rules', 'route_policies',
        'orders', 'usage_records', 'provider_costs', 'customer_charges',
        'ledger_accounts', 'ledger_transactions', 'ledger_entries',
        'outcome_feedback', 'tenant_features', 'audit_log', 'outbox_events',
        'inbox_events', 'command_idempotency', 'auth_sessions'
    ] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', table_name);
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', table_name);
        EXECUTE format(
            'CREATE POLICY tenant_isolation ON %I USING (tenant_id = NULLIF(current_setting(''app.tenant_id'', true), '''')::uuid) WITH CHECK (tenant_id = NULLIF(current_setting(''app.tenant_id'', true), '''')::uuid)',
            table_name
        );
    END LOOP;
END $$;

CREATE TABLE reseller_levels (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id),
    name text NOT NULL, rank integer NOT NULL CHECK (rank > 0), default_commission_bps integer NOT NULL DEFAULT 0 CHECK (default_commission_bps BETWEEN 0 AND 10000),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active','retired')), created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE (tenant_id,name), UNIQUE (tenant_id,rank)
);

CREATE TABLE resellers (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), level_id uuid REFERENCES reseller_levels(id),
    name text NOT NULL, status text NOT NULL DEFAULT 'active' CHECK (status IN ('draft','active','suspended','terminated')),
    created_by text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE (tenant_id,name)
);

CREATE TABLE reseller_members (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), reseller_id uuid NOT NULL REFERENCES resellers(id),
    user_id text NOT NULL, role text NOT NULL CHECK (role IN ('owner','manager','member','finance')), status text NOT NULL DEFAULT 'active' CHECK (status IN ('active','suspended','removed')),
    created_at timestamptz NOT NULL DEFAULT now(), UNIQUE (tenant_id,reseller_id,user_id)
);

CREATE TABLE attribution_rules (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), name text NOT NULL,
    priority integer NOT NULL CHECK (priority > 0), definition jsonb NOT NULL DEFAULT '{}', status text NOT NULL DEFAULT 'active' CHECK (status IN ('draft','active','retired')),
    version integer NOT NULL DEFAULT 1 CHECK (version > 0), created_by text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id,name,version)
);
CREATE UNIQUE INDEX attribution_rule_active_priority_unique ON attribution_rules (tenant_id,priority) WHERE status='active';

CREATE TABLE lead_ownerships (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), lead_id uuid NOT NULL REFERENCES leads(id),
    reseller_id uuid NOT NULL REFERENCES resellers(id), attribution_rule_id uuid NOT NULL REFERENCES attribution_rules(id),
    status text NOT NULL DEFAULT 'protected' CHECK (status IN ('protected','transferred','released','conflicted')),
    protection_expires_at timestamptz NOT NULL, version integer NOT NULL DEFAULT 1 CHECK (version > 0), created_by text NOT NULL,
    acquired_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX lead_ownership_active_unique ON lead_ownerships (tenant_id,lead_id) WHERE status IN ('protected','conflicted');

CREATE TABLE customer_ownerships (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), customer_id text NOT NULL,
    reseller_id uuid NOT NULL REFERENCES resellers(id), source_lead_ownership_id uuid REFERENCES lead_ownerships(id),
    status text NOT NULL DEFAULT 'protected' CHECK (status IN ('protected','transferred','released','conflicted')),
    protection_expires_at timestamptz NOT NULL, version integer NOT NULL DEFAULT 1 CHECK (version > 0), created_by text NOT NULL,
    acquired_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX customer_ownership_active_unique ON customer_ownerships (tenant_id,customer_id) WHERE status IN ('protected','conflicted');

CREATE TABLE transfer_requests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), ownership_type text NOT NULL CHECK (ownership_type IN ('lead','customer')),
    ownership_id uuid NOT NULL, from_reseller_id uuid NOT NULL REFERENCES resellers(id), to_reseller_id uuid NOT NULL REFERENCES resellers(id),
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected','cancelled')), rationale text NOT NULL,
    requested_by text NOT NULL, reviewed_by text, version integer NOT NULL DEFAULT 1, created_at timestamptz NOT NULL DEFAULT now(), reviewed_at timestamptz,
    CHECK (from_reseller_id <> to_reseller_id)
);
CREATE UNIQUE INDEX transfer_request_pending_unique ON transfer_requests (tenant_id,ownership_type,ownership_id) WHERE status='pending';

CREATE TABLE conflict_records (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), ownership_type text NOT NULL CHECK (ownership_type IN ('lead','customer')),
    ownership_id uuid NOT NULL, claimant_reseller_ids uuid[] NOT NULL, status text NOT NULL DEFAULT 'open' CHECK (status IN ('open','resolved','dismissed')),
    resolution jsonb NOT NULL DEFAULT '{}', created_by text NOT NULL, resolved_by text, created_at timestamptz NOT NULL DEFAULT now(), resolved_at timestamptz
);

CREATE TABLE commission_rules (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), name text NOT NULL,
    reseller_id uuid REFERENCES resellers(id), reseller_level_id uuid REFERENCES reseller_levels(id), trigger_type text NOT NULL DEFAULT 'customer_charge' CHECK (trigger_type='customer_charge'),
    basis_points integer NOT NULL CHECK (basis_points BETWEEN 0 AND 10000), status text NOT NULL DEFAULT 'active' CHECK (status IN ('draft','active','retired')),
    version integer NOT NULL DEFAULT 1, effective_from timestamptz NOT NULL DEFAULT now(), effective_until timestamptz, created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), CHECK (reseller_id IS NOT NULL OR reseller_level_id IS NOT NULL), UNIQUE (tenant_id,name,version)
);

CREATE TABLE commission_locks (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), customer_charge_id uuid NOT NULL REFERENCES customer_charges(id),
    reseller_id uuid NOT NULL REFERENCES resellers(id), commission_rule_id uuid NOT NULL REFERENCES commission_rules(id), commission_id uuid REFERENCES commissions(id),
    currency char(3) NOT NULL, amount_minor bigint NOT NULL CHECK (amount_minor > 0), status text NOT NULL DEFAULT 'locked' CHECK (status IN ('locked','posted','released','settled','reversed')),
    idempotency_key text NOT NULL, created_by text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id,customer_charge_id,reseller_id), UNIQUE (tenant_id,idempotency_key)
);

CREATE TABLE settlement_cycles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), reseller_id uuid NOT NULL REFERENCES resellers(id),
    name text NOT NULL, period_start date NOT NULL, period_end date NOT NULL, status text NOT NULL DEFAULT 'open' CHECK (status IN ('open','closed','settling','completed','cancelled')),
    created_by text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), CHECK (period_end >= period_start), UNIQUE (tenant_id,reseller_id,period_start,period_end)
);

CREATE TABLE suppliers (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), name text NOT NULL,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('draft','active','suspended','terminated')), created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE (tenant_id,name), UNIQUE (tenant_id,id)
);

CREATE TABLE supplier_members (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), supplier_id uuid NOT NULL REFERENCES suppliers(id),
    user_id text NOT NULL, role text NOT NULL CHECK (role IN ('owner','operator','finance','quality')), status text NOT NULL DEFAULT 'active' CHECK (status IN ('active','suspended','removed')),
    created_at timestamptz NOT NULL DEFAULT now(), UNIQUE (tenant_id,supplier_id,user_id)
);

CREATE TABLE supplier_capabilities (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), supplier_id uuid NOT NULL REFERENCES suppliers(id), capability_id uuid NOT NULL REFERENCES capabilities(id),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active','suspended','retired')), created_at timestamptz NOT NULL DEFAULT now(), UNIQUE (tenant_id,supplier_id,capability_id)
);

ALTER TABLE providers ADD CONSTRAINT providers_tenant_id_unique UNIQUE (tenant_id,id);
ALTER TABLE providers ADD COLUMN supplier_id uuid;
ALTER TABLE providers ADD CONSTRAINT providers_supplier_tenant_fk FOREIGN KEY (tenant_id,supplier_id) REFERENCES suppliers(tenant_id,id);

CREATE TABLE supplier_contracts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), supplier_id uuid NOT NULL REFERENCES suppliers(id), provider_id uuid REFERENCES providers(id),
    name text NOT NULL, status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','pending_approval','approved','active','suspended','expired','terminated')),
    currency char(3) NOT NULL, terms jsonb NOT NULL DEFAULT '{}', version integer NOT NULL DEFAULT 1, starts_at timestamptz, ends_at timestamptz,
    created_by text NOT NULL, approved_by text, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE (tenant_id,supplier_id,name,version),
    FOREIGN KEY (tenant_id,supplier_id) REFERENCES suppliers(tenant_id,id), FOREIGN KEY (tenant_id,provider_id) REFERENCES providers(tenant_id,id)
);

CREATE TABLE supplier_rates (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), contract_id uuid NOT NULL REFERENCES supplier_contracts(id), capability_id uuid NOT NULL REFERENCES capabilities(id),
    unit text NOT NULL, rate_minor bigint NOT NULL CHECK (rate_minor >= 0), version integer NOT NULL DEFAULT 1, status text NOT NULL DEFAULT 'active' CHECK (status IN ('active','retired')),
    created_by text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), UNIQUE (tenant_id,contract_id,capability_id,unit,version)
);

CREATE TABLE supplier_quality_records (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), supplier_id uuid NOT NULL REFERENCES suppliers(id),
    provider_id uuid REFERENCES providers(id), provider_endpoint_id uuid REFERENCES provider_endpoints(id), metric text NOT NULL,
    score_bps integer NOT NULL CHECK (score_bps BETWEEN 0 AND 10000), evidence jsonb NOT NULL DEFAULT '{}', period_start date NOT NULL, period_end date NOT NULL,
    created_by text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), CHECK (period_end >= period_start)
);

CREATE TABLE developers (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), name text NOT NULL,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('draft','active','suspended','terminated')), created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE (tenant_id,name)
);

CREATE TABLE publishers (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), developer_id uuid NOT NULL REFERENCES developers(id), name text NOT NULL,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('draft','active','suspended')), created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE (tenant_id,name)
);

CREATE TABLE listings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), publisher_id uuid NOT NULL REFERENCES publishers(id),
    name text NOT NULL, listing_type text NOT NULL CHECK (listing_type IN ('adapter','capability','workflow','agent','mcp','business_blueprint','pricing_template','growth_playbook')),
    status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','submitted','automated_review','manual_review','sandbox_testing','limited_release','published','suspended','removed')),
    version integer NOT NULL DEFAULT 1, created_by text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE (tenant_id,publisher_id,name)
);

CREATE TABLE listing_versions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), listing_id uuid NOT NULL REFERENCES listings(id), version integer NOT NULL,
    capability_manifest jsonb NOT NULL DEFAULT '{}', permission_manifest jsonb NOT NULL DEFAULT '{}', content_ref text NOT NULL DEFAULT '', checksum text NOT NULL,
    created_by text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), UNIQUE (tenant_id,listing_id,version), UNIQUE (tenant_id,listing_id,checksum)
);

CREATE TABLE listing_reviews (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), listing_id uuid NOT NULL REFERENCES listings(id), listing_version_id uuid NOT NULL REFERENCES listing_versions(id),
    review_type text NOT NULL CHECK (review_type IN ('automated','security','license','manual')), decision text NOT NULL CHECK (decision IN ('approved','rejected','changes_requested')),
    rationale text NOT NULL, reviewed_by text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), UNIQUE (tenant_id,listing_version_id,review_type)
);

CREATE TABLE sandbox_runs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), listing_version_id uuid NOT NULL REFERENCES listing_versions(id),
    status text NOT NULL CHECK (status IN ('queued','running','succeeded','failed','cancelled')), policy jsonb NOT NULL DEFAULT '{}', result jsonb NOT NULL DEFAULT '{}',
    created_by text NOT NULL, started_at timestamptz NOT NULL DEFAULT now(), completed_at timestamptz
);

CREATE TABLE quality_scores (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), listing_version_id uuid NOT NULL REFERENCES listing_versions(id),
    score_bps integer NOT NULL CHECK (score_bps BETWEEN 0 AND 10000), dimensions jsonb NOT NULL DEFAULT '{}', created_by text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), UNIQUE (tenant_id,listing_version_id)
);

CREATE TABLE incident_records (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), listing_id uuid NOT NULL REFERENCES listings(id),
    severity text NOT NULL CHECK (severity IN ('low','medium','high','critical')), summary text NOT NULL, status text NOT NULL DEFAULT 'open' CHECK (status IN ('open','mitigated','resolved')),
    created_by text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), resolved_at timestamptz
);

CREATE TABLE revenue_share_rules (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), listing_id uuid NOT NULL REFERENCES listings(id), publisher_id uuid NOT NULL REFERENCES publishers(id),
    basis_points integer NOT NULL CHECK (basis_points BETWEEN 0 AND 10000), currency char(3) NOT NULL, status text NOT NULL DEFAULT 'active' CHECK (status IN ('draft','active','retired')),
    version integer NOT NULL DEFAULT 1, created_by text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), UNIQUE (tenant_id,listing_id,version)
);

CREATE TABLE payout_reserves (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), listing_id uuid NOT NULL REFERENCES listings(id), publisher_id uuid NOT NULL REFERENCES publishers(id),
    currency char(3) NOT NULL, amount_minor bigint NOT NULL CHECK (amount_minor > 0), status text NOT NULL DEFAULT 'reserved' CHECK (status IN ('reserved','released','paid','reversed')),
    reference_type text NOT NULL, reference_id text NOT NULL, created_by text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE marketplace_disputes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), listing_id uuid NOT NULL REFERENCES listings(id),
    claimant_type text NOT NULL CHECK (claimant_type IN ('developer','publisher','customer','supplier','reseller','platform')), claimant_id text NOT NULL,
    reason text NOT NULL, status text NOT NULL DEFAULT 'open' CHECK (status IN ('open','in_review','resolved','rejected')), resolution jsonb NOT NULL DEFAULT '{}',
    created_by text NOT NULL, resolved_by text, created_at timestamptz NOT NULL DEFAULT now(), resolved_at timestamptz
);

CREATE TABLE takedowns (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), tenant_id uuid NOT NULL REFERENCES tenants(id), listing_id uuid NOT NULL REFERENCES listings(id),
    reason text NOT NULL, status text NOT NULL DEFAULT 'requested' CHECK (status IN ('requested','approved','executed','rejected')), requested_by text NOT NULL,
    reviewed_by text, created_at timestamptz NOT NULL DEFAULT now(), reviewed_at timestamptz
);

CREATE INDEX idx_lead_ownership_reseller ON lead_ownerships (tenant_id,reseller_id,status,protection_expires_at);
CREATE INDEX idx_customer_ownership_reseller ON customer_ownerships (tenant_id,reseller_id,status,protection_expires_at);
CREATE INDEX idx_supplier_contract_status ON supplier_contracts (tenant_id,supplier_id,status,updated_at DESC);
CREATE INDEX idx_listing_status ON listings (tenant_id,status,updated_at DESC);
CREATE INDEX idx_listing_review ON listing_reviews (tenant_id,listing_version_id,review_type);

DO $$
DECLARE table_name text;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'reseller_levels','resellers','reseller_members','attribution_rules','lead_ownerships','customer_ownerships','transfer_requests','conflict_records',
        'commission_rules','commission_locks','settlement_cycles','suppliers','supplier_members','supplier_capabilities','supplier_contracts','supplier_rates','supplier_quality_records',
        'developers','publishers','listings','listing_versions','listing_reviews','sandbox_runs','quality_scores','incident_records','revenue_share_rules','payout_reserves','marketplace_disputes','takedowns'
    ] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', table_name);
        EXECUTE format('CREATE POLICY tenant_isolation ON %I USING (tenant_id = NULLIF(current_setting(''app.tenant_id'', true), '''')::uuid) WITH CHECK (tenant_id = NULLIF(current_setting(''app.tenant_id'', true), '''')::uuid)', table_name);
    END LOOP;
END $$;

INSERT INTO feature_definitions (key,description,default_enabled) VALUES
    ('channel.reseller','Reseller ownership, attribution, commission lock, and settlement-cycle control plane',true),
    ('channel.supplier','Supplier contracts, rates, quality, and canonical payable settlement views',true),
    ('marketplace.internal','Internal developer listing, review, sandbox, dispute, and takedown workflow',true),
    ('marketplace.payout','Marketplace payout reserve and external payout delivery',false)
ON CONFLICT (key) DO UPDATE SET description=EXCLUDED.description,default_enabled=EXCLUDED.default_enabled;

GRANT SELECT,INSERT,UPDATE,DELETE ON ALL TABLES IN SCHEMA public TO opportunity_app;
GRANT USAGE,SELECT ON ALL SEQUENCES IN SCHEMA public TO opportunity_app;

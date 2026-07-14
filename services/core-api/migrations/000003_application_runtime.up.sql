CREATE TABLE command_idempotency (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    idempotency_key text NOT NULL,
    operation text NOT NULL,
    aggregate_id uuid,
    response jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, idempotency_key)
);

CREATE TABLE opportunity_reviews (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    opportunity_id uuid NOT NULL REFERENCES opportunities(id),
    actor_id text NOT NULL,
    decision text NOT NULL CHECK (decision IN ('approved', 'rejected')),
    rationale text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE incubation_projects
    ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();

CREATE INDEX idx_evidence_opportunity
    ON opportunity_evidence (tenant_id, opportunity_id, created_at);

CREATE INDEX idx_reviews_opportunity
    ON opportunity_reviews (tenant_id, opportunity_id, created_at DESC);

CREATE INDEX idx_incubation_opportunity
    ON incubation_projects (tenant_id, opportunity_id, created_at DESC);

CREATE INDEX idx_blueprints_opportunity
    ON business_blueprints (tenant_id, source_opportunity_id, updated_at DESC);

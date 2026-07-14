ALTER TABLE capabilities
    ADD COLUMN description text NOT NULL DEFAULT '';

ALTER TABLE products
    ADD COLUMN created_by text NOT NULL DEFAULT 'system';

ALTER TABLE product_versions
    ADD COLUMN created_by text NOT NULL DEFAULT 'system';

ALTER TABLE skus
    ADD COLUMN name text NOT NULL DEFAULT '',
    ADD COLUMN status text NOT NULL DEFAULT 'draft',
    ADD COLUMN created_by text NOT NULL DEFAULT 'system',
    ADD COLUMN created_at timestamptz NOT NULL DEFAULT now();

ALTER TABLE sku_versions
    ADD COLUMN created_by text NOT NULL DEFAULT 'system',
    ADD COLUMN created_at timestamptz NOT NULL DEFAULT now();

CREATE TABLE product_capability_bindings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    product_version_id uuid NOT NULL REFERENCES product_versions(id),
    capability_id uuid NOT NULL REFERENCES capabilities(id),
    required boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, product_version_id, capability_id)
);

CREATE TABLE product_workflow_bindings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    product_version_id uuid NOT NULL REFERENCES product_versions(id),
    workflow_definition_id uuid NOT NULL REFERENCES workflow_definitions(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, product_version_id)
);

CREATE TABLE product_metering_bindings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    product_version_id uuid NOT NULL REFERENCES product_versions(id),
    metering_definition_id uuid NOT NULL REFERENCES metering_definitions(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, product_version_id)
);

CREATE TABLE product_pricing_bindings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    product_version_id uuid NOT NULL REFERENCES product_versions(id),
    price_book_id uuid NOT NULL REFERENCES price_books(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, product_version_id)
);

CREATE TABLE product_routing_bindings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    product_version_id uuid NOT NULL REFERENCES product_versions(id),
    route_policy_id uuid NOT NULL REFERENCES route_policies(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, product_version_id)
);

CREATE TABLE product_form_definitions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    product_version_id uuid NOT NULL REFERENCES product_versions(id),
    form_schema jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, product_version_id)
);

CREATE TABLE product_output_definitions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    product_version_id uuid NOT NULL REFERENCES product_versions(id),
    output_schema jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, product_version_id)
);

CREATE TABLE product_growth_bindings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    product_version_id uuid NOT NULL REFERENCES product_versions(id),
    growth_playbook jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, product_version_id)
);

CREATE TABLE product_compliance_bindings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    product_version_id uuid NOT NULL REFERENCES product_versions(id),
    compliance_profile jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, product_version_id)
);

CREATE TABLE publications (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    product_id uuid NOT NULL REFERENCES products(id),
    product_version_id uuid NOT NULL REFERENCES product_versions(id),
    status text NOT NULL CHECK (status IN ('published', 'suspended', 'removed')),
    published_by text NOT NULL,
    published_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, product_version_id)
);

CREATE INDEX idx_products_tenant_updated ON products (tenant_id, updated_at DESC);
CREATE INDEX idx_product_versions_product ON product_versions (tenant_id, product_id, version DESC);
CREATE INDEX idx_skus_product ON skus (tenant_id, product_id);

DO $$
DECLARE
    table_name text;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'product_capability_bindings', 'product_workflow_bindings',
        'product_metering_bindings', 'product_pricing_bindings',
        'product_routing_bindings', 'product_form_definitions',
        'product_output_definitions', 'product_growth_bindings',
        'product_compliance_bindings', 'publications'
    ] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', table_name);
        EXECUTE format(
            'CREATE POLICY tenant_isolation ON %I USING (tenant_id = NULLIF(current_setting(''app.tenant_id'', true), '''')::uuid) WITH CHECK (tenant_id = NULLIF(current_setting(''app.tenant_id'', true), '''')::uuid)',
            table_name
        );
    END LOOP;
END $$;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO opportunity_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO opportunity_app;

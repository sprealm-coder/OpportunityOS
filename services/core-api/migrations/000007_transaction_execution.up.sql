CREATE TABLE quotes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    deal_id text NOT NULL,
    customer_id text NOT NULL,
    status text NOT NULL CHECK (status IN ('draft', 'sent', 'accepted', 'rejected', 'expired', 'cancelled')),
    version integer NOT NULL DEFAULT 1 CHECK (version > 0),
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE quote_versions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    quote_id uuid NOT NULL REFERENCES quotes(id),
    version integer NOT NULL CHECK (version > 0),
    currency char(3) NOT NULL,
    amount_minor bigint NOT NULL CHECK (amount_minor >= 0),
    valid_until timestamptz NOT NULL,
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, quote_id, version)
);

CREATE TABLE quote_version_items (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    quote_version_id uuid NOT NULL REFERENCES quote_versions(id),
    product_version_id uuid NOT NULL REFERENCES product_versions(id),
    sku_version_id uuid NOT NULL REFERENCES sku_versions(id),
    workflow_version_id uuid NOT NULL REFERENCES workflow_definitions(id),
    pricing_version_id uuid NOT NULL REFERENCES price_books(id),
    routing_version_id uuid NOT NULL REFERENCES route_policies(id),
    quantity bigint NOT NULL CHECK (quantity > 0),
    unit_amount_minor bigint NOT NULL CHECK (unit_amount_minor >= 0),
    amount_minor bigint NOT NULL CHECK (amount_minor >= 0),
    input jsonb NOT NULL DEFAULT '{}'
);

ALTER TABLE orders
    ADD COLUMN quote_version_id uuid REFERENCES quote_versions(id),
    ADD COLUMN version integer NOT NULL DEFAULT 1 CHECK (version > 0),
    ADD COLUMN created_by text NOT NULL DEFAULT 'system',
    ADD CONSTRAINT orders_status_check CHECK (status IN ('created', 'awaiting_payment', 'paid', 'provisioning', 'active', 'completed', 'cancelled', 'refund_pending', 'refunded', 'disputed'));

CREATE TABLE order_items (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    order_id uuid NOT NULL REFERENCES orders(id),
    quote_version_item_id uuid NOT NULL REFERENCES quote_version_items(id),
    product_version_id uuid NOT NULL REFERENCES product_versions(id),
    sku_version_id uuid NOT NULL REFERENCES sku_versions(id),
    workflow_version_id uuid NOT NULL REFERENCES workflow_definitions(id),
    pricing_version_id uuid NOT NULL REFERENCES price_books(id),
    routing_version_id uuid NOT NULL REFERENCES route_policies(id),
    quantity bigint NOT NULL CHECK (quantity > 0),
    unit_amount_minor bigint NOT NULL CHECK (unit_amount_minor >= 0),
    amount_minor bigint NOT NULL CHECK (amount_minor >= 0),
    input jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, order_id, quote_version_item_id)
);

CREATE TABLE subscriptions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    order_id uuid NOT NULL REFERENCES orders(id),
    order_item_id uuid NOT NULL REFERENCES order_items(id),
    customer_id text NOT NULL,
    sku_version_id uuid NOT NULL REFERENCES sku_versions(id),
    status text NOT NULL CHECK (status IN ('pending', 'active', 'paused', 'cancelled', 'expired')),
    starts_at timestamptz,
    ends_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, order_item_id)
);

CREATE TABLE entitlements (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    order_id uuid NOT NULL REFERENCES orders(id),
    order_item_id uuid NOT NULL REFERENCES order_items(id),
    subscription_id uuid REFERENCES subscriptions(id),
    entitlement_key text NOT NULL,
    entitlement_value jsonb NOT NULL,
    status text NOT NULL CHECK (status IN ('pending', 'active', 'suspended', 'revoked', 'expired')),
    starts_at timestamptz,
    ends_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, order_item_id, entitlement_key)
);

CREATE TABLE execution_orders (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    order_id uuid NOT NULL REFERENCES orders(id),
    order_item_id uuid NOT NULL REFERENCES order_items(id),
    product_version_id uuid NOT NULL REFERENCES product_versions(id),
    sku_version_id uuid NOT NULL REFERENCES sku_versions(id),
    workflow_version_id uuid NOT NULL REFERENCES workflow_definitions(id),
    pricing_version_id uuid NOT NULL REFERENCES price_books(id),
    routing_version_id uuid NOT NULL REFERENCES route_policies(id),
    provider_endpoint_id uuid REFERENCES provider_endpoints(id),
    status text NOT NULL CHECK (status IN ('created', 'validating', 'reserved', 'queued', 'submitted', 'processing', 'succeeded', 'failed', 'cancelled', 'reconciling', 'settled')),
    attempt integer NOT NULL DEFAULT 0 CHECK (attempt >= 0),
    idempotency_key text NOT NULL,
    external_id text,
    input jsonb NOT NULL DEFAULT '{}',
    output jsonb NOT NULL DEFAULT '{}',
    error jsonb NOT NULL DEFAULT '{}',
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, idempotency_key),
    UNIQUE (tenant_id, order_item_id)
);

CREATE TABLE delivery_projects (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    order_id uuid NOT NULL REFERENCES orders(id),
    order_item_id uuid NOT NULL REFERENCES order_items(id),
    execution_order_id uuid NOT NULL REFERENCES execution_orders(id),
    mode text NOT NULL CHECK (mode IN ('workflow', 'realtime', 'async', 'provisioning', 'manual')),
    status text NOT NULL CHECK (status IN ('created', 'in_progress', 'waiting', 'completed', 'failed', 'cancelled')),
    assignee text,
    artifacts jsonb NOT NULL DEFAULT '[]',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, execution_order_id)
);

ALTER TABLE usage_records
    ADD COLUMN order_item_id uuid REFERENCES order_items(id),
    ADD COLUMN execution_order_id uuid REFERENCES execution_orders(id),
    ADD COLUMN idempotency_key text,
    ADD COLUMN created_by text NOT NULL DEFAULT 'system',
    ADD COLUMN created_at timestamptz NOT NULL DEFAULT now(),
    ADD CONSTRAINT usage_records_idempotency_unique UNIQUE (tenant_id, idempotency_key);

ALTER TABLE provider_costs
    ADD COLUMN order_item_id uuid REFERENCES order_items(id),
    ADD COLUMN execution_order_id uuid REFERENCES execution_orders(id),
    ADD COLUMN idempotency_key text,
    ADD COLUMN created_by text NOT NULL DEFAULT 'system',
    ADD COLUMN created_at timestamptz NOT NULL DEFAULT now(),
    ADD CONSTRAINT provider_costs_idempotency_unique UNIQUE (tenant_id, idempotency_key);

ALTER TABLE customer_charges
    ADD COLUMN order_item_id uuid REFERENCES order_items(id),
    ADD COLUMN execution_order_id uuid REFERENCES execution_orders(id),
    ADD COLUMN idempotency_key text,
    ADD COLUMN status text NOT NULL DEFAULT 'calculated' CHECK (status IN ('calculated', 'pending', 'posted', 'reversed')),
    ADD COLUMN created_by text NOT NULL DEFAULT 'system',
    ADD COLUMN created_at timestamptz NOT NULL DEFAULT now(),
    ADD CONSTRAINT customer_charges_idempotency_unique UNIQUE (tenant_id, idempotency_key),
    ADD CONSTRAINT customer_charges_execution_unique UNIQUE (tenant_id, execution_order_id);

CREATE INDEX idx_quotes_tenant_updated ON quotes (tenant_id, updated_at DESC);
CREATE INDEX idx_orders_tenant_updated ON orders (tenant_id, updated_at DESC);
CREATE INDEX idx_execution_orders_status ON execution_orders (tenant_id, status, updated_at);
CREATE INDEX idx_delivery_projects_status ON delivery_projects (tenant_id, status, updated_at);

DO $$
DECLARE
    table_name text;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'quotes', 'quote_versions', 'quote_version_items', 'order_items',
        'subscriptions', 'entitlements', 'execution_orders', 'delivery_projects'
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

CREATE TABLE wallets (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    owner_type text NOT NULL CHECK (owner_type IN ('customer', 'provider', 'reseller', 'platform')),
    owner_id text NOT NULL,
    currency char(3) NOT NULL,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'frozen', 'closed')),
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, owner_type, owner_id, currency)
);

ALTER TABLE ledger_accounts
    ADD COLUMN wallet_id uuid REFERENCES wallets(id),
    ADD COLUMN purpose text NOT NULL DEFAULT 'general',
    ADD CONSTRAINT ledger_accounts_type_check CHECK (account_type IN ('asset', 'liability', 'revenue', 'expense', 'equity'));

ALTER TABLE ledger_transactions
    ADD COLUMN transaction_type text NOT NULL DEFAULT 'adjustment',
    ADD COLUMN created_by text NOT NULL DEFAULT 'system',
    ADD COLUMN metadata jsonb NOT NULL DEFAULT '{}';

ALTER TABLE customer_charges
    ADD COLUMN ledger_transaction_id uuid REFERENCES ledger_transactions(id),
    ADD COLUMN posted_at timestamptz;

CREATE TABLE balance_snapshots (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    account_id uuid NOT NULL REFERENCES ledger_accounts(id),
    transaction_id uuid NOT NULL REFERENCES ledger_transactions(id),
    balance_minor bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, account_id, transaction_id)
);

CREATE TABLE holds (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    wallet_id uuid NOT NULL REFERENCES wallets(id),
    order_id uuid NOT NULL REFERENCES orders(id),
    currency char(3) NOT NULL,
    amount_minor bigint NOT NULL CHECK (amount_minor > 0),
    captured_minor bigint NOT NULL DEFAULT 0 CHECK (captured_minor >= 0),
    released_minor bigint NOT NULL DEFAULT 0 CHECK (released_minor >= 0),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'partially_captured', 'captured', 'released')),
    ledger_transaction_id uuid NOT NULL REFERENCES ledger_transactions(id),
    idempotency_key text NOT NULL,
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (captured_minor + released_minor <= amount_minor),
    UNIQUE (tenant_id, idempotency_key)
);

CREATE TABLE hold_releases (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    hold_id uuid NOT NULL REFERENCES holds(id),
    amount_minor bigint NOT NULL CHECK (amount_minor > 0),
    ledger_transaction_id uuid NOT NULL REFERENCES ledger_transactions(id),
    idempotency_key text NOT NULL,
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, idempotency_key)
);

CREATE TABLE refunds (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    customer_charge_id uuid NOT NULL REFERENCES customer_charges(id),
    wallet_id uuid NOT NULL REFERENCES wallets(id),
    currency char(3) NOT NULL,
    amount_minor bigint NOT NULL CHECK (amount_minor > 0),
    reason text NOT NULL,
    status text NOT NULL DEFAULT 'posted' CHECK (status IN ('posted', 'reversed')),
    ledger_transaction_id uuid NOT NULL REFERENCES ledger_transactions(id),
    idempotency_key text NOT NULL,
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, idempotency_key)
);

CREATE TABLE financial_adjustments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    wallet_id uuid NOT NULL REFERENCES wallets(id),
    direction text NOT NULL CHECK (direction IN ('credit', 'debit')),
    currency char(3) NOT NULL,
    amount_minor bigint NOT NULL CHECK (amount_minor > 0),
    reason text NOT NULL,
    ledger_transaction_id uuid NOT NULL REFERENCES ledger_transactions(id),
    idempotency_key text NOT NULL,
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, idempotency_key)
);

CREATE TABLE commissions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    customer_charge_id uuid NOT NULL REFERENCES customer_charges(id),
    beneficiary_type text NOT NULL CHECK (beneficiary_type IN ('reseller', 'agent', 'partner')),
    beneficiary_id text NOT NULL,
    currency char(3) NOT NULL,
    amount_minor bigint NOT NULL CHECK (amount_minor > 0),
    settled_minor bigint NOT NULL DEFAULT 0 CHECK (settled_minor >= 0),
    status text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'partially_settled', 'settled', 'reversed')),
    payable_account_id uuid NOT NULL REFERENCES ledger_accounts(id),
    ledger_transaction_id uuid NOT NULL REFERENCES ledger_transactions(id),
    idempotency_key text NOT NULL,
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (settled_minor <= amount_minor),
    UNIQUE (tenant_id, idempotency_key)
);

CREATE TABLE provider_payables (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    provider_cost_id uuid NOT NULL REFERENCES provider_costs(id),
    provider_id uuid NOT NULL REFERENCES providers(id),
    currency char(3) NOT NULL,
    amount_minor bigint NOT NULL CHECK (amount_minor > 0),
    settled_minor bigint NOT NULL DEFAULT 0 CHECK (settled_minor >= 0),
    status text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'partially_settled', 'settled', 'reversed')),
    payable_account_id uuid NOT NULL REFERENCES ledger_accounts(id),
    ledger_transaction_id uuid NOT NULL REFERENCES ledger_transactions(id),
    idempotency_key text NOT NULL,
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (settled_minor <= amount_minor),
    UNIQUE (tenant_id, provider_cost_id),
    UNIQUE (tenant_id, idempotency_key)
);

CREATE TABLE settlements (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    source_type text NOT NULL CHECK (source_type IN ('provider_payable', 'commission')),
    source_id uuid NOT NULL,
    beneficiary_type text NOT NULL,
    beneficiary_id text NOT NULL,
    currency char(3) NOT NULL,
    amount_minor bigint NOT NULL CHECK (amount_minor > 0),
    status text NOT NULL DEFAULT 'completed' CHECK (status IN ('completed', 'reversed')),
    ledger_transaction_id uuid NOT NULL REFERENCES ledger_transactions(id),
    idempotency_key text NOT NULL,
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, idempotency_key)
);

CREATE TABLE payment_receivables (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    order_id uuid NOT NULL REFERENCES orders(id),
    customer_id text NOT NULL,
    currency char(3) NOT NULL,
    amount_minor bigint NOT NULL CHECK (amount_minor > 0),
    received_minor bigint NOT NULL DEFAULT 0 CHECK (received_minor >= 0),
    status text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'partially_received', 'received', 'written_off')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (received_minor <= amount_minor),
    UNIQUE (tenant_id, order_id)
);

CREATE TABLE promotional_credits (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    wallet_id uuid NOT NULL REFERENCES wallets(id),
    currency char(3) NOT NULL,
    amount_minor bigint NOT NULL CHECK (amount_minor > 0),
    consumed_minor bigint NOT NULL DEFAULT 0 CHECK (consumed_minor >= 0),
    expires_at timestamptz,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'consumed', 'expired', 'revoked')),
    ledger_transaction_id uuid NOT NULL REFERENCES ledger_transactions(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (consumed_minor <= amount_minor)
);

CREATE TABLE chargeback_reserves (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    wallet_id uuid NOT NULL REFERENCES wallets(id),
    currency char(3) NOT NULL,
    amount_minor bigint NOT NULL CHECK (amount_minor > 0),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'released', 'applied')),
    ledger_transaction_id uuid NOT NULL REFERENCES ledger_transactions(id),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE settlement_reserves (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    source_type text NOT NULL,
    source_id uuid NOT NULL,
    currency char(3) NOT NULL,
    amount_minor bigint NOT NULL CHECK (amount_minor > 0),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'released', 'applied')),
    ledger_transaction_id uuid NOT NULL REFERENCES ledger_transactions(id),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE reconciliation_runs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    order_id uuid REFERENCES orders(id),
    status text NOT NULL CHECK (status IN ('matched', 'discrepancy')),
    checked_count integer NOT NULL CHECK (checked_count >= 0),
    discrepancy_count integer NOT NULL CHECK (discrepancy_count >= 0),
    started_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz NOT NULL DEFAULT now(),
    created_by text NOT NULL,
    idempotency_key text NOT NULL,
    UNIQUE (tenant_id, idempotency_key)
);

CREATE TABLE reconciliation_items (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    run_id uuid NOT NULL REFERENCES reconciliation_runs(id),
    reference_type text NOT NULL,
    reference_id uuid NOT NULL,
    currency char(3) NOT NULL,
    expected_minor bigint NOT NULL CHECK (expected_minor >= 0),
    actual_minor bigint NOT NULL CHECK (actual_minor >= 0),
    status text NOT NULL CHECK (status IN ('matched', 'discrepancy')),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE reconciliation_discrepancies (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    run_id uuid NOT NULL REFERENCES reconciliation_runs(id),
    item_id uuid NOT NULL REFERENCES reconciliation_items(id),
    discrepancy_type text NOT NULL,
    difference_minor bigint NOT NULL,
    status text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'resolved', 'accepted')),
    resolution_adjustment_id uuid REFERENCES financial_adjustments(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    resolved_at timestamptz
);

CREATE INDEX idx_ledger_entries_account_created ON ledger_entries (tenant_id, account_id, created_at, id);
CREATE INDEX idx_ledger_transactions_reference ON ledger_transactions (tenant_id, reference_type, reference_id);
CREATE INDEX idx_holds_order_status ON holds (tenant_id, order_id, status);
CREATE INDEX idx_provider_payables_status ON provider_payables (tenant_id, status, created_at);
CREATE INDEX idx_commissions_status ON commissions (tenant_id, status, created_at);
CREATE INDEX idx_settlements_source ON settlements (tenant_id, source_type, source_id);
CREATE INDEX idx_reconciliation_runs_created ON reconciliation_runs (tenant_id, completed_at DESC);

CREATE OR REPLACE FUNCTION assert_ledger_transaction_balanced() RETURNS trigger AS $$
DECLARE
    entry_count integer;
    debit_total bigint;
    credit_total bigint;
    currency_count integer;
BEGIN
    SELECT count(*),
           COALESCE(sum(amount_minor) FILTER (WHERE direction='debit'), 0),
           COALESCE(sum(amount_minor) FILTER (WHERE direction='credit'), 0),
           count(DISTINCT currency)
      INTO entry_count, debit_total, credit_total, currency_count
      FROM ledger_entries
     WHERE tenant_id=NEW.tenant_id AND transaction_id=NEW.id;
    IF entry_count < 2 OR debit_total <> credit_total OR currency_count <> 1 THEN
        RAISE EXCEPTION 'ledger transaction % is not balanced', NEW.id USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER ledger_transaction_balance_check
    AFTER INSERT ON ledger_transactions
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION assert_ledger_transaction_balanced();

CREATE OR REPLACE FUNCTION reject_ledger_mutation() RETURNS trigger AS $$
BEGIN
    IF current_user = 'opportunity_app' THEN
        RAISE EXCEPTION 'ledger history is append-only' USING ERRCODE='55000';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER ledger_transactions_immutable
    BEFORE UPDATE OR DELETE ON ledger_transactions
    FOR EACH ROW EXECUTE FUNCTION reject_ledger_mutation();
CREATE TRIGGER ledger_entries_immutable
    BEFORE UPDATE OR DELETE ON ledger_entries
    FOR EACH ROW EXECUTE FUNCTION reject_ledger_mutation();
CREATE TRIGGER balance_snapshots_immutable
    BEFORE UPDATE OR DELETE ON balance_snapshots
    FOR EACH ROW EXECUTE FUNCTION reject_ledger_mutation();

DO $$
DECLARE
    table_name text;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'wallets', 'balance_snapshots', 'holds', 'hold_releases', 'refunds',
        'financial_adjustments', 'commissions', 'provider_payables', 'settlements',
        'payment_receivables', 'promotional_credits', 'chargeback_reserves',
        'settlement_reserves', 'reconciliation_runs', 'reconciliation_items',
        'reconciliation_discrepancies'
    ] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', table_name);
        EXECUTE format(
            'CREATE POLICY tenant_isolation ON %I USING (tenant_id = NULLIF(current_setting(''app.tenant_id'', true), '''')::uuid) WITH CHECK (tenant_id = NULLIF(current_setting(''app.tenant_id'', true), '''')::uuid)',
            table_name
        );
    END LOOP;
END $$;

INSERT INTO feature_definitions (key, description, default_enabled)
VALUES ('finance.ledger', 'Wallet, append-only ledger, settlement, and reconciliation', true)
ON CONFLICT (key) DO UPDATE SET description=EXCLUDED.description, default_enabled=EXCLUDED.default_enabled;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO opportunity_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO opportunity_app;

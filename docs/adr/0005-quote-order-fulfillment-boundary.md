# ADR 0005: Quote acceptance freezes the commercial version boundary

## Status

Accepted.

## Decision

Quote pricing is calculated by the core API from the active PriceBook bound to a published SKUVersion. Clients submit SKUVersion, quantity, and schema-driven input; they do not submit authoritative amounts. A QuoteVersion and its line items are immutable.

An Order can only be created from the latest accepted and unexpired QuoteVersion. Each OrderItem copies the ProductVersion, SKUVersion, Workflow, Pricing, and Routing identifiers. Later product or provider changes cannot change an existing order promise.

The `paid -> provisioning` command creates Subscription, Entitlement, ExecutionOrder, and DeliveryProject records in the same PostgreSQL transaction as the order transition, audit records, and Outbox events. Activation requires successful execution and completed delivery. Completion additionally requires settled execution.

Usage, ProviderCost, and CustomerCharge are separate operational facts. They are not LedgerEntry records. Phase E is responsible for idempotent balanced postings, holds, releases, refunds, commissions, payables, settlement, and reconciliation.

## Consequences

- Cross-tenant or unpublished SKU references cannot enter a quote.
- Stale or expired quote versions cannot create orders.
- Fulfillment records cannot be partially created after an order enters provisioning.
- Provider selection can change during execution without mutating the customer-facing order version.
- A calculated customer charge does not affect wallet or ledger balances until a later accounting command posts it.

# Roadmap

## Current cycle

- [x] Establish monorepo, Compose infrastructure, architecture records, and neutral domain contracts.
- [x] Implement the first executable core vertical slice and tests.
- [x] Apply real PostgreSQL migrations and neutral seed data locally.
- [x] Build all seven portal entry points; browser-check admin and operator desktop/mobile views.
- [x] Replace in-memory API wiring with PostgreSQL repositories and transaction-scoped audit/outbox writes.
- [x] Connect admin-web and operator-console to tenant-scoped API queries and command forms.
- [x] Replace development identity headers with PostgreSQL-backed Cookie sessions and permission-derived commands.
- [x] Enforce tenant isolation with PostgreSQL RLS through a non-owner runtime role.
- [x] Publish persisted outbox events through leased Redis Streams delivery with retries and dead-letter handling.
- [x] Persist inbound event deduplication in the tenant-scoped PostgreSQL Inbox.
- [x] Persist Capability, Provider, ProviderEndpoint, Product, ProductVersion, SKU, SKUVersion, and Publication.
- [x] Bind product versions to JSON Schema, controlled workflow, metering, integer pricing, routing, growth, delivery, and compliance definitions.
- [x] Enforce publication readiness with healthy-provider and SKU-version gates.
- [x] Connect the operator product factory to authenticated product configuration and publication commands.
- [x] Persist Quote, immutable QuoteVersion, and server-priced quote line bindings.
- [x] Persist Order and OrderItem snapshots bound to published ProductVersion, SKUVersion, Workflow, Pricing, and Routing records.
- [x] Create Subscription, Entitlement, ExecutionOrder, and DeliveryProject records atomically at the provisioning boundary.
- [x] Persist controlled execution/delivery transitions and separate Usage, ProviderCost, and CustomerCharge facts.
- [x] Connect the operator transaction workspace to authenticated quote-to-fulfillment commands.

## First-release scope retained

The remaining parts of phases C-H remain in scope: supplier contracts and quality, finance persistence, growth, reseller/supplier/marketplace workflows, and production end-to-end hardening.

## Next development task

Implement phase E persistence on the authenticated/RLS-aware repository boundary: Wallet, LedgerAccount, append-only LedgerTransaction/LedgerEntry, Hold, Release, Charge posting, Refund, Commission, ProviderPayable, Settlement, and Reconciliation. Connect calculated CustomerCharge and ProviderCost facts to balanced, idempotent ledger commands without merging operational facts into accounting entries.

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

## First-release scope retained

The remaining parts of phases C-H remain in scope: supplier contracts and quality, transaction/execution persistence, finance persistence, growth, reseller/supplier/marketplace workflows, and production end-to-end hardening.

## Next development task

Implement phase D persistence on the authenticated/RLS-aware repository boundary: Quote/QuoteVersion, Order/OrderItem, Subscription, Entitlement, ExecutionOrder and DeliveryProject. Bind every order to immutable ProductVersion, SKUVersion, Workflow, Pricing and Routing versions, with transactional audit/Outbox and repository integration tests.

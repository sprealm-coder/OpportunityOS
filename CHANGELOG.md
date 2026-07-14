# Changelog

## Unreleased

### Added

- PostgreSQL-backed Cookie sessions, role-derived permissions, tenant RLS, and authenticated admin/operator portals.
- Leased Outbox delivery with retry/dead-letter handling, Redis Streams adapter checks, and persisted Inbox deduplication.
- Capability, Provider, and ProviderEndpoint runtime APIs.
- Product, immutable ProductVersion, SKU, SKUVersion, binding, and Publication migrations and repositories.
- Publication readiness checks for schema, workflow, provider availability, SKU bindings, metering, pricing, routing, delivery, and compliance.
- Operator product factory for resource setup, blueprint approval, version configuration, SKU binding, and publication.
- Quote, immutable QuoteVersion and line-item persistence with server-side integer pricing from active PriceBooks.
- Order and OrderItem version snapshots that reject unpublished, stale, expired, or cross-tenant references.
- Atomic provisioning of Subscription, Entitlement, ExecutionOrder, and DeliveryProject records.
- Controlled execution and delivery transitions plus separate Usage, ProviderCost, and CustomerCharge records.
- Authenticated transaction/execution APIs and an operator workflow for quote-to-fulfillment commands.
- Tenant-scoped Wallet and available/held liability accounts with integer balances rebuilt from append-only entries.
- PostgreSQL-backed Hold, Release, CustomerCharge posting, Refund, Commission, ProviderPayable, Settlement, and Reconciliation commands.
- Deferred debit/credit balance enforcement, immutable runtime ledger history, account locking, and append-only balance snapshots.
- Finance permissions, OpenAPI `0.5.0`, shared TypeScript contracts, and operator/admin finance views.

### Changed

- Browser clients now derive tenant and actor from trusted sessions instead of request headers.
- Growth Quote now aliases the canonical transaction-domain Quote instead of defining a duplicate model.
- Orders now require sufficient active held funds before payment, and executions require posted charges and provider payables before settlement.
- The next development boundary advances to phase F product-agnostic growth persistence: MarketSegment, ICP, Lead, Evidence, Proof, Campaign, Conversation, Deal, and Experiment.

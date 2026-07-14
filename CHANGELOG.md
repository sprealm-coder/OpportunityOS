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

### Changed

- Browser clients now derive tenant and actor from trusted sessions instead of request headers.
- Growth Quote now aliases the canonical transaction-domain Quote instead of defining a duplicate model.
- The next development boundary advances to phase E wallet, hold, ledger posting, refund, commission, settlement, and reconciliation persistence.

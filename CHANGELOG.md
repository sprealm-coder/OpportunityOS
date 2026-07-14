# Changelog

## Unreleased

### Added

- PostgreSQL-backed Cookie sessions, role-derived permissions, tenant RLS, and authenticated admin/operator portals.
- Leased Outbox delivery with retry/dead-letter handling, Redis Streams adapter checks, and persisted Inbox deduplication.
- Capability, Provider, and ProviderEndpoint runtime APIs.
- Product, immutable ProductVersion, SKU, SKUVersion, binding, and Publication migrations and repositories.
- Publication readiness checks for schema, workflow, provider availability, SKU bindings, metering, pricing, routing, delivery, and compliance.
- Operator product factory for resource setup, blueprint approval, version configuration, SKU binding, and publication.

### Changed

- Browser clients now derive tenant and actor from trusted sessions instead of request headers.
- The next development boundary advances to persistent quote, order, subscription, entitlement, execution, and delivery workflows.

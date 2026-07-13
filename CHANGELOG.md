# Changelog

## Unreleased

- Initialized the OpportunityOS monorepo and local infrastructure.
- Added architecture records and ADRs for modularity, versioning, and ledger rules.
- Added the first neutral, executable core domain slice with tests.
- Added two PostgreSQL migrations, idempotent neutral seed data, explicit SQL, sqlc configuration, and OpenAPI foundations.
- Implemented tenant-scoped opportunity commands, audit/outbox/inbox contracts, all required lifecycle maps, controlled workflow DAG execution, schema validation, metering, integer pricing, provider routing, product publication checks, version-bound orders, Adapter contracts, and a balanced append-only ledger.
- Added HMAC webhook replay protection and public-address URL validation for crawler SSRF boundaries.
- Added seven runnable Next.js portals, with Tailwind-enabled admin and operator consoles plus shared typed contracts and UI.
- Added typed Python intelligence and crawler worker foundations with passing security tests.
- Verified the neutral end-to-end chain from opportunity evidence through publication, proof, order, execution, usage, charge, ledger, commission, settlement, and outcome feedback.

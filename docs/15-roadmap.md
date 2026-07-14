# Roadmap

## Current cycle

- [x] Establish monorepo, Compose infrastructure, architecture records, and neutral domain contracts.
- [x] Implement the first executable core vertical slice and tests.
- [x] Apply real PostgreSQL migrations and neutral seed data locally.
- [x] Build all seven portal entry points; browser-check admin and operator desktop/mobile views.
- [x] Replace in-memory API wiring with PostgreSQL repositories and transaction-scoped audit/outbox writes.
- [x] Connect admin-web and operator-console to tenant-scoped API queries and command forms using explicit development identities.
- [ ] Replace development identity headers with authenticated session claims and permission-derived commands.
- [ ] Publish persisted outbox events to Redis Streams with leases, retries, and dead-letter handling.

## First-release scope retained

Phases C-H remain in scope: full product factory, transaction/execution, finance, growth, reseller/supplier/marketplace, and production end-to-end hardening.

## Next development task

Implement authenticated operator sessions and permission-derived command authorization. Replace browser-provided tenant and actor identity with trusted server-side claims, then add PostgreSQL RLS and authorization integration tests before exposing additional portals.

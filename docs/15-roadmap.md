# Roadmap

## Current cycle

- [x] Establish monorepo, Compose infrastructure, architecture records, and neutral domain contracts.
- [x] Implement the first executable core vertical slice and tests.
- [x] Apply real PostgreSQL migrations and neutral seed data locally.
- [x] Build all seven portal entry points; browser-check admin and operator desktop/mobile views.
- [ ] Replace in-memory API wiring with PostgreSQL repositories and transaction-scoped audit/outbox writes.
- [ ] Connect admin-web and operator-console to authenticated API queries and command forms.

## First-release scope retained

Phases C-H remain in scope: full product factory, transaction/execution, finance, growth, reseller/supplier/marketplace, and production end-to-end hardening.

## Next development task

Implement pgx/sqlc repositories for opportunity, evidence, review, incubation, and blueprint aggregates. Each command must use one database transaction for aggregate state, audit, and outbox, with API contract and tenant-isolation integration tests.

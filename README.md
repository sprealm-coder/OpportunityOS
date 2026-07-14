# OpportunityOS

OpportunityOS is a product-agnostic commercial operating system. This monorepo contains the control, execution, growth, governance, and data/learning planes required to turn evidence into versioned products, orders, execution, accounting, settlement, and outcome feedback.

The current verified slice includes PostgreSQL-backed tenant-scoped opportunity, evidence, review, incubation, and business-blueprint commands; capability/provider/endpoint registration; Product, immutable ProductVersion, SKU and SKUVersion persistence; versioned schema/workflow/metering/pricing/routing bindings; gated publication; transaction-scoped audit and outbox writes; state machines; version-bound orders; execution adapters; webhook/SSRF defenses; and an append-only balanced ledger. Neutral end-to-end tests use only Test Capability, Test Provider, Test Product, and Test Customer.

This is an engineering foundation and executable control-plane slice, not a claim that every first-release module is production complete. The core API uses pgx/sqlc PostgreSQL repositories at runtime. Database-backed sessions establish trusted tenant, actor, and role claims; command routes enforce role permissions; PostgreSQL RLS runs business transactions through a non-owner role. Admin and operator portals use HttpOnly Cookie sessions, and the operator portal exposes the current opportunity-to-publication control chain. Persisted outbox delivery uses concurrent leases, exponential retry, dead-letter state, and a Redis Streams publisher; inbound events have PostgreSQL-backed tenant-scoped deduplication.

## Prerequisites

- Go 1.26+
- Node.js 24+
- pnpm 11+
- Redis 5+ for Streams delivery (Compose uses Redis 7.4)
- Docker with Compose
- GNU Make is optional; the underlying task runner works on Windows without Make.

## Commands

| Make command | Cross-platform equivalent | Purpose |
| --- | --- | --- |
| `make setup` | `corepack pnpm node scripts/task.mjs setup` | Install Go and web dependencies |
| `make migrate` | `corepack pnpm node scripts/task.mjs migrate` | Apply PostgreSQL migrations |
| `make seed` | `corepack pnpm node scripts/task.mjs seed` | Load neutral Test fixtures |
| `make dev` | `corepack pnpm node scripts/task.mjs dev` | Start dependencies and primary web consoles |
| `make test` | `corepack pnpm node scripts/task.mjs test` | Run Go and workspace tests |
| `make lint` | `corepack pnpm node scripts/task.mjs lint` | Run Go vet and TypeScript checks |
| `make build` | `corepack pnpm node scripts/task.mjs build` | Build services and web applications |
| `make e2e` | `corepack pnpm node scripts/task.mjs e2e` | Run the neutral commercial-chain acceptance test |
| `make reset` | `corepack pnpm node scripts/task.mjs reset` | Remove local containers and development data |

Start the current control-plane stack:

```bash
corepack pnpm node scripts/task.mjs dev
```

The core API listens on `:8080`; `admin-web` uses port 3000 and `operator-console` uses port 3001. Browser clients authenticate through `POST /v1/auth/sessions`; mutating API requests require `Idempotency-Key`. The neutral development login is `admin@opportunity.local` / `opportunity-dev`, scoped to tenant `00000000-0000-4000-8000-000000000001`.

Run the reliable event publisher after PostgreSQL and Redis are available:

```bash
go run ./services/core-api/cmd/outbox-worker
```

The worker accepts `DATABASE_URL`, `REDIS_URL`, `OUTBOX_STREAM`, `OUTBOX_BATCH_SIZE`, `OUTBOX_MAX_ATTEMPTS`, and `OUTBOX_POLL_MS`.

Worker tests run in a repository-local Python environment:

```bash
py -3.12 -m venv .venv
.venv/Scripts/python -m pip install -e "services/intelligence-worker[test]" -e "services/crawler-worker[test]"
.venv/Scripts/python -m pytest services/intelligence-worker/tests services/crawler-worker/tests -q
```

## Repository map

- `apps/`: role-specific Next.js portals; admin and operator are the current primary surfaces.
- `services/core-api/`: Go modular core, authenticated HTTP API, RLS-aware pgx/sqlc repositories, delivery worker, migrations, OpenAPI, and tests.
- `services/intelligence-worker/`: typed, advisory-only intelligence adapter contracts.
- `services/crawler-worker/`: public-source collection with SSRF policy enforcement.
- `packages/`: shared web contracts and UI.
- `registry/`: versionable schemas and neutral test fixtures, never vertical products.
- `docs/`: architecture, domain rules, security boundaries, roadmap, risks, and ADRs.

See [roadmap](docs/15-roadmap.md) and [risk register](docs/16-risk-register.md) for the exact verified boundary and remaining first-release work.

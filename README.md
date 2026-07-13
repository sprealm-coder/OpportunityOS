# OpportunityOS

OpportunityOS is a product-agnostic commercial operating system. This monorepo contains the control, execution, growth, governance, and data/learning planes required to turn evidence into versioned products, orders, execution, accounting, settlement, and outcome feedback.

The current verified slice includes tenant-scoped opportunity commands, audit and outbox contracts, state machines, business blueprints, capability/provider separation, schema validation, a controlled workflow DAG, metering, integer pricing, routing, immutable product publication checks, version-bound orders, execution adapters, webhook/SSRF defenses, and an append-only balanced ledger. Neutral end-to-end tests use only Test Capability, Test Provider, Test Product, and Test Customer.

This is an engineering foundation and executable vertical slice, not a claim that every first-release module is production complete. The API currently uses an in-memory opportunity repository while the PostgreSQL repository boundary is validated independently; replacing that runtime wiring is the next task.

## Prerequisites

- Go 1.26+
- Node.js 24+
- pnpm 11+
- Docker with Compose
- GNU Make is optional; the underlying task runner works on Windows without Make.

## Commands

| Make command | Cross-platform equivalent | Purpose |
| --- | --- | --- |
| `make setup` | `pnpm node scripts/task.mjs setup` | Install Go and web dependencies |
| `make migrate` | `pnpm node scripts/task.mjs migrate` | Apply PostgreSQL migrations |
| `make seed` | `pnpm node scripts/task.mjs seed` | Load neutral Test fixtures |
| `make dev` | `pnpm node scripts/task.mjs dev` | Start dependencies and primary web consoles |
| `make test` | `pnpm node scripts/task.mjs test` | Run Go and workspace tests |
| `make lint` | `pnpm node scripts/task.mjs lint` | Run Go vet and TypeScript checks |
| `make build` | `pnpm node scripts/task.mjs build` | Build services and web applications |
| `make e2e` | `pnpm node scripts/task.mjs e2e` | Run the neutral commercial-chain acceptance test |
| `make reset` | `pnpm node scripts/task.mjs reset` | Remove local containers and development data |

Start infrastructure before migration:

```bash
docker compose up -d postgres redis minio
pnpm node scripts/task.mjs migrate
pnpm node scripts/task.mjs seed
```

The core API listens on `:8080`; `admin-web` uses port 3000 and `operator-console` uses port 3001. Mutating API requests require `X-Tenant-ID`, `X-Actor-ID`, and `Idempotency-Key`.

Worker tests run in a repository-local Python environment:

```bash
py -3.12 -m venv .venv
.venv/Scripts/python -m pip install -e "services/intelligence-worker[test]" -e "services/crawler-worker[test]"
.venv/Scripts/python -m pytest services/intelligence-worker/tests services/crawler-worker/tests -q
```

## Repository map

- `apps/`: role-specific Next.js portals; admin and operator are the current primary surfaces.
- `services/core-api/`: Go modular core, migrations, explicit SQL, OpenAPI, and tests.
- `services/intelligence-worker/`: typed, advisory-only intelligence adapter contracts.
- `services/crawler-worker/`: public-source collection with SSRF policy enforcement.
- `packages/`: shared web contracts and UI.
- `registry/`: versionable schemas and neutral test fixtures, never vertical products.
- `docs/`: architecture, domain rules, security boundaries, roadmap, risks, and ADRs.

See [roadmap](docs/15-roadmap.md) and [risk register](docs/16-risk-register.md) for the exact verified boundary and remaining first-release work.

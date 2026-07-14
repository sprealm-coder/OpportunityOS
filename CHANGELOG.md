# Changelog

## Unreleased

### Added

- PostgreSQL migration `000011_phase_h_integration` for Source/Signal lineage, OutcomeFeedback projections, WorkflowRun/step leases, Adapter identities/receipts, Outbox replay records, and operational alerts under RLS.
- Source creation, generic Signal import/fingerprint deduplication, and atomic Signal-to-Opportunity/Evidence promotion.
- Persistent workflow side-effect steps with bounded leases, attempt limits, exponential retry scheduling, completion/error records, and execution bindings.
- Provider-bound Adapter identities and HMAC-SHA256 result ingress with timestamp tolerance, nonce/event replay protection, persistent Inbox receipts, and atomic execution/audit/Outbox updates.
- Settled, reconciled Order-to-Opportunity lineage validation and server-derived integer OutcomeFeedback metrics without rewriting historical score facts.
- Outbox dead-letter alerts, audited replay commands, delivery health summaries, deployment readiness checks, secure Cookie configuration, a production check command, hardened Compose contract, and GitHub Actions verification.
- Shared Phase H Zod contracts, operator Source/Signal/workflow/feedback workspace, and admin operational health/outcome views.
- OpenAPI `0.8.0` and a real PostgreSQL Source-to-OutcomeFeedback acceptance chain spanning phases A-G.

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
- PostgreSQL migration `000009_growth_engine` for MarketSegment, ICPDefinition, Lead, LeadEvidence, Contact provenance, generic Proof, Campaign approval, suppression, quotas, outreach plans, Conversation, Deal, and Experiment.
- Product-agnostic GrowthStore commands with tenant RLS, idempotency, audit, Outbox, state-machine, retention, consent, suppression, quota, and approval enforcement.
- Canonical `quotes.growth_deal_id` binding while retaining historical/external `quotes.deal_id` compatibility.
- Authenticated growth APIs, OpenAPI `0.6.0`, and shared TypeScript/Zod growth contracts.
- Operator growth-sales workspace and admin growth-governance view; outbound delivery remains visibly disabled.
- PostgreSQL phase F acceptance test covering Segment through canonical Quote and Experiment, including cross-tenant isolation and suppression-blocked outreach.
- PostgreSQL migration `000010_channel_marketplace` for reseller ownership, supplier contract/quality, developer marketplace, disputes, and takedowns under RLS.
- Reseller levels, versioned attribution, protected Lead/Customer ownership, independent transfer review, canonical CommissionLock, and settlement-cycle commands.
- Supplier capability and Provider ownership, independently approved contract lifecycle, integer rates, quality evidence, and canonical ProviderPayable/Settlement projections.
- Developer, Publisher, immutable ListingVersion, automatic/security/license/manual review, controlled sandbox, quality-gated release, dispute, and takedown commands.
- Separated channel permissions, OpenAPI `0.7.0`, shared Zod contracts, and authenticated reseller, supplier, and marketplace portals.
- Phase G PostgreSQL integration coverage for idempotency, cross-tenant isolation, four self-approval boundaries, audit, Outbox, canonical finance reuse, and Listing publication gates.

### Changed

- Browser clients now derive tenant and actor from trusted sessions instead of request headers.
- Growth Quote now aliases the canonical transaction-domain Quote instead of defining a duplicate model.
- Orders now require sufficient active held funds before payment, and executions require posted charges and provider payables before settlement.
- Browser execution transitions are administrative only; submitted, processing, succeeded, failed, external ID, output, and error facts require trusted Adapter ingress.
- `make e2e` now runs the Phase H PostgreSQL acceptance chain before the in-memory domain-engine check.
- Phase H completes the planned A-H first-release core chain; subsequent work is production pilot hardening rather than removal of retained modules.

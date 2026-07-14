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
- [x] Persist Quote, immutable QuoteVersion, and server-priced quote line bindings.
- [x] Persist Order and OrderItem snapshots bound to published ProductVersion, SKUVersion, Workflow, Pricing, and Routing records.
- [x] Create Subscription, Entitlement, ExecutionOrder, and DeliveryProject records atomically at the provisioning boundary.
- [x] Persist controlled execution/delivery transitions and separate Usage, ProviderCost, and CustomerCharge facts.
- [x] Connect the operator transaction workspace to authenticated quote-to-fulfillment commands.
- [x] Persist Wallet, LedgerAccount, append-only LedgerTransaction/LedgerEntry, and BalanceSnapshot under tenant RLS.
- [x] Enforce account-locked Hold, Release, CustomerCharge posting, Refund, Commission, ProviderPayable, and Settlement commands.
- [x] Reconcile CustomerCharge, ProviderCost, and Commission facts against accounting entries with stored discrepancies.
- [x] Gate order payment and execution settlement on financial prerequisites.
- [x] Connect operator finance operations and admin finance governance to authenticated APIs.
- [x] Persist MarketSegment, ICPDefinition, Lead, LeadEvidence, Contact provenance, generic Proof, Campaign approval, suppression, quota, outreach planning, Conversation, Deal, and Experiment under tenant RLS.
- [x] Bind canonical Growth Deal to the existing transaction Quote without creating duplicate Customer, Quote, Order, or ledger facts.
- [x] Separate growth operators from Proof/Campaign approvers and keep outbound delivery feature-gated.
- [x] Connect the operator growth-sales workspace and admin governance view to authenticated APIs and shared Zod contracts.
- [x] Verify the phase F PostgreSQL chain, idempotency, cross-tenant isolation, audit, Outbox, suppression blocking, and quota release.
- [x] Persist Reseller levels, attribution, protected Lead/Customer ownership, independent transfers, canonical Commission locks, and settlement cycles.
- [x] Persist Supplier capability/Provider ownership, independently approved contracts, integer rates, quality evidence, and canonical payable/settlement views.
- [x] Persist Developer, Publisher, immutable ListingVersion, review, sandbox, quality, dispute, and takedown workflows under tenant RLS.
- [x] Separate channel operators from ownership, supplier, marketplace, and takedown approvers with Store-level self-review guards.
- [x] Connect authenticated reseller, supplier, and marketplace portals to Core API commands and shared Zod contracts.
- [x] Verify the phase G PostgreSQL chain, idempotency, cross-tenant isolation, canonical finance reuse, release gates, audit, and Outbox.

## First-release scope retained

Phase H remains in scope: neutral end-to-end integration across phases A-G, outcome feedback, operational observability, and production hardening.

## Next development task

Implement phase H across the authenticated/RLS-aware boundary: run one neutral Source-to-OutcomeFeedback acceptance chain through opportunity, blueprint, product, growth, channel, transaction, execution, finance, marketplace governance, and analytics. Add workflow leases, trusted Adapter result ingress, operational replay/alerts, and production deployment checks without enabling external outreach or marketplace payouts.

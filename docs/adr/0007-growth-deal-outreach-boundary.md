# ADR 0007: Growth, Deal-Quote, and Outreach Boundary

## Status

Accepted

## Context

Phase F needs product-agnostic lead discovery, Proof, Campaign, conversation, and sales-pipeline records without turning a Lead into a Customer, a Deal into an Order, or a Campaign into an outbound delivery engine. OpportunityOS already has canonical Quote, Order, pricing, execution, and ledger facts. Duplicating those facts in Growth would make pricing and fulfillment ambiguous. Accepting arbitrary external content as a delivery instruction would also bypass consent, suppression, approval, and quota controls.

## Decision

Growth is a tenant-scoped PostgreSQL/RLS module with MarketSegment, ICPDefinition, Lead, LeadEvidence, Contact provenance, ProofTemplate/Request/Instance, Campaign/Step/Approval, SuppressionEntry, SendQuota, OutreachMessage, Conversation/Message, Deal, and Experiment.

- Lead remains independent from Customer. A Deal explicitly references both its source Lead and the existing customer identifier.
- Deal remains independent from Order. The existing transaction-domain Quote is reused.
- `quotes.growth_deal_id` is a nullable foreign key to canonical Deal. The existing text `deal_id` stays for historical and external compatibility.
- `POST /deals/{id}/quotes` loads the Deal server-side, derives its customer and default currency, and invokes the existing server-priced Quote command.
- Deal proposal requires a canonical Quote; Deal win requires an accepted canonical Quote.
- Proof is type, JSON Schema, workflow, retention, access-policy, and review driven. It contains no product-specific report, assistant, media, or form logic.
- Campaign steps are editable only in draft. Submission, review, and activation increment versions; activation requires an approved record for the current version.
- Suppression and consent are evaluated before quota reservation. Blocked attempts persist for audit. Quota reservations are locked and released on cancellation.
- Phase F exposes planning and cancellation only. `growth.outbound_delivery` defaults to disabled, and the browser API rejects sent/delivered transitions. Trusted adapter identity and signed result ingress are prerequisites for real delivery.
- Conversation and Proof content are untrusted structured data and cannot trigger pricing, ledger, secrets, system commands, cross-tenant access, or delivery.

Every growth command writes its business record, audit record, idempotency result, and Outbox event in one tenant transaction.

## Consequences

- Growth can evolve without owning transaction, fulfillment, or accounting truth.
- Existing external quote integrations remain compatible while new OpportunityOS Deals are referentially enforced.
- Approval, suppression, quota, and audit controls are testable before any delivery adapter exists.
- Real outbound delivery, signed provider callbacks, bounce/complaint webhook ingestion, and release policy remain explicit later work rather than hidden phase F behavior.

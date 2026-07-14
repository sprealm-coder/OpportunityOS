# ADR 0008: Channel, Supplier, and Internal Marketplace Boundary

## Status

Accepted

## Context

Phase G needs reseller ownership, supplier commercial controls, and an internal developer marketplace. OpportunityOS already owns canonical Lead, customer identifiers, Provider, Commission, ProviderPayable, Settlement, Product, workflow, and ledger facts. Recreating any of them inside a portal or channel module would split attribution, fulfillment, or financial truth. Reviews and takedowns also require actor separation and immutable evidence.

## Decision

- LeadOwnership references Growth Lead; CustomerOwnership stores the existing customer identifier. One active protected owner is permitted per tenant and subject.
- Ownership transfer is a separate request. Approval atomically updates the protected ownership and rejects requester self-review.
- CommissionLock references an existing finance Commission whose reseller beneficiary matches the ownership scope. SettlementCycle is operational grouping, not a second Settlement ledger.
- Supplier owns existing Provider records through a tenant-consistent binding. SupplierContract, SupplierRate, and SupplierQualityRecord describe commercial terms and evidence.
- Supplier payable and settlement views project canonical ProviderPayable and Settlement records. No supplier-specific payable, payment, or ledger facts are introduced.
- Developer owns Publisher; Publisher owns Listing; ListingVersion is immutable. Listing is metadata and governance around an artifact, not a duplicate Product, Capability, workflow, Agent, or MCP fact.
- Release requires the latest version to pass automated review, security review, license review, manual review, a controlled sandbox result, and a quality score of at least 8000 basis points.
- Listing creators cannot review their own versions. Takedown requesters cannot approve their own request. Removal occurs only through takedown review and preserves versions, reviews, disputes, incidents, audit, and Outbox history.
- Marketplace payout remains feature-disabled. Phase G creates no external money movement.

All commands execute under tenant RLS, idempotency reservation, audit, and Outbox in one PostgreSQL transaction.

## Consequences

- Reseller and supplier portals can expose operational ownership and finance projections without becoming sources of accounting truth.
- Provider routing can later select a versioned SupplierContract without changing Product or SKU definitions.
- Marketplace governance is testable before trusted sandbox workers, signed artifacts, or payouts are enabled.
- Expiry automation, arbitration SLA, trusted sandbox identity, route-time contract binding, and payout posting remain explicit phase H work.

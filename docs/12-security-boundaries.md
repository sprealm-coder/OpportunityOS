# Security Boundaries

Tenant scope is mandatory in repositories, cache keys, object paths, queue envelopes, and APIs. External content is untrusted and cannot invoke ledger, price, permission, messaging, secret, cross-tenant, command, or privileged tool operations.

The crawler permits only public HTTP/HTTPS targets after DNS resolution and revalidates every redirect to block loopback, private, link-local, multicast, and cloud metadata destinations.

Growth commands separate operators from approvers: operators may configure Proof and Campaign records but cannot review them; reviewers can review but cannot mutate lead, outreach, deal, or experiment records. Active suppression and opted-out contacts block planning before quota reservation. Contact identifiers are normalized but suppression lookups use tenant-scoped SHA-256 keys. Public APIs can create outbound drafts or plans and can cancel plans, but cannot record delivery while `growth.outbound_delivery` is disabled. External content is stored as untrusted JSON and cannot initiate pricing, ledger, secret, cross-tenant, command, or outbound-delivery actions.

Channel commands apply the same separation. Operators can create reseller ownership requests, supplier contracts, and marketplace Listings, but cannot approve ownership transfers, supplier contracts, Listing reviews, disputes, or takedowns. Store commands additionally compare actor identifiers so admin wildcard permission cannot bypass creator/reviewer separation. Supplier/Provider and every review target are checked against the transaction tenant. Sandbox endpoints accept only structured policy and result records from authorized reviewers; they do not execute uploaded code. Marketplace payout remains disabled and cannot create external money movement.

# Security Boundaries

Tenant scope is mandatory in repositories, cache keys, object paths, queue envelopes, and APIs. External content is untrusted and cannot invoke ledger, price, permission, messaging, secret, cross-tenant, command, or privileged tool operations.

The crawler permits only public HTTP/HTTPS targets after DNS resolution and revalidates every redirect to block loopback, private, link-local, multicast, and cloud metadata destinations.


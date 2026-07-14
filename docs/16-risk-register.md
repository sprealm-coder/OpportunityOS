# Risk Register

| Risk | Severity | Current control | Next action |
| --- | --- | --- | --- |
| Domain breadth may hide inconsistent invariants | High | One versioned aggregate chain and neutral E2E test | Add repository contract tests per module |
| Tenant scope omission | Critical | PostgreSQL RLS and non-owner business transactions enforce the trusted session tenant, including product factory binding tables | Extend RLS tests with each transaction and finance repository |
| Money posting error | Critical | Integer minor units and balanced append-only ledger | Add concurrency and reversal integration tests |
| Workflow side effects repeat | High | Persisted Inbox deduplication and leased Outbox delivery | Persist workflow step leases and retry schedules |
| SSRF through crawler redirects/DNS | Critical | Documented deny policy | Implement resolver pinning and redirect tests in crawler worker |
| Portal skeletons may outpace backend authorization | Medium | Feature gates and API-side checks are mandatory | Connect only authorized routes per release channel |
| Session theft or stale privileges | High | HttpOnly SameSite cookies, hashed tokens, expiry, revocation, and membership resolution on each request | Add production TLS-only cookie policy, session rotation, and membership-change revocation |
| Outbox delivery outage | High | Leases, ownership checks, exponential retry, dead-letter state, and Redis Streams adapter | Add operational dead-letter replay and delivery-lag alerts |
| Redis and MinIO images were not pulled | Medium | Compose contracts are present; PostgreSQL is healthy | Retry image pulls when Docker Hub TLS is stable, then add health integration tests |
| Shared portal navigation uses placeholder anchors | Medium | Each portal has one bounded responsibility view | Add authorized routes only as backing commands become available |
| Published version references may drift | Critical | ProductVersion and SKUVersion are append-only records with explicit workflow/metering/pricing/routing bindings | Make phase D order creation reject mutable or unpublished bindings |

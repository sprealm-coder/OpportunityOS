# Risk Register

| Risk | Severity | Current control | Next action |
| --- | --- | --- | --- |
| Domain breadth may hide inconsistent invariants | High | One versioned aggregate chain and neutral E2E test | Add repository contract tests per module |
| Tenant scope omission | Critical | PostgreSQL RLS and non-owner business transactions enforce the trusted session tenant, including product factory and transaction/execution tables | Extend RLS tests with each finance repository |
| Money posting error | Critical | Integer minor units, account locks, deferred debit/credit validation, immutable runtime history, snapshots, idempotency, and PostgreSQL integration tests | Add external payment reconciliation and multi-currency FX policy before production money movement |
| Workflow side effects repeat | High | Persisted Inbox deduplication and leased Outbox delivery | Persist workflow step leases and retry schedules |
| SSRF through crawler redirects/DNS | Critical | Documented deny policy | Implement resolver pinning and redirect tests in crawler worker |
| Portal skeletons may outpace backend authorization | Medium | Feature gates and API-side checks are mandatory | Connect only authorized routes per release channel |
| Session theft or stale privileges | High | HttpOnly SameSite cookies, hashed tokens, expiry, revocation, and membership resolution on each request | Add production TLS-only cookie policy, session rotation, and membership-change revocation |
| Outbox delivery outage | High | Leases, ownership checks, exponential retry, dead-letter state, and Redis Streams adapter | Add operational dead-letter replay and delivery-lag alerts |
| Redis and MinIO images were not pulled | Medium | Compose contracts are present; PostgreSQL is healthy | Retry image pulls when Docker Hub TLS is stable, then add health integration tests |
| Shared portal navigation uses placeholder anchors | Medium | Each portal has one bounded responsibility view | Add authorized routes only as backing commands become available |
| Published version references may drift | Critical | Quote creation requires an active Publication; OrderItem copies immutable product, SKU, workflow, pricing, and routing identifiers | Add contract-version binding when supplier contracts become persistent |
| Operational charge may be mistaken for posted revenue | Critical | Usage, ProviderCost, CustomerCharge, and LedgerEntry are separate; explicit posting atomically advances CustomerCharge and writes balanced entries | Require finance views to display calculated and posted amounts separately in every future portal |
| Administrative wallet adjustment may be abused | Critical | Dedicated `finance.adjust` permission is admin-only; every adjustment is idempotent, balanced, audited, and emitted through Outbox | Add dual approval and policy limits before production funding adjustments |
| Reconciliation discrepancy may remain unresolved | High | Runs persist per-fact expected/actual values and discrepancy records without changing ledger history | Add assignment, SLA, evidence, and adjustment-approval workflow |
| Manual execution transition may overstate completion | High | Controlled state machines, provider readiness, delivery completion, and activation gates | Bind Adapter results and workflow step leases to execution transitions |
| Suppressed contact may be planned or quota-counted | Critical | Tenant-keyed active suppression is checked before quota reservation; suppressed attempts persist as `blocked`; cancellation releases reservations | Add provider-level bounce/complaint ingestion through signed, replay-protected webhooks |
| Campaign operator may self-approve outbound scope | Critical | Separate `campaign.write` and `campaign.approve` permissions; approval binds the exact campaign version | Add dual approval and policy thresholds before enabling any delivery adapter |
| External content may trigger delivery or privileged actions | Critical | Conversation content and Proof results are untrusted JSON; public outreach API permits planning/cancellation only; `growth.outbound_delivery` defaults off | Introduce a trusted adapter identity and signed delivery-result ingress before real sending |
| Deal, Quote, and Order facts may diverge | High | `quotes.growth_deal_id` references canonical Deal; server derives customer from Deal; proposal/win gates check Quote state; Order remains transaction-owned | Add quote-version amendment and multi-deal merge/split policies before advanced CRM workflows |

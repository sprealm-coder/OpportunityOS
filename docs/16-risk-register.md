# Risk Register

| Risk | Severity | Current control | Next action |
| --- | --- | --- | --- |
| Domain breadth may hide inconsistent invariants | High | One versioned aggregate chain and neutral E2E test | Add repository contract tests per module |
| Tenant scope omission | Critical | Tenant context and tenant-keyed repositories | Add PostgreSQL RLS for financial and identity tables |
| Money posting error | Critical | Integer minor units and balanced append-only ledger | Add concurrency and reversal integration tests |
| Workflow side effects repeat | High | Tenant-scoped idempotency and inbox/outbox contracts | Persist step leases and retry schedules |
| SSRF through crawler redirects/DNS | Critical | Documented deny policy | Implement resolver pinning and redirect tests in crawler worker |
| Portal skeletons may outpace backend authorization | Medium | Feature gates and API-side checks are mandatory | Connect only authorized routes per release channel |
| Browser development identity headers are not authentication | Critical | API remains tenant scoped and development-only portals use a fixed neutral tenant | Replace headers with trusted session claims and permission-derived commands |
| Outbox events are persisted but not delivered | High | Aggregate, audit, and outbox writes share one PostgreSQL transaction | Add leased Redis Streams publisher, retries, and dead-letter handling |
| Redis and MinIO images were not pulled | Medium | Compose contracts are present; PostgreSQL is healthy | Retry image pulls when Docker Hub TLS is stable, then add health integration tests |
| Shared portal navigation uses placeholder anchors | Medium | Each portal has one bounded responsibility view | Add authorized routes only as backing commands become available |

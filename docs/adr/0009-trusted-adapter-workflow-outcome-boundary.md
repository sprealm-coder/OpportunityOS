# ADR 0009: Trusted Adapter, workflow lease, and outcome boundary

## Status

Accepted.

## Context

The phase D browser route could write external execution IDs and successful output. Workflow tables existed but did not own durable leases or retry schedules. Source/Signal and OutcomeFeedback tables were not connected to the PostgreSQL commercial chain, so the previous acceptance test could only name those checkpoints in memory.

## Decision

Keep ExecutionOrder, Order, finance, and Opportunity as the canonical facts. Persist side-effect WorkflowStepRun leases against ExecutionOrder and ProviderEndpoint-bound Adapter identities. Accept external results only through timestamped HMAC requests with persistent nonce/event deduplication and active lease ownership. Commit Inbox receipt, Adapter receipt, workflow/execution state, audit, and Outbox in one tenant transaction.

Persist Source/Signal lineage explicitly. Accept OutcomeFeedback only after blueprint lineage, order completion, execution settlement, finance settlement, and matched reconciliation checks. Derive monetary metrics from canonical integer facts and retain versioned feedback while updating a read projection on Opportunity.

Outbox dead letters create operational alerts. Admin replay preserves the original payload, records reason and prior attempts, and produces its own audit/event trail. External outreach and marketplace payout stay disabled.

## Consequences

- A compromised browser session cannot fabricate provider success or output.
- Duplicate or stale Adapter results cannot repeat execution side effects.
- Retry ownership survives process restarts and concurrent workers.
- Commercial feedback is traceable to the exact Source-to-Order lineage.
- Deployments must inject Adapter secrets and pass TLS, RLS, feature-gate, alert, and dead-letter checks.
- Full DAG dependency scheduling, compensation orchestration, Adapter key rotation, mTLS, and real external delivery/payment adapters remain production-hardening work.

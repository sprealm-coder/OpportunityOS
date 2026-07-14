# State Machines

State transitions are domain commands. Each command checks tenant, actor permission, current state, preconditions, and idempotency before updating and appending an audit record plus outbox event.

Implemented transition maps live in `internal/state`. Unsupported transitions return `invalid_transition`; callers never write status strings directly.

Phase F adds:

- Lead: `discovered -> enriched -> qualified -> proof_requested -> proof_ready -> approved_for_outreach -> contacted -> replied -> meeting -> proposal -> won`, with controlled lost and suppression exits.
- ProofRequest: `requested -> processing -> review -> ready`, with rejected, expired, and deleted exits.
- Campaign: `draft -> pending_approval -> approved -> active -> paused/completed`, with rejection and cancellation exits. Approval is stored against the post-review version and activation checks that exact version.
- OutreachMessage: `planned -> sent -> delivered/replied/bounced/complained` plus cancellation. The public phase F API exposes cancellation only; trusted adapter delivery remains feature-gated.
- Deal: `open -> proposal -> won`, with lost and cancellation exits. Proposal and won are gated by canonical Quote state.
- Experiment: `draft -> running -> completed`, with cancellation exits and a required result on completion.

Phase G adds:

- SupplierContract: `draft -> pending_approval -> approved -> active -> suspended/expired/terminated`. The creator cannot approve; activation requires an active integer rate bound to a Supplier capability.
- Listing: `draft -> submitted -> automated_review -> manual_review -> sandbox_testing -> limited_release -> published`, with suspension and governed removal exits. The latest immutable version must pass automated, security, license, and manual reviews, a controlled sandbox run, and a quality score of at least 8000 basis points.
- OwnershipTransfer: `pending -> approved/rejected/cancelled`. Approval atomically moves the protected LeadOwnership or CustomerOwnership, and the requester cannot review it.
- Takedown: `requested -> executed/rejected`. Only the governed review command can move a Listing to `removed`; records remain append-only and visible to audit.

# State Machines

State transitions are domain commands. Each command checks tenant, actor permission, current state, preconditions, and idempotency before updating and appending an audit record plus outbox event.

Implemented transition maps live in `internal/state`. Unsupported transitions return `invalid_transition`; callers never write status strings directly.


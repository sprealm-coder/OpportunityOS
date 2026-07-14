# Workflow Engine

The engine validates a versioned DAG and supports controlled node types: start, validate, transform, condition, approval, realtime_call, async_submit, async_wait, provision, manual_task, webhook_wait, meter, notify, compensate, and end.

Runs and steps are persisted separately. Idempotency is scoped by tenant and workflow version. A queued ExecutionOrder creates one WorkflowRun and persisted side-effect steps. A due step can be leased only to an active Adapter identity whose ProviderEndpoint matches the execution route. Leases have an owner, expiry, attempt, maximum attempt count, and next retry time; expired leases can be safely reclaimed with `FOR UPDATE SKIP LOCKED`.

Adapter failures below the attempt limit move the step to `retry_wait` with exponential backoff. Exhausted and unknown results create operational alerts; unknown results move execution into reconciliation. Successful receipts complete the owned step and only complete the run when no unfinished step remains. Pause, resume, cancel, compensation, parallel scheduling, and multi-node dependency release remain explicit follow-up engine commands rather than arbitrary code execution.

# Workflow Engine

The engine validates a versioned DAG and supports controlled node types: start, validate, transform, condition, approval, realtime_call, async_submit, async_wait, provision, manual_task, webhook_wait, meter, notify, compensate, and end.

Runs and steps are persisted separately. Idempotency is scoped by tenant and workflow version. Pause, resume, cancel, retry, timeout, reconciliation, and compensation are explicit commands.


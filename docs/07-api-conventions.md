# API Conventions

JSON APIs use `/v1`, UUID-compatible opaque IDs, RFC 3339 timestamps, and integer minor-unit money. Mutating requests require `Idempotency-Key`; tenant routes require `X-Tenant-ID` and `X-Actor-ID`. Errors contain `code`, `message`, `request_id`, and optional field details. Pagination is cursor based.


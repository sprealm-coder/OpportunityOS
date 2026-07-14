# ADR 0004: Product publication is an immutable version boundary

## Status

Accepted.

## Decision

A Product is a lifecycle aggregate. ProductVersion and SKUVersion are append-only definitions. Each ProductVersion owns explicit tenant-scoped bindings to capabilities, workflow, metering, price book, route policy, form/output schema, growth playbook, delivery mode, and compliance profile.

Publication is a domain command, not a direct status update. It succeeds only when all JSON Schemas and the controlled workflow are valid, every required capability has an active Provider with a healthy endpoint, pricing and metering definitions are valid, governance bindings exist, and at least one SKUVersion references the ProductVersion.

The Product, Publication, audit record, and Outbox event are committed in one PostgreSQL transaction under the tenant RLS role. A later release creates new ProductVersion and SKUVersion records; it does not modify a published version.

## Consequences

- Orders can bind immutable commercial and execution definitions.
- Provider changes do not alter the customer-facing product definition.
- Publication failures are explicit and leave no partial records.
- New versions increase storage and require lifecycle tooling instead of in-place edits.

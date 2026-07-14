# Architecture

The first release is a Go modular monolith using chi, pgx, explicit SQL, PostgreSQL, Redis Streams, and an S3-compatible object store. Transactional writes place domain events in the PostgreSQL outbox. Workers deliver them to Redis Streams and deduplicate inbox records.

Web applications live in a pnpm workspace. Intelligence and crawler workers are Python processes behind typed adapters. Deployment starts with Docker Compose; later Kafka, Temporal, OpenFGA, OpenMeter, and analytical stores remain replaceable adapters.

The transaction boundary starts from an accepted immutable QuoteVersion. Order creation copies every ProductVersion, SKUVersion, Workflow, Pricing, and Routing identifier into OrderItem facts. Moving an order to provisioning creates Subscription, Entitlement, ExecutionOrder, and DeliveryProject records in the same tenant-scoped transaction. Usage, ProviderCost, and CustomerCharge remain separate operational facts; phase E converts eligible charges into append-only balanced ledger transactions.

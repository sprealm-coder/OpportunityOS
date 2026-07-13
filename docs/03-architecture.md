# Architecture

The first release is a Go modular monolith using chi, pgx, explicit SQL, PostgreSQL, Redis Streams, and an S3-compatible object store. Transactional writes place domain events in the PostgreSQL outbox. Workers deliver them to Redis Streams and deduplicate inbox records.

Web applications live in a pnpm workspace. Intelligence and crawler workers are Python processes behind typed adapters. Deployment starts with Docker Compose; later Kafka, Temporal, OpenFGA, OpenMeter, and analytical stores remain replaceable adapters.


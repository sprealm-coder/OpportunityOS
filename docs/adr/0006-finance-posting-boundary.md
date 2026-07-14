# ADR 0006: Finance Posting Boundary

## Status

Accepted

## Context

Phase D persisted Usage, ProviderCost, and calculated CustomerCharge facts. Treating those rows as balances would merge metering, commercial pricing, provider economics, and accounting. It would also leave no reliable boundary for holds, refunds, commissions, settlements, or reconciliation.

## Decision

OpportunityOS uses a tenant-scoped, append-only double-entry ledger in PostgreSQL. Operational facts are converted into accounting effects only by explicit idempotent commands:

- PlaceOrderHold and ReleaseHold move value between wallet available and held liability accounts.
- PostCustomerCharge converts a calculated CustomerCharge into held-liability debit and platform-revenue credit entries.
- CreateProviderPayable converts a ProviderCost into provider-cost expense and provider-payable entries.
- CreateCommission records commission expense and a beneficiary payable independently from the charge.
- CreateSettlement reduces a provider or commission payable against platform cash.
- RefundCustomerCharge creates a new revenue debit and wallet-liability credit transaction.

Every command locks affected accounts, validates integer minor-unit amounts and business ownership, writes balanced entries and balance snapshots, and commits audit and Outbox records in the same transaction. PostgreSQL RLS scopes all finance tables. Deferred database validation rejects unbalanced transactions, and the runtime role cannot update or delete ledger history.

Order `paid` now requires sufficient active held funds. Execution `settled` requires posted customer charges and ProviderPayables for all recorded provider costs.

## Consequences

- Wallet and platform balances can be rebuilt from entries; snapshots are projections rather than facts.
- Usage, cost, charge, and ledger data can be reconciled instead of being silently conflated.
- Refunds and corrections add compensating history, increasing row count while preserving auditability.
- Provider and reseller master-data workflows remain later-phase concerns; phase E records neutral beneficiary identifiers and accounting obligations without hardcoding a business model.

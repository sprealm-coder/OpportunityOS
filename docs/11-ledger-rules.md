# Ledger Rules

The PostgreSQL ledger is the accounting record; Usage, ProviderCost, CustomerCharge, Hold, Commission, ProviderPayable, and Settlement remain operational records that reference accounting transactions. None of those facts is itself a LedgerEntry.

## Invariants

- Amounts use signed `bigint` balances and positive integer minor-unit entries. Floating point money is forbidden.
- A LedgerTransaction contains at least two entries, one currency, and equal debit and credit totals.
- The runtime database role cannot update or delete LedgerTransaction, LedgerEntry, or BalanceSnapshot rows.
- Corrections create refund, reversal, or adjustment transactions. Historical entries are never edited.
- Tenant RLS applies to wallets, accounts, entries, snapshots, holds, refunds, commissions, payables, settlements, and reconciliation records.
- Account rows are locked in stable identifier order before funds checks and posting, so concurrent commands cannot spend the same available balance.
- Command idempotency, business state, audit, Outbox, transaction, entries, and snapshots commit in one PostgreSQL transaction.
- BalanceSnapshot is an append-only projection. Current balances remain rebuildable from LedgerEntry rows.

## Posting Matrix

| Action | Debit | Credit | Accounting effect |
| --- | --- | --- | --- |
| Wallet funding adjustment | Platform cash | Customer wallet available liability | Records funded customer value |
| Hold | Wallet available liability | Wallet held liability | Reserves funds; no revenue |
| Release | Wallet held liability | Wallet available liability | Returns unused reservation |
| Customer charge | Wallet held liability | Platform revenue | Posts an eligible CustomerCharge |
| Refund | Platform revenue | Wallet available liability | Returns posted customer value with a new transaction |
| Provider payable | Provider cost expense | Provider payable liability | Converts ProviderCost into an accounting obligation |
| Commission | Commission expense | Beneficiary payable liability | Records a separate commission obligation |
| Settlement | Payable liability | Platform cash | Pays provider or commission beneficiary |

## State Gates

- An order cannot transition to `paid` until active holds cover its positive amount.
- A calculated positive CustomerCharge can be posted only once and only against same-order, same-currency held funds.
- A hold may be partially captured; any remainder must be released explicitly.
- Total refunds cannot exceed the posted CustomerCharge.
- Total commissions cannot exceed the posted CustomerCharge.
- A ProviderCost creates at most one ProviderPayable.
- An execution cannot transition to `settled` while its charge is unposted or a ProviderCost lacks a payable.
- Settlement cannot exceed outstanding payable value or available platform cash.

## Reconciliation

Reconciliation compares CustomerCharge with charge credits, ProviderCost with ProviderPayable, and Commission with its payable credit. Each run stores matched items or append-only discrepancy records. Resolving a discrepancy requires a new Adjustment; reconciliation never rewrites the source fact or ledger history.

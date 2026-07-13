# ADR 0001: Modular Monolith First

Status: Accepted

OpportunityOS starts as one Go modular monolith with bounded workers. Cross-domain invariants, transactions, and the ledger remain local and explicit. Modules publish events through the transactional outbox. A service extraction is allowed only when ownership, load, and failure isolation justify it.


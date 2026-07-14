# Event Catalog

Initial events: `tenant.created`, `signal.imported`, `opportunity.created`, `opportunity.transitioned`, `blueprint.created`, `product.drafted`, `product.published`, `workflow.started`, `workflow.completed`, `order.created`, `wallet.created`, `wallet.adjusted`, `funds.held`, `funds.released`, `customer_charge.posted`, `customer_charge.refunded`, `provider_payable.recorded`, `commission.recorded`, `settlement.completed`, `reconciliation.completed`, and `outcome.recorded`.

Every envelope carries event ID, tenant ID, aggregate ID, aggregate version, trace ID, occurred-at time, and schema version.

Phase F emits `market_segment.created`, `icp.created`, `lead.created`, `lead.evidence_added`, `lead.transitioned`, `contact.created`, `proof_template.created`, `proof.requested`, `proof.generated`, `proof.reviewed`, `campaign.created`, `campaign.step_added`, `campaign.transitioned`, `campaign.reviewed`, `suppression.created`, `outreach.planned`, `outreach.blocked`, `outreach.transitioned`, `conversation.created`, `conversation.message_added`, `deal.created`, `deal.transitioned`, `experiment.created`, and `experiment.transitioned`. Business rows, audit records, and these Outbox envelopes commit in one transaction.

Phase G emits reseller level, reseller, attribution, ownership assignment/transfer, commission lock, settlement cycle, supplier capability/Provider binding, contract/rate/quality, developer/publisher, Listing version/transition/review/sandbox/quality, dispute, and takedown events. Event names follow `<aggregate>.<past-tense action>` and commit with the business row, audit row, and idempotency response. Canonical Commission, ProviderPayable, and Settlement events remain finance-owned and are not duplicated by channel commands.

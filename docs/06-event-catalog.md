# Event Catalog

Initial events: `tenant.created`, `signal.imported`, `opportunity.created`, `opportunity.transitioned`, `blueprint.created`, `product.drafted`, `product.published`, `workflow.started`, `workflow.completed`, `order.created`, `wallet.created`, `wallet.adjusted`, `funds.held`, `funds.released`, `customer_charge.posted`, `customer_charge.refunded`, `provider_payable.recorded`, `commission.recorded`, `settlement.completed`, `reconciliation.completed`, and `outcome.recorded`.

Every envelope carries event ID, tenant ID, aggregate ID, aggregate version, trace ID, occurred-at time, and schema version.

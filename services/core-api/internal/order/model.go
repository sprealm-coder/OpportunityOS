package order

import (
	"time"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/state"
)

type VersionBindings struct{ ProductVersionID, SKUVersionID, PricingVersionID, WorkflowVersionID, RoutingVersionID, ContractVersionID string }
type Order struct {
	ID, TenantID, CustomerID, Status, Currency, IdempotencyKey string
	AmountMinor                                                int64
	Bindings                                                   VersionBindings
	CreatedAt, UpdatedAt                                       time.Time
}

func New(tenantID, customerID, idempotencyKey, currency string, amountMinor int64, bindings VersionBindings) (Order, error) {
	if tenantID == "" {
		return Order{}, platform.ErrTenantRequired
	}
	if idempotencyKey == "" {
		return Order{}, platform.ErrIdempotencyKeyRequired
	}
	if amountMinor < 0 {
		return Order{}, platform.Invalid("invalid_amount", "amount cannot be negative")
	}
	now := time.Now().UTC()
	return Order{ID: platform.NewID("ord"), TenantID: tenantID, CustomerID: customerID, Status: "created", Currency: currency, AmountMinor: amountMinor, IdempotencyKey: idempotencyKey, Bindings: bindings, CreatedAt: now, UpdatedAt: now}, nil
}
func (o *Order) Transition(to string) error {
	if err := state.Order.Transition(o.Status, to); err != nil {
		return err
	}
	o.Status = to
	o.UpdatedAt = time.Now().UTC()
	return nil
}

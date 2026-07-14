package order

import (
	"time"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/state"
)

type VersionBindings struct {
	ProductVersionID  string `json:"product_version_id"`
	SKUVersionID      string `json:"sku_version_id"`
	PricingVersionID  string `json:"pricing_version_id"`
	WorkflowVersionID string `json:"workflow_version_id"`
	RoutingVersionID  string `json:"routing_version_id"`
	ContractVersionID string `json:"contract_version_id,omitempty"`
}

func (b VersionBindings) Validate() error {
	if b.ProductVersionID == "" || b.SKUVersionID == "" || b.PricingVersionID == "" || b.WorkflowVersionID == "" || b.RoutingVersionID == "" {
		return platform.Invalid("version_bindings_incomplete", "product, SKU, pricing, workflow, and routing versions are required")
	}
	return nil
}

type Quote struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenant_id"`
	DealID     string         `json:"deal_id"`
	CustomerID string         `json:"customer_id"`
	Status     string         `json:"status"`
	Version    int            `json:"version"`
	CreatedBy  string         `json:"created_by"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	Versions   []QuoteVersion `json:"versions"`
}

type QuoteVersion struct {
	ID          string      `json:"id"`
	TenantID    string      `json:"tenant_id"`
	QuoteID     string      `json:"quote_id"`
	Version     int         `json:"version"`
	Currency    string      `json:"currency"`
	AmountMinor int64       `json:"amount_minor"`
	ValidUntil  time.Time   `json:"valid_until"`
	CreatedBy   string      `json:"created_by"`
	CreatedAt   time.Time   `json:"created_at"`
	Items       []QuoteItem `json:"items"`
}

type QuoteItem struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	QuoteVersionID  string          `json:"quote_version_id"`
	Quantity        int64           `json:"quantity"`
	UnitAmountMinor int64           `json:"unit_amount_minor"`
	AmountMinor     int64           `json:"amount_minor"`
	Input           map[string]any  `json:"input"`
	Bindings        VersionBindings `json:"bindings"`
}

type Order struct {
	ID              string            `json:"id"`
	TenantID        string            `json:"tenant_id"`
	CustomerID      string            `json:"customer_id"`
	QuoteVersionID  string            `json:"quote_version_id"`
	Status          string            `json:"status"`
	Currency        string            `json:"currency"`
	IdempotencyKey  string            `json:"idempotency_key"`
	AmountMinor     int64             `json:"amount_minor"`
	Version         int               `json:"version"`
	Bindings        VersionBindings   `json:"bindings"`
	CreatedBy       string            `json:"created_by"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Items           []OrderItem       `json:"items"`
	Subscriptions   []Subscription    `json:"subscriptions"`
	Entitlements    []Entitlement     `json:"entitlements"`
	Executions      []ExecutionOrder  `json:"executions"`
	Deliveries      []DeliveryProject `json:"deliveries"`
	Usage           []UsageRecord     `json:"usage"`
	ProviderCosts   []ProviderCost    `json:"provider_costs"`
	CustomerCharges []CustomerCharge  `json:"customer_charges"`
}

type OrderItem struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	OrderID         string          `json:"order_id"`
	Quantity        int64           `json:"quantity"`
	UnitAmountMinor int64           `json:"unit_amount_minor"`
	AmountMinor     int64           `json:"amount_minor"`
	Input           map[string]any  `json:"input"`
	Bindings        VersionBindings `json:"bindings"`
}

type Subscription struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id"`
	OrderID      string     `json:"order_id"`
	OrderItemID  string     `json:"order_item_id"`
	CustomerID   string     `json:"customer_id"`
	SKUVersionID string     `json:"sku_version_id"`
	Status       string     `json:"status"`
	StartsAt     *time.Time `json:"starts_at,omitempty"`
	EndsAt       *time.Time `json:"ends_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type Entitlement struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	OrderID        string     `json:"order_id"`
	OrderItemID    string     `json:"order_item_id"`
	SubscriptionID string     `json:"subscription_id"`
	Key            string     `json:"key"`
	Status         string     `json:"status"`
	Value          any        `json:"value"`
	StartsAt       *time.Time `json:"starts_at,omitempty"`
	EndsAt         *time.Time `json:"ends_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type ExecutionOrder struct {
	ID                 string          `json:"id"`
	TenantID           string          `json:"tenant_id"`
	OrderID            string          `json:"order_id"`
	OrderItemID        string          `json:"order_item_id"`
	Status             string          `json:"status"`
	ProviderEndpointID string          `json:"provider_endpoint_id,omitempty"`
	ExternalID         string          `json:"external_id,omitempty"`
	IdempotencyKey     string          `json:"idempotency_key"`
	CreatedBy          string          `json:"created_by"`
	Attempt            int             `json:"attempt"`
	Input              map[string]any  `json:"input"`
	Output             map[string]any  `json:"output"`
	Error              map[string]any  `json:"error"`
	Bindings           VersionBindings `json:"bindings"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type DeliveryProject struct {
	ID               string           `json:"id"`
	TenantID         string           `json:"tenant_id"`
	OrderID          string           `json:"order_id"`
	OrderItemID      string           `json:"order_item_id"`
	ExecutionOrderID string           `json:"execution_order_id"`
	Mode             string           `json:"mode"`
	Status           string           `json:"status"`
	Assignee         string           `json:"assignee,omitempty"`
	Artifacts        []map[string]any `json:"artifacts"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

type UsageRecord struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	OrderID          string    `json:"order_id"`
	OrderItemID      string    `json:"order_item_id"`
	ExecutionOrderID string    `json:"execution_order_id"`
	MeterID          string    `json:"meter_id"`
	IdempotencyKey   string    `json:"idempotency_key"`
	CreatedBy        string    `json:"created_by"`
	Quantity         int64     `json:"quantity"`
	OccurredAt       time.Time `json:"occurred_at"`
	CreatedAt        time.Time `json:"created_at"`
}

type ProviderCost struct {
	ID                 string    `json:"id"`
	TenantID           string    `json:"tenant_id"`
	OrderID            string    `json:"order_id"`
	OrderItemID        string    `json:"order_item_id"`
	ExecutionOrderID   string    `json:"execution_order_id"`
	ProviderEndpointID string    `json:"provider_endpoint_id"`
	Currency           string    `json:"currency"`
	IdempotencyKey     string    `json:"idempotency_key"`
	CreatedBy          string    `json:"created_by"`
	AmountMinor        int64     `json:"amount_minor"`
	CreatedAt          time.Time `json:"created_at"`
}

type CustomerCharge struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	OrderID          string    `json:"order_id"`
	OrderItemID      string    `json:"order_item_id"`
	ExecutionOrderID string    `json:"execution_order_id"`
	PriceBookID      string    `json:"price_book_id"`
	Currency         string    `json:"currency"`
	Status           string    `json:"status"`
	IdempotencyKey   string    `json:"idempotency_key"`
	CreatedBy        string    `json:"created_by"`
	AmountMinor      int64     `json:"amount_minor"`
	CreatedAt        time.Time `json:"created_at"`
}

func (q *Quote) Transition(to string) error {
	if err := state.Quote.Transition(q.Status, to); err != nil {
		return err
	}
	q.Status = to
	q.Version++
	q.UpdatedAt = time.Now().UTC()
	return nil
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
	if len(currency) != 3 {
		return Order{}, platform.Invalid("invalid_currency", "currency must use a three-letter code")
	}
	if err := bindings.Validate(); err != nil {
		return Order{}, err
	}
	now := time.Now().UTC()
	return Order{ID: platform.NewID("ord"), TenantID: tenantID, CustomerID: customerID, Status: "created", Currency: currency, AmountMinor: amountMinor, Version: 1, IdempotencyKey: idempotencyKey, Bindings: bindings, CreatedAt: now, UpdatedAt: now}, nil
}
func (o *Order) Transition(to string) error {
	if err := state.Order.Transition(o.Status, to); err != nil {
		return err
	}
	o.Status = to
	o.Version++
	o.UpdatedAt = time.Now().UTC()
	return nil
}

func (e *ExecutionOrder) Transition(to string) error {
	if err := state.Execution.Transition(e.Status, to); err != nil {
		return err
	}
	e.Status = to
	e.UpdatedAt = time.Now().UTC()
	return nil
}

func (d *DeliveryProject) Transition(to string) error {
	if err := state.Delivery.Transition(d.Status, to); err != nil {
		return err
	}
	d.Status = to
	d.UpdatedAt = time.Now().UTC()
	return nil
}

func ValidateLine(quantity, unitAmountMinor, amountMinor int64) error {
	if quantity <= 0 {
		return platform.Invalid("invalid_quantity", "quantity must be positive")
	}
	if unitAmountMinor < 0 || amountMinor < 0 {
		return platform.Invalid("invalid_amount", "amount cannot be negative")
	}
	if quantity > 0 && unitAmountMinor > 0 && unitAmountMinor > (1<<63-1)/quantity {
		return platform.Invalid("amount_overflow", "line amount exceeds the supported range")
	}
	return nil
}

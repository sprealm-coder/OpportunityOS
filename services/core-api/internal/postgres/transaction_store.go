package postgres

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/application"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/catalog"
	orderdomain "github.com/opportunity-os/opportunity-os/services/core-api/internal/order"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/pricing"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

type saleBinding struct {
	Bindings     orderdomain.VersionBindings
	MeteringID   string
	DeliveryMode string
	Entitlements map[string]any
	Currency     string
	AmountMinor  int64
}

func loadSaleBinding(ctx context.Context, tx pgx.Tx, tenantID, skuVersionID string, quantity int64) (saleBinding, error) {
	if quantity <= 0 {
		return saleBinding{}, platform.Invalid("invalid_quantity", "quantity must be positive")
	}
	var productVersionID, bindingsJSON, priceBookID, currency, workflowID, meteringID, routingID, deliveryMode string
	err := tx.QueryRow(ctx, `
		SELECT version.product_version_id,version.bindings::text,pricing.price_book_id,book.currency,
		       workflow.workflow_definition_id,metering.metering_definition_id,routing.route_policy_id,
		       product_version.definition->>'delivery_mode'
		FROM sku_versions version
		JOIN skus sku ON sku.id=version.sku_id AND sku.tenant_id=version.tenant_id
		JOIN products product ON product.id=sku.product_id AND product.tenant_id=version.tenant_id
		JOIN product_versions product_version ON product_version.id=version.product_version_id AND product_version.tenant_id=version.tenant_id
		JOIN publications publication ON publication.product_version_id=version.product_version_id AND publication.product_id=product.id AND publication.tenant_id=version.tenant_id
		JOIN product_pricing_bindings pricing ON pricing.product_version_id=version.product_version_id AND pricing.tenant_id=version.tenant_id
		JOIN price_books book ON book.id=pricing.price_book_id AND book.tenant_id=version.tenant_id
		JOIN product_workflow_bindings workflow ON workflow.product_version_id=version.product_version_id AND workflow.tenant_id=version.tenant_id
		JOIN product_metering_bindings metering ON metering.product_version_id=version.product_version_id AND metering.tenant_id=version.tenant_id
		JOIN product_routing_bindings routing ON routing.product_version_id=version.product_version_id AND routing.tenant_id=version.tenant_id
		WHERE version.tenant_id=$1 AND version.id=$2 AND product.status='published' AND sku.status='published'
		  AND publication.status='published' AND book.status='active'
		ORDER BY publication.published_at DESC LIMIT 1`, tenantID, skuVersionID).Scan(
		&productVersionID, &bindingsJSON, &priceBookID, &currency, &workflowID, &meteringID, &routingID, &deliveryMode,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return saleBinding{}, platform.Invalid("sku_not_orderable", "SKU version is not part of an active publication")
		}
		return saleBinding{}, err
	}
	var skuVersion catalog.SKUVersion
	if err = json.Unmarshal([]byte(bindingsJSON), &skuVersion); err != nil {
		return saleBinding{}, err
	}
	bindings := orderdomain.VersionBindings{
		ProductVersionID: productVersionID, SKUVersionID: skuVersionID,
		PricingVersionID: priceBookID, WorkflowVersionID: workflowID, RoutingVersionID: routingID,
	}
	if err = bindings.Validate(); err != nil {
		return saleBinding{}, err
	}
	if skuVersion.ProductVersionID != productVersionID || skuVersion.PricingVersionID != priceBookID || skuVersion.WorkflowVersionID != workflowID || skuVersion.RoutingVersionID != routingID {
		return saleBinding{}, platform.Invalid("version_binding_mismatch", "SKU version bindings do not match the published product version")
	}
	book := pricing.PriceBook{ID: priceBookID, TenantID: tenantID, Currency: strings.ToUpper(currency), Version: 1, Rules: []pricing.Rule{}}
	rows, err := tx.Query(ctx, `SELECT definition FROM price_rules WHERE tenant_id=$1 AND price_book_id=$2 ORDER BY priority,id`, tenantID, priceBookID)
	if err != nil {
		return saleBinding{}, err
	}
	for rows.Next() {
		var definition []byte
		if err = rows.Scan(&definition); err != nil {
			rows.Close()
			return saleBinding{}, err
		}
		var rule pricing.Rule
		if err = json.Unmarshal(definition, &rule); err != nil {
			rows.Close()
			return saleBinding{}, err
		}
		book.Rules = append(book.Rules, rule)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return saleBinding{}, err
	}
	money, err := book.Calculate(quantity)
	if err != nil {
		return saleBinding{}, platform.Invalid("pricing_failed", err.Error())
	}
	return saleBinding{Bindings: bindings, MeteringID: meteringID, DeliveryMode: deliveryMode, Entitlements: skuVersion.Entitlements, Currency: money.Currency, AmountMinor: money.Minor}, nil
}

func scanQuote(row rowScanner) (orderdomain.Quote, error) {
	var item orderdomain.Quote
	err := row.Scan(&item.ID, &item.TenantID, &item.DealID, &item.CanonicalDealID, &item.CustomerID, &item.Status, &item.Version, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	item.Versions = []orderdomain.QuoteVersion{}
	return item, err
}

func scanQuoteVersion(row rowScanner) (orderdomain.QuoteVersion, error) {
	var item orderdomain.QuoteVersion
	err := row.Scan(&item.ID, &item.TenantID, &item.QuoteID, &item.Version, &item.Currency, &item.AmountMinor, &item.ValidUntil, &item.CreatedBy, &item.CreatedAt)
	item.Currency = strings.TrimSpace(item.Currency)
	item.Items = []orderdomain.QuoteItem{}
	return item, err
}

func loadQuoteItems(ctx context.Context, tx pgx.Tx, tenantID, quoteVersionID string) ([]orderdomain.QuoteItem, error) {
	rows, err := tx.Query(ctx, `SELECT id,tenant_id,quote_version_id,product_version_id,sku_version_id,workflow_version_id,pricing_version_id,routing_version_id,quantity,unit_amount_minor,amount_minor,input FROM quote_version_items WHERE tenant_id=$1 AND quote_version_id=$2 ORDER BY id`, tenantID, quoteVersionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []orderdomain.QuoteItem{}
	for rows.Next() {
		var item orderdomain.QuoteItem
		var input []byte
		if err = rows.Scan(&item.ID, &item.TenantID, &item.QuoteVersionID, &item.Bindings.ProductVersionID, &item.Bindings.SKUVersionID, &item.Bindings.WorkflowVersionID, &item.Bindings.PricingVersionID, &item.Bindings.RoutingVersionID, &item.Quantity, &item.UnitAmountMinor, &item.AmountMinor, &input); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(input, &item.Input); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func loadQuoteVersions(ctx context.Context, tx pgx.Tx, tenantID, quoteID string) ([]orderdomain.QuoteVersion, error) {
	rows, err := tx.Query(ctx, `SELECT id,tenant_id,quote_id,version,currency,amount_minor,valid_until,created_by,created_at FROM quote_versions WHERE tenant_id=$1 AND quote_id=$2 ORDER BY version,id`, tenantID, quoteID)
	if err != nil {
		return nil, err
	}
	versions := []orderdomain.QuoteVersion{}
	for rows.Next() {
		item, scanErr := scanQuoteVersion(rows)
		if scanErr != nil {
			rows.Close()
			return nil, scanErr
		}
		versions = append(versions, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for index := range versions {
		versions[index].Items, err = loadQuoteItems(ctx, tx, tenantID, versions[index].ID)
		if err != nil {
			return nil, err
		}
	}
	return versions, nil
}

func getQuote(ctx context.Context, tx pgx.Tx, tenantID, quoteID string, lock bool) (orderdomain.Quote, error) {
	query := `SELECT id,tenant_id,deal_id,COALESCE(growth_deal_id::text,''),customer_id,status,version,created_by,created_at,updated_at FROM quotes WHERE tenant_id=$1 AND id=$2`
	if lock {
		query += ` FOR UPDATE`
	}
	item, err := scanQuote(tx.QueryRow(ctx, query, tenantID, quoteID))
	if err != nil {
		return item, err
	}
	item.Versions, err = loadQuoteVersions(ctx, tx, tenantID, quoteID)
	return item, err
}

func (s *Store) CreateQuote(ctx context.Context, scope tenancy.Scope, input application.QuoteInput, key string) (orderdomain.Quote, error) {
	input.DealID, input.CustomerID, input.Currency = strings.TrimSpace(input.DealID), strings.TrimSpace(input.CustomerID), strings.ToUpper(strings.TrimSpace(input.Currency))
	if input.DealID == "" || input.CustomerID == "" || len(input.Currency) != 3 || len(input.Items) == 0 {
		return orderdomain.Quote{}, platform.Invalid("quote_incomplete", "deal, customer, currency, and at least one item are required")
	}
	if input.ValidUntil.IsZero() || !input.ValidUntil.After(time.Now().UTC()) {
		return orderdomain.Quote{}, platform.Invalid("invalid_quote_expiry", "quote expiry must be in the future")
	}
	return runCommand(ctx, s, scope, key, "quote.create", func(tx pgx.Tx) (orderdomain.Quote, string, error) {
		canonicalDealID := ""
		var dealCustomerID, dealStatus, dealCurrency string
		dealErr := tx.QueryRow(ctx, `SELECT id::text,customer_id,status,currency FROM deals WHERE tenant_id=$1 AND id::text=$2`, scope.TenantID, input.DealID).Scan(&canonicalDealID, &dealCustomerID, &dealStatus, &dealCurrency)
		if dealErr != nil && dealErr != pgx.ErrNoRows {
			return orderdomain.Quote{}, "", dealErr
		}
		if dealErr == nil {
			if dealCustomerID != input.CustomerID {
				return orderdomain.Quote{}, "", platform.Invalid("deal_customer_mismatch", "canonical deal and quote customer must match")
			}
			if dealStatus != "open" && dealStatus != "proposal" {
				return orderdomain.Quote{}, "", platform.Invalid("deal_closed", "closed canonical deal cannot accept a quote")
			}
			if strings.TrimSpace(dealCurrency) != input.Currency {
				return orderdomain.Quote{}, "", platform.Invalid("deal_currency_mismatch", "canonical deal and quote currency must match")
			}
		}
		bindings := make([]saleBinding, len(input.Items))
		var total int64
		for index, itemInput := range input.Items {
			binding, err := loadSaleBinding(ctx, tx, scope.TenantID, itemInput.SKUVersionID, itemInput.Quantity)
			if err != nil {
				return orderdomain.Quote{}, "", err
			}
			if binding.Currency != input.Currency {
				return orderdomain.Quote{}, "", platform.Invalid("currency_mismatch", "all quote items must use the quote currency")
			}
			if binding.AmountMinor > math.MaxInt64-total {
				return orderdomain.Quote{}, "", platform.Invalid("amount_overflow", "quote amount exceeds the supported range")
			}
			total += binding.AmountMinor
			bindings[index] = binding
		}
		quote, err := scanQuote(tx.QueryRow(ctx, `INSERT INTO quotes (tenant_id,deal_id,growth_deal_id,customer_id,status,version,created_by) VALUES($1,$2,$3,$4,'draft',1,$5) RETURNING id,tenant_id,deal_id,COALESCE(growth_deal_id::text,''),customer_id,status,version,created_by,created_at,updated_at`, scope.TenantID, input.DealID, nullableUUID(canonicalDealID), input.CustomerID, scope.ActorID))
		if err != nil {
			return quote, "", err
		}
		version, err := scanQuoteVersion(tx.QueryRow(ctx, `INSERT INTO quote_versions (tenant_id,quote_id,version,currency,amount_minor,valid_until,created_by) VALUES($1,$2,1,$3,$4,$5,$6) RETURNING id,tenant_id,quote_id,version,currency,amount_minor,valid_until,created_by,created_at`, scope.TenantID, quote.ID, input.Currency, total, input.ValidUntil.UTC(), scope.ActorID))
		if err != nil {
			return quote, "", err
		}
		for index, itemInput := range input.Items {
			binding := bindings[index]
			itemInputJSON, marshalErr := json.Marshal(itemInput.Input)
			if marshalErr != nil {
				return quote, "", marshalErr
			}
			unitAmount := binding.AmountMinor
			if itemInput.Quantity > 0 {
				unitAmount = binding.AmountMinor / itemInput.Quantity
			}
			var item orderdomain.QuoteItem
			item.Bindings = binding.Bindings
			err = tx.QueryRow(ctx, `INSERT INTO quote_version_items (tenant_id,quote_version_id,product_version_id,sku_version_id,workflow_version_id,pricing_version_id,routing_version_id,quantity,unit_amount_minor,amount_minor,input) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id,tenant_id,quote_version_id,quantity,unit_amount_minor,amount_minor`, scope.TenantID, version.ID, binding.Bindings.ProductVersionID, binding.Bindings.SKUVersionID, binding.Bindings.WorkflowVersionID, binding.Bindings.PricingVersionID, binding.Bindings.RoutingVersionID, itemInput.Quantity, unitAmount, binding.AmountMinor, itemInputJSON).Scan(&item.ID, &item.TenantID, &item.QuoteVersionID, &item.Quantity, &item.UnitAmountMinor, &item.AmountMinor)
			if err != nil {
				return quote, "", err
			}
			item.Input = itemInput.Input
			version.Items = append(version.Items, item)
		}
		quote.Versions = []orderdomain.QuoteVersion{version}
		if err = appendAudit(ctx, tx, scope, "quote.create", "quote", quote.ID, key, map[string]any{"quote_version_id": version.ID, "growth_deal_id": canonicalDealID, "amount_minor": total, "currency": input.Currency}); err != nil {
			return quote, "", err
		}
		if err = appendEvent(ctx, tx, scope, "quote", quote.ID, "quote.created", quote.Version, map[string]any{"quote_version_id": version.ID, "growth_deal_id": canonicalDealID, "amount_minor": total, "currency": input.Currency}); err != nil {
			return quote, "", err
		}
		return quote, quote.ID, nil
	})
}

func (s *Store) ListQuotes(ctx context.Context, scope tenancy.Scope) ([]orderdomain.Quote, error) {
	return runTenantRead(ctx, s, scope.TenantID, func(tx pgx.Tx) ([]orderdomain.Quote, error) {
		rows, err := tx.Query(ctx, `SELECT id,tenant_id,deal_id,COALESCE(growth_deal_id::text,''),customer_id,status,version,created_by,created_at,updated_at FROM quotes WHERE tenant_id=$1 ORDER BY updated_at DESC,id LIMIT 200`, scope.TenantID)
		if err != nil {
			return nil, err
		}
		items := []orderdomain.Quote{}
		for rows.Next() {
			item, scanErr := scanQuote(rows)
			if scanErr != nil {
				rows.Close()
				return nil, scanErr
			}
			items = append(items, item)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
		for index := range items {
			items[index].Versions, err = loadQuoteVersions(ctx, tx, scope.TenantID, items[index].ID)
			if err != nil {
				return nil, err
			}
		}
		return items, nil
	})
}

func (s *Store) GetQuote(ctx context.Context, scope tenancy.Scope, id string) (orderdomain.Quote, error) {
	return runTenantRead(ctx, s, scope.TenantID, func(tx pgx.Tx) (orderdomain.Quote, error) {
		return getQuote(ctx, tx, scope.TenantID, id, false)
	})
}

func (s *Store) TransitionQuote(ctx context.Context, scope tenancy.Scope, id, to, key string) (orderdomain.Quote, error) {
	return runCommand(ctx, s, scope, key, "quote.transition", func(tx pgx.Tx) (orderdomain.Quote, string, error) {
		item, err := getQuote(ctx, tx, scope.TenantID, id, true)
		if err != nil {
			return item, "", err
		}
		from := item.Status
		if to == "accepted" {
			if len(item.Versions) == 0 || !item.Versions[len(item.Versions)-1].ValidUntil.After(time.Now().UTC()) {
				return item, "", platform.Invalid("quote_expired", "expired quote cannot be accepted")
			}
		}
		if err = item.Transition(to); err != nil {
			return item, "", err
		}
		if _, err = tx.Exec(ctx, `UPDATE quotes SET status=$3,version=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, id, item.Status, item.Version); err != nil {
			return item, "", err
		}
		metadata := map[string]any{"from": from, "to": to}
		if err = appendAudit(ctx, tx, scope, "quote.transition", "quote", id, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "quote", id, "quote.transitioned", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, id, nil
	})
}

func scanOrder(row rowScanner) (orderdomain.Order, error) {
	var item orderdomain.Order
	var bindings []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.CustomerID, &item.QuoteVersionID, &item.Status, &item.Currency, &item.AmountMinor, &item.IdempotencyKey, &bindings, &item.Version, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	if err == nil {
		err = json.Unmarshal(bindings, &item.Bindings)
	}
	item.Currency = strings.TrimSpace(item.Currency)
	return item, err
}

func loadOrderItems(ctx context.Context, tx pgx.Tx, tenantID, orderID string) ([]orderdomain.OrderItem, error) {
	rows, err := tx.Query(ctx, `SELECT id,tenant_id,order_id,product_version_id,sku_version_id,workflow_version_id,pricing_version_id,routing_version_id,quantity,unit_amount_minor,amount_minor,input FROM order_items WHERE tenant_id=$1 AND order_id=$2 ORDER BY id`, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []orderdomain.OrderItem{}
	for rows.Next() {
		var item orderdomain.OrderItem
		var input []byte
		if err = rows.Scan(&item.ID, &item.TenantID, &item.OrderID, &item.Bindings.ProductVersionID, &item.Bindings.SKUVersionID, &item.Bindings.WorkflowVersionID, &item.Bindings.PricingVersionID, &item.Bindings.RoutingVersionID, &item.Quantity, &item.UnitAmountMinor, &item.AmountMinor, &input); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(input, &item.Input); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func loadSubscriptions(ctx context.Context, tx pgx.Tx, tenantID, orderID string) ([]orderdomain.Subscription, error) {
	rows, err := tx.Query(ctx, `SELECT id,tenant_id,order_id,order_item_id,customer_id,sku_version_id,status,starts_at,ends_at,created_at,updated_at FROM subscriptions WHERE tenant_id=$1 AND order_id=$2 ORDER BY id`, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []orderdomain.Subscription{}
	for rows.Next() {
		var item orderdomain.Subscription
		if err = rows.Scan(&item.ID, &item.TenantID, &item.OrderID, &item.OrderItemID, &item.CustomerID, &item.SKUVersionID, &item.Status, &item.StartsAt, &item.EndsAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func loadEntitlements(ctx context.Context, tx pgx.Tx, tenantID, orderID string) ([]orderdomain.Entitlement, error) {
	rows, err := tx.Query(ctx, `SELECT id,tenant_id,order_id,order_item_id,COALESCE(subscription_id::text,''),entitlement_key,entitlement_value,status,starts_at,ends_at,created_at,updated_at FROM entitlements WHERE tenant_id=$1 AND order_id=$2 ORDER BY entitlement_key,id`, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []orderdomain.Entitlement{}
	for rows.Next() {
		var item orderdomain.Entitlement
		var value []byte
		if err = rows.Scan(&item.ID, &item.TenantID, &item.OrderID, &item.OrderItemID, &item.SubscriptionID, &item.Key, &value, &item.Status, &item.StartsAt, &item.EndsAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(value, &item.Value); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanExecution(row rowScanner) (orderdomain.ExecutionOrder, error) {
	var item orderdomain.ExecutionOrder
	var input, output, errorJSON []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.OrderID, &item.OrderItemID, &item.Bindings.ProductVersionID, &item.Bindings.SKUVersionID, &item.Bindings.WorkflowVersionID, &item.Bindings.PricingVersionID, &item.Bindings.RoutingVersionID, &item.ProviderEndpointID, &item.Status, &item.Attempt, &item.IdempotencyKey, &item.ExternalID, &input, &output, &errorJSON, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	if err = json.Unmarshal(input, &item.Input); err != nil {
		return item, err
	}
	if err = json.Unmarshal(output, &item.Output); err != nil {
		return item, err
	}
	err = json.Unmarshal(errorJSON, &item.Error)
	return item, err
}

func loadExecutions(ctx context.Context, tx pgx.Tx, tenantID, orderID string) ([]orderdomain.ExecutionOrder, error) {
	rows, err := tx.Query(ctx, `SELECT id,tenant_id,order_id,order_item_id,product_version_id,sku_version_id,workflow_version_id,pricing_version_id,routing_version_id,COALESCE(provider_endpoint_id::text,''),status,attempt,idempotency_key,COALESCE(external_id,''),input,output,error,created_by,created_at,updated_at FROM execution_orders WHERE tenant_id=$1 AND order_id=$2 ORDER BY created_at,id`, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []orderdomain.ExecutionOrder{}
	for rows.Next() {
		item, scanErr := scanExecution(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanDelivery(row rowScanner) (orderdomain.DeliveryProject, error) {
	var item orderdomain.DeliveryProject
	var artifacts []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.OrderID, &item.OrderItemID, &item.ExecutionOrderID, &item.Mode, &item.Status, &item.Assignee, &artifacts, &item.CreatedAt, &item.UpdatedAt)
	if err == nil {
		err = json.Unmarshal(artifacts, &item.Artifacts)
	}
	return item, err
}

func loadDeliveries(ctx context.Context, tx pgx.Tx, tenantID, orderID string) ([]orderdomain.DeliveryProject, error) {
	rows, err := tx.Query(ctx, `SELECT id,tenant_id,order_id,order_item_id,execution_order_id,mode,status,COALESCE(assignee,''),artifacts,created_at,updated_at FROM delivery_projects WHERE tenant_id=$1 AND order_id=$2 ORDER BY created_at,id`, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []orderdomain.DeliveryProject{}
	for rows.Next() {
		item, scanErr := scanDelivery(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func loadUsage(ctx context.Context, tx pgx.Tx, tenantID, orderID string) ([]orderdomain.UsageRecord, error) {
	rows, err := tx.Query(ctx, `SELECT id,tenant_id,order_id,COALESCE(order_item_id::text,''),COALESCE(execution_order_id::text,''),meter_id,quantity,COALESCE(idempotency_key,''),created_by,occurred_at,created_at FROM usage_records WHERE tenant_id=$1 AND order_id=$2 ORDER BY occurred_at,id`, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []orderdomain.UsageRecord{}
	for rows.Next() {
		var item orderdomain.UsageRecord
		if err = rows.Scan(&item.ID, &item.TenantID, &item.OrderID, &item.OrderItemID, &item.ExecutionOrderID, &item.MeterID, &item.Quantity, &item.IdempotencyKey, &item.CreatedBy, &item.OccurredAt, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func loadProviderCosts(ctx context.Context, tx pgx.Tx, tenantID, orderID string) ([]orderdomain.ProviderCost, error) {
	rows, err := tx.Query(ctx, `SELECT id,tenant_id,order_id,COALESCE(order_item_id::text,''),COALESCE(execution_order_id::text,''),provider_endpoint_id,currency,amount_minor,COALESCE(idempotency_key,''),created_by,created_at FROM provider_costs WHERE tenant_id=$1 AND order_id=$2 ORDER BY created_at,id`, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []orderdomain.ProviderCost{}
	for rows.Next() {
		var item orderdomain.ProviderCost
		if err = rows.Scan(&item.ID, &item.TenantID, &item.OrderID, &item.OrderItemID, &item.ExecutionOrderID, &item.ProviderEndpointID, &item.Currency, &item.AmountMinor, &item.IdempotencyKey, &item.CreatedBy, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Currency = strings.TrimSpace(item.Currency)
		items = append(items, item)
	}
	return items, rows.Err()
}

func loadCustomerCharges(ctx context.Context, tx pgx.Tx, tenantID, orderID string) ([]orderdomain.CustomerCharge, error) {
	rows, err := tx.Query(ctx, `SELECT id,tenant_id,order_id,COALESCE(order_item_id::text,''),COALESCE(execution_order_id::text,''),price_book_id,currency,amount_minor,status,COALESCE(idempotency_key,''),created_by,created_at FROM customer_charges WHERE tenant_id=$1 AND order_id=$2 ORDER BY created_at,id`, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []orderdomain.CustomerCharge{}
	for rows.Next() {
		var item orderdomain.CustomerCharge
		if err = rows.Scan(&item.ID, &item.TenantID, &item.OrderID, &item.OrderItemID, &item.ExecutionOrderID, &item.PriceBookID, &item.Currency, &item.AmountMinor, &item.Status, &item.IdempotencyKey, &item.CreatedBy, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Currency = strings.TrimSpace(item.Currency)
		items = append(items, item)
	}
	return items, rows.Err()
}

func getOrder(ctx context.Context, tx pgx.Tx, tenantID, orderID string, lock bool) (orderdomain.Order, error) {
	query := `SELECT id,tenant_id,customer_id,COALESCE(quote_version_id::text,''),status,currency,amount_minor,idempotency_key,version_bindings,version,created_by,created_at,updated_at FROM orders WHERE tenant_id=$1 AND id=$2`
	if lock {
		query += ` FOR UPDATE`
	}
	item, err := scanOrder(tx.QueryRow(ctx, query, tenantID, orderID))
	if err != nil {
		return item, err
	}
	if item.Items, err = loadOrderItems(ctx, tx, tenantID, orderID); err != nil {
		return item, err
	}
	if item.Subscriptions, err = loadSubscriptions(ctx, tx, tenantID, orderID); err != nil {
		return item, err
	}
	if item.Entitlements, err = loadEntitlements(ctx, tx, tenantID, orderID); err != nil {
		return item, err
	}
	if item.Executions, err = loadExecutions(ctx, tx, tenantID, orderID); err != nil {
		return item, err
	}
	if item.Deliveries, err = loadDeliveries(ctx, tx, tenantID, orderID); err != nil {
		return item, err
	}
	if item.Usage, err = loadUsage(ctx, tx, tenantID, orderID); err != nil {
		return item, err
	}
	if item.ProviderCosts, err = loadProviderCosts(ctx, tx, tenantID, orderID); err != nil {
		return item, err
	}
	item.CustomerCharges, err = loadCustomerCharges(ctx, tx, tenantID, orderID)
	return item, err
}

func (s *Store) CreateOrder(ctx context.Context, scope tenancy.Scope, quoteVersionID, key string) (orderdomain.Order, error) {
	return runCommand(ctx, s, scope, key, "order.create", func(tx pgx.Tx) (orderdomain.Order, string, error) {
		var quoteID, customerID, quoteStatus, currency string
		var amountMinor int64
		var validUntil time.Time
		var quoteVersion int
		if err := tx.QueryRow(ctx, `SELECT quote.id,quote.customer_id,quote.status,version.currency,version.amount_minor,version.valid_until,version.version FROM quote_versions version JOIN quotes quote ON quote.id=version.quote_id AND quote.tenant_id=version.tenant_id WHERE version.tenant_id=$1 AND version.id=$2 FOR UPDATE OF quote`, scope.TenantID, quoteVersionID).Scan(&quoteID, &customerID, &quoteStatus, &currency, &amountMinor, &validUntil, &quoteVersion); err != nil {
			return orderdomain.Order{}, "", err
		}
		if quoteStatus != "accepted" || !validUntil.After(time.Now().UTC()) {
			return orderdomain.Order{}, "", platform.Invalid("quote_not_orderable", "quote must be accepted and unexpired")
		}
		var latestVersion int
		if err := tx.QueryRow(ctx, `SELECT max(version) FROM quote_versions WHERE tenant_id=$1 AND quote_id=$2`, scope.TenantID, quoteID).Scan(&latestVersion); err != nil {
			return orderdomain.Order{}, "", err
		}
		if quoteVersion != latestVersion {
			return orderdomain.Order{}, "", platform.Invalid("quote_version_stale", "order must use the latest accepted quote version")
		}
		quoteItems, err := loadQuoteItems(ctx, tx, scope.TenantID, quoteVersionID)
		if err != nil || len(quoteItems) == 0 {
			if err == nil {
				err = platform.Invalid("quote_incomplete", "quote version has no items")
			}
			return orderdomain.Order{}, "", err
		}
		bindingsJSON, _ := json.Marshal(quoteItems[0].Bindings)
		item, err := scanOrder(tx.QueryRow(ctx, `INSERT INTO orders (tenant_id,customer_id,quote_version_id,status,currency,amount_minor,idempotency_key,version_bindings,version,created_by) VALUES($1,$2,$3,'created',$4,$5,$6,$7,1,$8) RETURNING id,tenant_id,customer_id,quote_version_id,status,currency,amount_minor,idempotency_key,version_bindings,version,created_by,created_at,updated_at`, scope.TenantID, customerID, quoteVersionID, strings.TrimSpace(currency), amountMinor, key, bindingsJSON, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		for _, quoteItem := range quoteItems {
			inputJSON, marshalErr := json.Marshal(quoteItem.Input)
			if marshalErr != nil {
				return item, "", marshalErr
			}
			var orderItem orderdomain.OrderItem
			orderItem.Bindings = quoteItem.Bindings
			if err = tx.QueryRow(ctx, `INSERT INTO order_items (tenant_id,order_id,quote_version_item_id,product_version_id,sku_version_id,workflow_version_id,pricing_version_id,routing_version_id,quantity,unit_amount_minor,amount_minor,input) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id,tenant_id,order_id,quantity,unit_amount_minor,amount_minor`, scope.TenantID, item.ID, quoteItem.ID, quoteItem.Bindings.ProductVersionID, quoteItem.Bindings.SKUVersionID, quoteItem.Bindings.WorkflowVersionID, quoteItem.Bindings.PricingVersionID, quoteItem.Bindings.RoutingVersionID, quoteItem.Quantity, quoteItem.UnitAmountMinor, quoteItem.AmountMinor, inputJSON).Scan(&orderItem.ID, &orderItem.TenantID, &orderItem.OrderID, &orderItem.Quantity, &orderItem.UnitAmountMinor, &orderItem.AmountMinor); err != nil {
				return item, "", err
			}
			orderItem.Input = quoteItem.Input
			item.Items = append(item.Items, orderItem)
		}
		if err = appendAudit(ctx, tx, scope, "order.create", "order", item.ID, key, map[string]any{"quote_version_id": quoteVersionID, "amount_minor": amountMinor, "currency": strings.TrimSpace(currency)}); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "order", item.ID, "order.created", item.Version, map[string]any{"quote_version_id": quoteVersionID, "amount_minor": amountMinor, "currency": strings.TrimSpace(currency)}); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) ListOrders(ctx context.Context, scope tenancy.Scope) ([]orderdomain.Order, error) {
	return runTenantRead(ctx, s, scope.TenantID, func(tx pgx.Tx) ([]orderdomain.Order, error) {
		rows, err := tx.Query(ctx, `SELECT id,tenant_id,customer_id,COALESCE(quote_version_id::text,''),status,currency,amount_minor,idempotency_key,version_bindings,version,created_by,created_at,updated_at FROM orders WHERE tenant_id=$1 ORDER BY updated_at DESC,id LIMIT 200`, scope.TenantID)
		if err != nil {
			return nil, err
		}
		items := []orderdomain.Order{}
		for rows.Next() {
			item, scanErr := scanOrder(rows)
			if scanErr != nil {
				rows.Close()
				return nil, scanErr
			}
			items = append(items, item)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
		for index := range items {
			items[index], err = getOrder(ctx, tx, scope.TenantID, items[index].ID, false)
			if err != nil {
				return nil, err
			}
		}
		return items, nil
	})
}

func (s *Store) GetOrder(ctx context.Context, scope tenancy.Scope, id string) (orderdomain.Order, error) {
	return runTenantRead(ctx, s, scope.TenantID, func(tx pgx.Tx) (orderdomain.Order, error) {
		return getOrder(ctx, tx, scope.TenantID, id, false)
	})
}

func createFulfillment(ctx context.Context, tx pgx.Tx, scope tenancy.Scope, item orderdomain.Order, key string) error {
	for _, orderItem := range item.Items {
		var skuBindings, productDefinition []byte
		if err := tx.QueryRow(ctx, `SELECT sku_version.bindings,product_version.definition FROM sku_versions sku_version JOIN product_versions product_version ON product_version.id=sku_version.product_version_id AND product_version.tenant_id=sku_version.tenant_id WHERE sku_version.tenant_id=$1 AND sku_version.id=$2 AND product_version.id=$3`, scope.TenantID, orderItem.Bindings.SKUVersionID, orderItem.Bindings.ProductVersionID).Scan(&skuBindings, &productDefinition); err != nil {
			return err
		}
		var skuVersion catalog.SKUVersion
		var productVersion catalog.ProductVersion
		if err := json.Unmarshal(skuBindings, &skuVersion); err != nil {
			return err
		}
		if err := json.Unmarshal(productDefinition, &productVersion); err != nil {
			return err
		}
		var subscriptionID string
		if err := tx.QueryRow(ctx, `INSERT INTO subscriptions (tenant_id,order_id,order_item_id,customer_id,sku_version_id,status) VALUES($1,$2,$3,$4,$5,'pending') RETURNING id`, scope.TenantID, item.ID, orderItem.ID, item.CustomerID, orderItem.Bindings.SKUVersionID).Scan(&subscriptionID); err != nil {
			return err
		}
		entitlementKeys := make([]string, 0, len(skuVersion.Entitlements))
		for entitlementKey := range skuVersion.Entitlements {
			entitlementKeys = append(entitlementKeys, entitlementKey)
		}
		sort.Strings(entitlementKeys)
		for _, entitlementKey := range entitlementKeys {
			valueJSON, err := json.Marshal(skuVersion.Entitlements[entitlementKey])
			if err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `INSERT INTO entitlements (tenant_id,order_id,order_item_id,subscription_id,entitlement_key,entitlement_value,status) VALUES($1,$2,$3,$4,$5,$6,'pending')`, scope.TenantID, item.ID, orderItem.ID, subscriptionID, entitlementKey, valueJSON); err != nil {
				return err
			}
		}
		inputJSON, _ := json.Marshal(orderItem.Input)
		var executionID string
		executionKey := key + ":" + orderItem.ID
		if err := tx.QueryRow(ctx, `INSERT INTO execution_orders (tenant_id,order_id,order_item_id,product_version_id,sku_version_id,workflow_version_id,pricing_version_id,routing_version_id,status,idempotency_key,input,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,'created',$9,$10,$11) RETURNING id`, scope.TenantID, item.ID, orderItem.ID, orderItem.Bindings.ProductVersionID, orderItem.Bindings.SKUVersionID, orderItem.Bindings.WorkflowVersionID, orderItem.Bindings.PricingVersionID, orderItem.Bindings.RoutingVersionID, executionKey, inputJSON, scope.ActorID).Scan(&executionID); err != nil {
			return err
		}
		var deliveryID string
		if err := tx.QueryRow(ctx, `INSERT INTO delivery_projects (tenant_id,order_id,order_item_id,execution_order_id,mode,status) VALUES($1,$2,$3,$4,$5,'created') RETURNING id`, scope.TenantID, item.ID, orderItem.ID, executionID, productVersion.DeliveryMode).Scan(&deliveryID); err != nil {
			return err
		}
		if err := appendAudit(ctx, tx, scope, "subscription.create", "subscription", subscriptionID, key, map[string]any{"order_id": item.ID, "order_item_id": orderItem.ID}); err != nil {
			return err
		}
		if err := appendAudit(ctx, tx, scope, "execution.create", "execution_order", executionID, key, map[string]any{"order_id": item.ID, "order_item_id": orderItem.ID}); err != nil {
			return err
		}
		if err := appendAudit(ctx, tx, scope, "delivery.create", "delivery_project", deliveryID, key, map[string]any{"order_id": item.ID, "mode": productVersion.DeliveryMode}); err != nil {
			return err
		}
		if err := appendEvent(ctx, tx, scope, "execution_order", executionID, "execution.created", 1, map[string]any{"order_id": item.ID, "order_item_id": orderItem.ID}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) TransitionOrder(ctx context.Context, scope tenancy.Scope, id, to, key string) (orderdomain.Order, error) {
	return runCommand(ctx, s, scope, key, "order.transition", func(tx pgx.Tx) (orderdomain.Order, string, error) {
		item, err := getOrder(ctx, tx, scope.TenantID, id, true)
		if err != nil {
			return item, "", err
		}
		from := item.Status
		if err = item.Transition(to); err != nil {
			return item, "", err
		}
		if to == "paid" && item.AmountMinor > 0 {
			var heldMinor int64
			if err = tx.QueryRow(ctx, `SELECT COALESCE(sum(amount_minor-captured_minor-released_minor),0) FROM holds WHERE tenant_id=$1 AND order_id=$2 AND status IN ('active','partially_captured')`, scope.TenantID, id).Scan(&heldMinor); err != nil {
				return item, "", err
			}
			if heldMinor < item.AmountMinor {
				return item, "", platform.Invalid("payment_hold_required", "order payment requires sufficient active held funds")
			}
		}
		if to == "provisioning" {
			if err = createFulfillment(ctx, tx, scope, item, key); err != nil {
				return item, "", err
			}
		}
		if to == "active" {
			var incomplete int
			if err = tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM execution_orders WHERE tenant_id=$1 AND order_id=$2 AND status NOT IN ('succeeded','reconciling','settled')) + (SELECT count(*) FROM delivery_projects WHERE tenant_id=$1 AND order_id=$2 AND status<>'completed')`, scope.TenantID, id).Scan(&incomplete); err != nil {
				return item, "", err
			}
			if incomplete > 0 {
				return item, "", platform.Invalid("fulfillment_incomplete", "executions and deliveries must complete before activation")
			}
			if _, err = tx.Exec(ctx, `UPDATE subscriptions SET status='active',starts_at=COALESCE(starts_at,now()),updated_at=now() WHERE tenant_id=$1 AND order_id=$2 AND status='pending'`, scope.TenantID, id); err != nil {
				return item, "", err
			}
			if _, err = tx.Exec(ctx, `UPDATE entitlements SET status='active',starts_at=COALESCE(starts_at,now()),updated_at=now() WHERE tenant_id=$1 AND order_id=$2 AND status='pending'`, scope.TenantID, id); err != nil {
				return item, "", err
			}
		}
		if to == "completed" {
			var unsettled int
			if err = tx.QueryRow(ctx, `SELECT count(*) FROM execution_orders WHERE tenant_id=$1 AND order_id=$2 AND status<>'settled'`, scope.TenantID, id).Scan(&unsettled); err != nil {
				return item, "", err
			}
			if unsettled > 0 {
				return item, "", platform.Invalid("execution_unsettled", "all executions must be settled before order completion")
			}
		}
		if _, err = tx.Exec(ctx, `UPDATE orders SET status=$3,version=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, id, item.Status, item.Version); err != nil {
			return item, "", err
		}
		metadata := map[string]any{"from": from, "to": to}
		if err = appendAudit(ctx, tx, scope, "order.transition", "order", id, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "order", id, "order.transitioned", item.Version, metadata); err != nil {
			return item, "", err
		}
		item, err = getOrder(ctx, tx, scope.TenantID, id, false)
		return item, id, err
	})
}

func getExecution(ctx context.Context, tx pgx.Tx, tenantID, id string, lock bool) (orderdomain.ExecutionOrder, error) {
	query := `SELECT id,tenant_id,order_id,order_item_id,product_version_id,sku_version_id,workflow_version_id,pricing_version_id,routing_version_id,COALESCE(provider_endpoint_id::text,''),status,attempt,idempotency_key,COALESCE(external_id,''),input,output,error,created_by,created_at,updated_at FROM execution_orders WHERE tenant_id=$1 AND id=$2`
	if lock {
		query += ` FOR UPDATE`
	}
	return scanExecution(tx.QueryRow(ctx, query, tenantID, id))
}

func (s *Store) TransitionExecution(ctx context.Context, scope tenancy.Scope, id string, input application.ExecutionTransitionInput, key string) (orderdomain.ExecutionOrder, error) {
	return runCommand(ctx, s, scope, key, "execution.transition", func(tx pgx.Tx) (orderdomain.ExecutionOrder, string, error) {
		item, err := getExecution(ctx, tx, scope.TenantID, id, true)
		if err != nil {
			return item, "", err
		}
		from := item.Status
		if input.ProviderEndpointID != "" {
			var healthy bool
			if err = tx.QueryRow(ctx, `SELECT status='healthy' FROM provider_endpoints WHERE tenant_id=$1 AND id=$2`, scope.TenantID, input.ProviderEndpointID).Scan(&healthy); err != nil {
				return item, "", err
			}
			if !healthy {
				return item, "", platform.Invalid("provider_unavailable", "provider endpoint must be healthy")
			}
			item.ProviderEndpointID = input.ProviderEndpointID
		}
		if input.To == "reserved" && item.ProviderEndpointID == "" {
			return item, "", platform.Invalid("provider_required", "provider endpoint is required before reservation")
		}
		if input.To == "settled" {
			var unpostedCharges, missingPayables int
			if err = tx.QueryRow(ctx, `
				SELECT
				  (SELECT count(*) FROM customer_charges charge WHERE charge.tenant_id=$1 AND charge.execution_order_id=$2 AND charge.status NOT IN ('posted','reversed'))
				  + CASE WHEN EXISTS(SELECT 1 FROM customer_charges charge WHERE charge.tenant_id=$1 AND charge.execution_order_id=$2) THEN 0 ELSE 1 END,
				  (SELECT count(*) FROM provider_costs cost LEFT JOIN provider_payables payable ON payable.tenant_id=cost.tenant_id AND payable.provider_cost_id=cost.id WHERE cost.tenant_id=$1 AND cost.execution_order_id=$2 AND payable.id IS NULL)`, scope.TenantID, id).Scan(&unpostedCharges, &missingPayables); err != nil {
				return item, "", err
			}
			if unpostedCharges > 0 || missingPayables > 0 {
				return item, "", platform.Invalid("financial_posting_incomplete", "customer charges and provider payables must be posted before execution settlement")
			}
		}
		if err = item.Transition(input.To); err != nil {
			return item, "", err
		}
		if input.To == "submitted" {
			item.Attempt++
		}
		if input.Output != nil {
			item.Output = input.Output
		}
		if input.Error != nil {
			item.Error = input.Error
		}
		if input.ExternalID != "" {
			item.ExternalID = input.ExternalID
		}
		outputJSON, _ := json.Marshal(item.Output)
		errorJSON, _ := json.Marshal(item.Error)
		if _, err = tx.Exec(ctx, `UPDATE execution_orders SET status=$3,attempt=$4,provider_endpoint_id=$5,external_id=$6,output=$7,error=$8,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, id, item.Status, item.Attempt, nullableUUID(item.ProviderEndpointID), nullableText(item.ExternalID), outputJSON, errorJSON); err != nil {
			return item, "", err
		}
		metadata := map[string]any{"from": from, "to": input.To, "attempt": item.Attempt}
		if err = appendAudit(ctx, tx, scope, "execution.transition", "execution_order", id, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "execution_order", id, "execution.transitioned", item.Attempt+1, metadata); err != nil {
			return item, "", err
		}
		return item, id, nil
	})
}

func (s *Store) TransitionDelivery(ctx context.Context, scope tenancy.Scope, id, to, key string) (orderdomain.DeliveryProject, error) {
	return runCommand(ctx, s, scope, key, "delivery.transition", func(tx pgx.Tx) (orderdomain.DeliveryProject, string, error) {
		item, err := scanDelivery(tx.QueryRow(ctx, `SELECT id,tenant_id,order_id,order_item_id,execution_order_id,mode,status,COALESCE(assignee,''),artifacts,created_at,updated_at FROM delivery_projects WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, id))
		if err != nil {
			return item, "", err
		}
		from := item.Status
		if to == "completed" {
			var executionStatus string
			if err = tx.QueryRow(ctx, `SELECT status FROM execution_orders WHERE tenant_id=$1 AND id=$2`, scope.TenantID, item.ExecutionOrderID).Scan(&executionStatus); err != nil {
				return item, "", err
			}
			if executionStatus != "succeeded" && executionStatus != "reconciling" && executionStatus != "settled" {
				return item, "", platform.Invalid("execution_incomplete", "execution must succeed before delivery completion")
			}
		}
		if err = item.Transition(to); err != nil {
			return item, "", err
		}
		if _, err = tx.Exec(ctx, `UPDATE delivery_projects SET status=$3,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, id, item.Status); err != nil {
			return item, "", err
		}
		metadata := map[string]any{"from": from, "to": to}
		if err = appendAudit(ctx, tx, scope, "delivery.transition", "delivery_project", id, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "delivery_project", id, "delivery.transitioned", 1, metadata); err != nil {
			return item, "", err
		}
		return item, id, nil
	})
}

func (s *Store) RecordUsage(ctx context.Context, scope tenancy.Scope, executionID string, quantity int64, occurredAt time.Time, key string) (orderdomain.UsageRecord, error) {
	if quantity < 0 {
		return orderdomain.UsageRecord{}, platform.Invalid("invalid_quantity", "usage quantity cannot be negative")
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	return runCommand(ctx, s, scope, key, "usage.record", func(tx pgx.Tx) (orderdomain.UsageRecord, string, error) {
		execution, err := getExecution(ctx, tx, scope.TenantID, executionID, false)
		if err != nil {
			return orderdomain.UsageRecord{}, "", err
		}
		if execution.Status != "succeeded" && execution.Status != "reconciling" && execution.Status != "settled" {
			return orderdomain.UsageRecord{}, "", platform.Invalid("execution_incomplete", "usage requires a succeeded execution")
		}
		var meterID string
		if err = tx.QueryRow(ctx, `SELECT metering_definition_id FROM product_metering_bindings WHERE tenant_id=$1 AND product_version_id=$2`, scope.TenantID, execution.Bindings.ProductVersionID).Scan(&meterID); err != nil {
			return orderdomain.UsageRecord{}, "", err
		}
		var item orderdomain.UsageRecord
		err = tx.QueryRow(ctx, `INSERT INTO usage_records (tenant_id,order_id,order_item_id,execution_order_id,meter_id,quantity,occurred_at,idempotency_key,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id,tenant_id,order_id,order_item_id,execution_order_id,meter_id,quantity,idempotency_key,created_by,occurred_at,created_at`, scope.TenantID, execution.OrderID, execution.OrderItemID, execution.ID, meterID, quantity, occurredAt.UTC(), key, scope.ActorID).Scan(&item.ID, &item.TenantID, &item.OrderID, &item.OrderItemID, &item.ExecutionOrderID, &item.MeterID, &item.Quantity, &item.IdempotencyKey, &item.CreatedBy, &item.OccurredAt, &item.CreatedAt)
		if err != nil {
			return item, "", err
		}
		if err = appendAudit(ctx, tx, scope, "usage.record", "usage_record", item.ID, key, map[string]any{"execution_order_id": executionID, "quantity": quantity}); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "execution_order", executionID, "usage.recorded", execution.Attempt+1, map[string]any{"usage_record_id": item.ID, "quantity": quantity}); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) RecordProviderCost(ctx context.Context, scope tenancy.Scope, executionID, providerEndpointID, currency string, amountMinor int64, key string) (orderdomain.ProviderCost, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if providerEndpointID == "" || len(currency) != 3 || amountMinor < 0 {
		return orderdomain.ProviderCost{}, platform.Invalid("invalid_provider_cost", "provider endpoint, currency, and non-negative amount are required")
	}
	return runCommand(ctx, s, scope, key, "provider_cost.record", func(tx pgx.Tx) (orderdomain.ProviderCost, string, error) {
		execution, err := getExecution(ctx, tx, scope.TenantID, executionID, false)
		if err != nil {
			return orderdomain.ProviderCost{}, "", err
		}
		var exists bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM provider_endpoints WHERE tenant_id=$1 AND id=$2)`, scope.TenantID, providerEndpointID).Scan(&exists); err != nil {
			return orderdomain.ProviderCost{}, "", err
		}
		if !exists {
			return orderdomain.ProviderCost{}, "", platform.Invalid("invalid_reference", "provider endpoint does not belong to the tenant")
		}
		var item orderdomain.ProviderCost
		err = tx.QueryRow(ctx, `INSERT INTO provider_costs (tenant_id,order_id,order_item_id,execution_order_id,provider_endpoint_id,currency,amount_minor,idempotency_key,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id,tenant_id,order_id,order_item_id,execution_order_id,provider_endpoint_id,currency,amount_minor,idempotency_key,created_by,created_at`, scope.TenantID, execution.OrderID, execution.OrderItemID, execution.ID, providerEndpointID, currency, amountMinor, key, scope.ActorID).Scan(&item.ID, &item.TenantID, &item.OrderID, &item.OrderItemID, &item.ExecutionOrderID, &item.ProviderEndpointID, &item.Currency, &item.AmountMinor, &item.IdempotencyKey, &item.CreatedBy, &item.CreatedAt)
		if err != nil {
			return item, "", err
		}
		item.Currency = strings.TrimSpace(item.Currency)
		if err = appendAudit(ctx, tx, scope, "provider_cost.record", "provider_cost", item.ID, key, map[string]any{"execution_order_id": executionID, "amount_minor": amountMinor, "currency": currency}); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "execution_order", executionID, "provider_cost.recorded", execution.Attempt+1, map[string]any{"provider_cost_id": item.ID, "amount_minor": amountMinor, "currency": currency}); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) CreateCustomerCharge(ctx context.Context, scope tenancy.Scope, executionID, key string) (orderdomain.CustomerCharge, error) {
	return runCommand(ctx, s, scope, key, "customer_charge.create", func(tx pgx.Tx) (orderdomain.CustomerCharge, string, error) {
		execution, err := getExecution(ctx, tx, scope.TenantID, executionID, false)
		if err != nil {
			return orderdomain.CustomerCharge{}, "", err
		}
		var quantity int64
		if err = tx.QueryRow(ctx, `SELECT COALESCE(sum(quantity),0) FROM usage_records WHERE tenant_id=$1 AND execution_order_id=$2`, scope.TenantID, executionID).Scan(&quantity); err != nil {
			return orderdomain.CustomerCharge{}, "", err
		}
		if quantity == 0 {
			return orderdomain.CustomerCharge{}, "", platform.Invalid("usage_required", "customer charge requires recorded usage")
		}
		book := pricing.PriceBook{ID: execution.Bindings.PricingVersionID, TenantID: scope.TenantID, Rules: []pricing.Rule{}}
		if err = tx.QueryRow(ctx, `SELECT currency,version FROM price_books WHERE tenant_id=$1 AND id=$2`, scope.TenantID, book.ID).Scan(&book.Currency, &book.Version); err != nil {
			return orderdomain.CustomerCharge{}, "", err
		}
		rows, err := tx.Query(ctx, `SELECT definition FROM price_rules WHERE tenant_id=$1 AND price_book_id=$2 ORDER BY priority,id`, scope.TenantID, book.ID)
		if err != nil {
			return orderdomain.CustomerCharge{}, "", err
		}
		for rows.Next() {
			var definition []byte
			if err = rows.Scan(&definition); err != nil {
				rows.Close()
				return orderdomain.CustomerCharge{}, "", err
			}
			var rule pricing.Rule
			if err = json.Unmarshal(definition, &rule); err != nil {
				rows.Close()
				return orderdomain.CustomerCharge{}, "", err
			}
			book.Rules = append(book.Rules, rule)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return orderdomain.CustomerCharge{}, "", err
		}
		money, err := book.Calculate(quantity)
		if err != nil {
			return orderdomain.CustomerCharge{}, "", platform.Invalid("pricing_failed", err.Error())
		}
		var item orderdomain.CustomerCharge
		err = tx.QueryRow(ctx, `INSERT INTO customer_charges (tenant_id,order_id,order_item_id,execution_order_id,price_book_id,currency,amount_minor,status,idempotency_key,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,'calculated',$8,$9) RETURNING id,tenant_id,order_id,order_item_id,execution_order_id,price_book_id,currency,amount_minor,status,idempotency_key,created_by,created_at`, scope.TenantID, execution.OrderID, execution.OrderItemID, execution.ID, book.ID, money.Currency, money.Minor, key, scope.ActorID).Scan(&item.ID, &item.TenantID, &item.OrderID, &item.OrderItemID, &item.ExecutionOrderID, &item.PriceBookID, &item.Currency, &item.AmountMinor, &item.Status, &item.IdempotencyKey, &item.CreatedBy, &item.CreatedAt)
		if err != nil {
			return item, "", err
		}
		item.Currency = strings.TrimSpace(item.Currency)
		if err = appendAudit(ctx, tx, scope, "customer_charge.create", "customer_charge", item.ID, key, map[string]any{"execution_order_id": executionID, "amount_minor": item.AmountMinor, "currency": item.Currency}); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "execution_order", executionID, "customer_charge.calculated", execution.Attempt+1, map[string]any{"customer_charge_id": item.ID, "amount_minor": item.AmountMinor, "currency": item.Currency}); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

var _ application.TransactionStore = (*Store)(nil)

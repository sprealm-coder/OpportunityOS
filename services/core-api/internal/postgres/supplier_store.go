package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/application"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/channel"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

func (s *Store) CreateSupplier(ctx context.Context, scope tenancy.Scope, input application.SupplierInput, key string) (channel.Supplier, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return channel.Supplier{}, platform.Invalid("supplier_name_required", "supplier name is required")
	}
	return runCommand(ctx, s, scope, key, "supplier.create", func(tx pgx.Tx) (channel.Supplier, string, error) {
		item, err := scanSupplier(tx.QueryRow(ctx, `INSERT INTO suppliers (tenant_id,name,created_by) VALUES($1,$2,$3) RETURNING id,tenant_id,name,status,created_by,created_at,updated_at`, scope.TenantID, input.Name, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"name": item.Name}
		if err = appendAudit(ctx, tx, scope, "supplier.create", "supplier", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "supplier", item.ID, "supplier.created", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) BindSupplierCapability(ctx context.Context, scope tenancy.Scope, input application.SupplierCapabilityInput, key string) (channel.SupplierCapability, error) {
	if input.SupplierID == "" || input.CapabilityID == "" {
		return channel.SupplierCapability{}, platform.Invalid("supplier_capability_required", "supplier and capability are required")
	}
	return runCommand(ctx, s, scope, key, "supplier.capability.bind", func(tx pgx.Tx) (channel.SupplierCapability, string, error) {
		var valid bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM suppliers s, capabilities c WHERE s.tenant_id=$1 AND s.id=$2 AND s.status='active' AND c.tenant_id=$1 AND c.id=$3)`, scope.TenantID, input.SupplierID, input.CapabilityID).Scan(&valid); err != nil {
			return channel.SupplierCapability{}, "", err
		}
		if !valid {
			return channel.SupplierCapability{}, "", platform.Invalid("supplier_capability_invalid", "active supplier and capability must belong to the tenant")
		}
		item, err := scanSupplierCapability(tx.QueryRow(ctx, `INSERT INTO supplier_capabilities (tenant_id,supplier_id,capability_id) VALUES($1,$2,$3) RETURNING id,tenant_id,supplier_id,capability_id,status,created_at`, scope.TenantID, input.SupplierID, input.CapabilityID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"supplier_id": item.SupplierID, "capability_id": item.CapabilityID}
		if err = appendAudit(ctx, tx, scope, "supplier.capability.bind", "supplier_capability", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "supplier", item.SupplierID, "supplier.capability_bound", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) BindProviderSupplier(ctx context.Context, scope tenancy.Scope, providerID string, input application.ProviderSupplierInput, key string) (channel.Supplier, error) {
	if providerID == "" || input.SupplierID == "" {
		return channel.Supplier{}, platform.Invalid("provider_supplier_required", "provider and supplier are required")
	}
	return runCommand(ctx, s, scope, key, "provider.supplier.bind", func(tx pgx.Tx) (channel.Supplier, string, error) {
		item, err := scanSupplier(tx.QueryRow(ctx, `SELECT id,tenant_id,name,status,created_by,created_at,updated_at FROM suppliers WHERE tenant_id=$1 AND id=$2 AND status='active'`, scope.TenantID, input.SupplierID))
		if err != nil {
			return item, "", err
		}
		var existing string
		if err = tx.QueryRow(ctx, `SELECT COALESCE(supplier_id::text,'') FROM providers WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, providerID).Scan(&existing); err != nil {
			return item, "", err
		}
		if existing != "" && existing != input.SupplierID {
			return item, "", platform.Invalid("provider_already_bound", "provider is already owned by another supplier")
		}
		if _, err = tx.Exec(ctx, `UPDATE providers SET supplier_id=$3 WHERE tenant_id=$1 AND id=$2`, scope.TenantID, providerID, input.SupplierID); err != nil {
			return item, "", err
		}
		metadata := map[string]any{"provider_id": providerID, "supplier_id": input.SupplierID}
		if err = appendAudit(ctx, tx, scope, "provider.supplier.bind", "provider", providerID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "provider", providerID, "provider.supplier_bound", 1, metadata); err != nil {
			return item, "", err
		}
		return item, providerID, nil
	})
}

func (s *Store) CreateSupplierContract(ctx context.Context, scope tenancy.Scope, input application.SupplierContractInput, key string) (channel.SupplierContract, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	if input.SupplierID == "" || input.Name == "" || len(input.Currency) != 3 || (input.StartsAt != nil && input.EndsAt != nil && input.EndsAt.Before(*input.StartsAt)) {
		return channel.SupplierContract{}, platform.Invalid("invalid_supplier_contract", "supplier, name, currency, and valid dates are required")
	}
	terms, err := marshalChannelJSON(input.Terms)
	if err != nil {
		return channel.SupplierContract{}, err
	}
	return runCommand(ctx, s, scope, key, "supplier_contract.create", func(tx pgx.Tx) (channel.SupplierContract, string, error) {
		var valid bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM suppliers s WHERE s.tenant_id=$1 AND s.id=$2 AND s.status='active' AND ($3::uuid IS NULL OR EXISTS(SELECT 1 FROM providers p WHERE p.tenant_id=$1 AND p.id=$3 AND p.supplier_id=s.id)))`, scope.TenantID, input.SupplierID, nullableUUID(input.ProviderID)).Scan(&valid); err != nil {
			return channel.SupplierContract{}, "", err
		}
		if !valid {
			return channel.SupplierContract{}, "", platform.Invalid("supplier_contract_binding_invalid", "supplier must be active and provider must be bound to it")
		}
		item, err := scanSupplierContract(tx.QueryRow(ctx, `INSERT INTO supplier_contracts (tenant_id,supplier_id,provider_id,name,currency,terms,starts_at,ends_at,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id,tenant_id,supplier_id,COALESCE(provider_id::text,''),name,status,currency,terms,version,starts_at,ends_at,created_by,COALESCE(approved_by,''),created_at,updated_at`, scope.TenantID, input.SupplierID, nullableUUID(input.ProviderID), input.Name, input.Currency, terms, input.StartsAt, input.EndsAt, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"supplier_id": item.SupplierID, "provider_id": item.ProviderID, "currency": item.Currency}
		if err = appendAudit(ctx, tx, scope, "supplier_contract.create", "supplier_contract", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "supplier_contract", item.ID, "supplier_contract.created", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func getSupplierContract(ctx context.Context, tx pgx.Tx, tenantID, id string, lock bool) (channel.SupplierContract, error) {
	query := `SELECT id,tenant_id,supplier_id,COALESCE(provider_id::text,''),name,status,currency,terms,version,starts_at,ends_at,created_by,COALESCE(approved_by,''),created_at,updated_at FROM supplier_contracts WHERE tenant_id=$1 AND id=$2`
	if lock {
		query += ` FOR UPDATE`
	}
	return scanSupplierContract(tx.QueryRow(ctx, query, tenantID, id))
}

func updateSupplierContract(ctx context.Context, tx pgx.Tx, scope tenancy.Scope, item channel.SupplierContract, from, key, action string) (channel.SupplierContract, string, error) {
	updated, err := scanSupplierContract(tx.QueryRow(ctx, `UPDATE supplier_contracts SET status=$3,version=$4,approved_by=$5,updated_at=now() WHERE tenant_id=$1 AND id=$2 RETURNING id,tenant_id,supplier_id,COALESCE(provider_id::text,''),name,status,currency,terms,version,starts_at,ends_at,created_by,COALESCE(approved_by,''),created_at,updated_at`, scope.TenantID, item.ID, item.Status, item.Version, nullableText(item.ApprovedBy)))
	if err != nil {
		return updated, "", err
	}
	metadata := map[string]any{"from": from, "to": updated.Status, "supplier_id": updated.SupplierID}
	if err = appendAudit(ctx, tx, scope, action, "supplier_contract", updated.ID, key, metadata); err != nil {
		return updated, "", err
	}
	if err = appendEvent(ctx, tx, scope, "supplier_contract", updated.ID, "supplier_contract."+updated.Status, updated.Version, metadata); err != nil {
		return updated, "", err
	}
	return updated, updated.ID, nil
}

func (s *Store) TransitionSupplierContract(ctx context.Context, scope tenancy.Scope, id, to, key string) (channel.SupplierContract, error) {
	to = strings.ToLower(strings.TrimSpace(to))
	if to == "approved" || to == "draft" {
		return channel.SupplierContract{}, platform.Invalid("supplier_review_required", "approval and rejection require the review route")
	}
	return runCommand(ctx, s, scope, key, "supplier_contract.transition", func(tx pgx.Tx) (channel.SupplierContract, string, error) {
		item, err := getSupplierContract(ctx, tx, scope.TenantID, id, true)
		if err != nil {
			return item, "", err
		}
		from := item.Status
		if to == "active" {
			now := time.Now().UTC()
			if item.StartsAt != nil && item.StartsAt.After(now) || item.EndsAt != nil && !item.EndsAt.After(now) {
				return item, "", platform.Invalid("contract_period_inactive", "contract dates do not permit activation")
			}
			var hasRate bool
			if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM supplier_rates WHERE tenant_id=$1 AND contract_id=$2 AND status='active')`, scope.TenantID, id).Scan(&hasRate); err != nil {
				return item, "", err
			}
			if !hasRate {
				return item, "", platform.Invalid("supplier_rate_required", "approved contract requires an active rate before activation")
			}
		}
		if err = item.Transition(to, scope.ActorID); err != nil {
			return item, "", err
		}
		return updateSupplierContract(ctx, tx, scope, item, from, key, "supplier_contract.transition")
	})
}

func (s *Store) ReviewSupplierContract(ctx context.Context, scope tenancy.Scope, id string, input application.ReviewInput, key string) (channel.SupplierContract, error) {
	input.Decision = strings.ToLower(strings.TrimSpace(input.Decision))
	to := "draft"
	if input.Decision == "approved" {
		to = "approved"
	} else if input.Decision != "rejected" {
		return channel.SupplierContract{}, platform.Invalid("invalid_contract_decision", "decision must be approved or rejected")
	}
	return runCommand(ctx, s, scope, key, "supplier_contract.review", func(tx pgx.Tx) (channel.SupplierContract, string, error) {
		item, err := getSupplierContract(ctx, tx, scope.TenantID, id, true)
		if err != nil {
			return item, "", err
		}
		if item.Status != "pending_approval" {
			return item, "", platform.Invalid("contract_not_pending", "only pending contracts can be reviewed")
		}
		if item.CreatedBy == scope.ActorID {
			return item, "", platform.ErrPermissionDenied
		}
		from := item.Status
		if err = item.Transition(to, scope.ActorID); err != nil {
			return item, "", err
		}
		return updateSupplierContract(ctx, tx, scope, item, from, key, "supplier_contract.review")
	})
}

func (s *Store) CreateSupplierRate(ctx context.Context, scope tenancy.Scope, input application.SupplierRateInput, key string) (channel.SupplierRate, error) {
	input.Unit = strings.TrimSpace(input.Unit)
	if input.ContractID == "" || input.CapabilityID == "" || input.Unit == "" || input.RateMinor < 0 {
		return channel.SupplierRate{}, platform.Invalid("invalid_supplier_rate", "contract, capability, unit, and non-negative integer rate are required")
	}
	return runCommand(ctx, s, scope, key, "supplier_rate.create", func(tx pgx.Tx) (channel.SupplierRate, string, error) {
		var valid bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM supplier_contracts sc JOIN supplier_capabilities cap ON cap.tenant_id=sc.tenant_id AND cap.supplier_id=sc.supplier_id AND cap.capability_id=$3 AND cap.status='active' WHERE sc.tenant_id=$1 AND sc.id=$2 AND sc.status IN ('approved','active'))`, scope.TenantID, input.ContractID, input.CapabilityID).Scan(&valid); err != nil {
			return channel.SupplierRate{}, "", err
		}
		if !valid {
			return channel.SupplierRate{}, "", platform.Invalid("supplier_rate_binding_invalid", "contract must be approved and capability must be bound to its supplier")
		}
		item, err := scanSupplierRate(tx.QueryRow(ctx, `INSERT INTO supplier_rates (tenant_id,contract_id,capability_id,unit,rate_minor,created_by) VALUES($1,$2,$3,$4,$5,$6) RETURNING id,tenant_id,contract_id,capability_id,unit,rate_minor,version,status,created_by,created_at`, scope.TenantID, input.ContractID, input.CapabilityID, input.Unit, input.RateMinor, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"contract_id": item.ContractID, "capability_id": item.CapabilityID, "unit": item.Unit, "rate_minor": item.RateMinor}
		if err = appendAudit(ctx, tx, scope, "supplier_rate.create", "supplier_rate", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "supplier_contract", item.ContractID, "supplier_rate.created", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) RecordSupplierQuality(ctx context.Context, scope tenancy.Scope, input application.SupplierQualityInput, key string) (channel.SupplierQualityRecord, error) {
	input.Metric = strings.TrimSpace(input.Metric)
	if input.SupplierID == "" || input.Metric == "" || input.ScoreBPS < 0 || input.ScoreBPS > 10000 || input.PeriodStart.IsZero() || input.PeriodEnd.Before(input.PeriodStart) {
		return channel.SupplierQualityRecord{}, platform.Invalid("invalid_supplier_quality", "supplier, metric, score, and valid period are required")
	}
	evidence, err := marshalChannelJSON(input.Evidence)
	if err != nil {
		return channel.SupplierQualityRecord{}, err
	}
	return runCommand(ctx, s, scope, key, "supplier_quality.record", func(tx pgx.Tx) (channel.SupplierQualityRecord, string, error) {
		var valid bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM suppliers s WHERE s.tenant_id=$1 AND s.id=$2 AND ($3::uuid IS NULL OR EXISTS(SELECT 1 FROM providers p WHERE p.tenant_id=$1 AND p.id=$3 AND p.supplier_id=s.id AND ($4::uuid IS NULL OR EXISTS(SELECT 1 FROM provider_endpoints ep WHERE ep.tenant_id=$1 AND ep.id=$4 AND ep.provider_id=p.id)))))`, scope.TenantID, input.SupplierID, nullableUUID(input.ProviderID), nullableUUID(input.ProviderEndpointID)).Scan(&valid); err != nil {
			return channel.SupplierQualityRecord{}, "", err
		}
		if !valid {
			return channel.SupplierQualityRecord{}, "", platform.Invalid("supplier_quality_binding_invalid", "supplier, provider, and endpoint binding is invalid")
		}
		item, err := scanSupplierQuality(tx.QueryRow(ctx, `INSERT INTO supplier_quality_records (tenant_id,supplier_id,provider_id,provider_endpoint_id,metric,score_bps,evidence,period_start,period_end,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id,tenant_id,supplier_id,COALESCE(provider_id::text,''),COALESCE(provider_endpoint_id::text,''),metric,score_bps,evidence,period_start,period_end,created_by,created_at`, scope.TenantID, input.SupplierID, nullableUUID(input.ProviderID), nullableUUID(input.ProviderEndpointID), input.Metric, input.ScoreBPS, evidence, input.PeriodStart, input.PeriodEnd, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"supplier_id": item.SupplierID, "provider_id": item.ProviderID, "metric": item.Metric, "score_bps": item.ScoreBPS}
		if err = appendAudit(ctx, tx, scope, "supplier_quality.record", "supplier_quality", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "supplier", item.SupplierID, "supplier_quality.recorded", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

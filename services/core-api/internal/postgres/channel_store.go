package postgres

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/application"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/channel"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

func marshalChannelJSON(value map[string]any) ([]byte, error) {
	if value == nil {
		value = map[string]any{}
	}
	return json.Marshal(value)
}

func unmarshalChannelJSON(raw []byte) (map[string]any, error) {
	value := map[string]any{}
	if len(raw) == 0 {
		return value, nil
	}
	return value, json.Unmarshal(raw, &value)
}

func collectChannelRows[T any](ctx context.Context, tx pgx.Tx, query string, scan func(rowScanner) (T, error), args ...any) ([]T, error) {
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []T{}
	for rows.Next() {
		item, scanErr := scan(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanResellerLevel(row rowScanner) (channel.ResellerLevel, error) {
	var item channel.ResellerLevel
	err := row.Scan(&item.ID, &item.TenantID, &item.Name, &item.Rank, &item.DefaultCommissionBPS, &item.Status, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanReseller(row rowScanner) (channel.Reseller, error) {
	var item channel.Reseller
	err := row.Scan(&item.ID, &item.TenantID, &item.LevelID, &item.Name, &item.Status, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanAttributionRule(row rowScanner) (channel.AttributionRule, error) {
	var item channel.AttributionRule
	var definition []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.Name, &item.Priority, &definition, &item.Status, &item.Version, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	if err == nil {
		item.Definition, err = unmarshalChannelJSON(definition)
	}
	return item, err
}

func scanLeadOwnership(row rowScanner) (channel.LeadOwnership, error) {
	var item channel.LeadOwnership
	err := row.Scan(&item.ID, &item.TenantID, &item.LeadID, &item.ResellerID, &item.AttributionRuleID, &item.Status, &item.ProtectionExpiresAt, &item.Version, &item.CreatedBy, &item.AcquiredAt, &item.UpdatedAt)
	return item, err
}

func scanCustomerOwnership(row rowScanner) (channel.CustomerOwnership, error) {
	var item channel.CustomerOwnership
	err := row.Scan(&item.ID, &item.TenantID, &item.CustomerID, &item.ResellerID, &item.SourceLeadOwnershipID, &item.Status, &item.ProtectionExpiresAt, &item.Version, &item.CreatedBy, &item.AcquiredAt, &item.UpdatedAt)
	return item, err
}

func scanTransferRequest(row rowScanner) (channel.TransferRequest, error) {
	var item channel.TransferRequest
	err := row.Scan(&item.ID, &item.TenantID, &item.OwnershipType, &item.OwnershipID, &item.FromResellerID, &item.ToResellerID, &item.Status, &item.Rationale, &item.RequestedBy, &item.ReviewedBy, &item.Version, &item.CreatedAt, &item.ReviewedAt)
	return item, err
}

func scanConflict(row rowScanner) (channel.ConflictRecord, error) {
	var item channel.ConflictRecord
	var resolution []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.OwnershipType, &item.OwnershipID, &item.ClaimantResellerIDs, &item.Status, &resolution, &item.CreatedBy, &item.ResolvedBy, &item.CreatedAt, &item.ResolvedAt)
	if err == nil {
		item.Resolution, err = unmarshalChannelJSON(resolution)
	}
	return item, err
}

func scanCommissionRule(row rowScanner) (channel.CommissionRule, error) {
	var item channel.CommissionRule
	err := row.Scan(&item.ID, &item.TenantID, &item.Name, &item.ResellerID, &item.ResellerLevelID, &item.TriggerType, &item.BasisPoints, &item.Status, &item.Version, &item.EffectiveFrom, &item.EffectiveUntil, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanCommissionLock(row rowScanner) (channel.CommissionLock, error) {
	var item channel.CommissionLock
	err := row.Scan(&item.ID, &item.TenantID, &item.CustomerChargeID, &item.ResellerID, &item.CommissionRuleID, &item.CommissionID, &item.Currency, &item.AmountMinor, &item.Status, &item.IdempotencyKey, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	item.Currency = strings.TrimSpace(item.Currency)
	return item, err
}

func scanSettlementCycle(row rowScanner) (channel.SettlementCycle, error) {
	var item channel.SettlementCycle
	err := row.Scan(&item.ID, &item.TenantID, &item.ResellerID, &item.Name, &item.PeriodStart, &item.PeriodEnd, &item.Status, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanSupplier(row rowScanner) (channel.Supplier, error) {
	var item channel.Supplier
	err := row.Scan(&item.ID, &item.TenantID, &item.Name, &item.Status, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanSupplierCapability(row rowScanner) (channel.SupplierCapability, error) {
	var item channel.SupplierCapability
	err := row.Scan(&item.ID, &item.TenantID, &item.SupplierID, &item.CapabilityID, &item.Status, &item.CreatedAt)
	return item, err
}

func scanProviderBinding(row rowScanner) (channel.ProviderSupplierBinding, error) {
	var item channel.ProviderSupplierBinding
	err := row.Scan(&item.ProviderID, &item.ProviderName, &item.SupplierID)
	return item, err
}

func scanSupplierContract(row rowScanner) (channel.SupplierContract, error) {
	var item channel.SupplierContract
	var terms []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.SupplierID, &item.ProviderID, &item.Name, &item.Status, &item.Currency, &terms, &item.Version, &item.StartsAt, &item.EndsAt, &item.CreatedBy, &item.ApprovedBy, &item.CreatedAt, &item.UpdatedAt)
	item.Currency = strings.TrimSpace(item.Currency)
	if err == nil {
		item.Terms, err = unmarshalChannelJSON(terms)
	}
	return item, err
}

func scanSupplierRate(row rowScanner) (channel.SupplierRate, error) {
	var item channel.SupplierRate
	err := row.Scan(&item.ID, &item.TenantID, &item.ContractID, &item.CapabilityID, &item.Unit, &item.RateMinor, &item.Version, &item.Status, &item.CreatedBy, &item.CreatedAt)
	return item, err
}

func scanSupplierQuality(row rowScanner) (channel.SupplierQualityRecord, error) {
	var item channel.SupplierQualityRecord
	var evidence []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.SupplierID, &item.ProviderID, &item.ProviderEndpointID, &item.Metric, &item.ScoreBPS, &evidence, &item.PeriodStart, &item.PeriodEnd, &item.CreatedBy, &item.CreatedAt)
	if err == nil {
		item.Evidence, err = unmarshalChannelJSON(evidence)
	}
	return item, err
}

func (s *Store) ListChannels(ctx context.Context, scope tenancy.Scope) (channel.Overview, error) {
	return runTenantRead(ctx, s, scope.TenantID, func(tx pgx.Tx) (channel.Overview, error) {
		var result channel.Overview
		var err error
		result.ResellerLevels, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,name,rank,default_commission_bps,status,created_by,created_at,updated_at FROM reseller_levels WHERE tenant_id=$1 ORDER BY rank,id`, scanResellerLevel, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.Resellers, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,COALESCE(level_id::text,''),name,status,created_by,created_at,updated_at FROM resellers WHERE tenant_id=$1 ORDER BY updated_at DESC,id`, scanReseller, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.AttributionRules, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,name,priority,definition,status,version,created_by,created_at,updated_at FROM attribution_rules WHERE tenant_id=$1 ORDER BY priority,id`, scanAttributionRule, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.LeadOwnerships, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,lead_id,reseller_id,attribution_rule_id,status,protection_expires_at,version,created_by,acquired_at,updated_at FROM lead_ownerships WHERE tenant_id=$1 ORDER BY updated_at DESC,id`, scanLeadOwnership, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.CustomerOwnerships, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,customer_id,reseller_id,COALESCE(source_lead_ownership_id::text,''),status,protection_expires_at,version,created_by,acquired_at,updated_at FROM customer_ownerships WHERE tenant_id=$1 ORDER BY updated_at DESC,id`, scanCustomerOwnership, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.TransferRequests, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,ownership_type,ownership_id,from_reseller_id,to_reseller_id,status,rationale,requested_by,COALESCE(reviewed_by,''),version,created_at,reviewed_at FROM transfer_requests WHERE tenant_id=$1 ORDER BY created_at DESC,id`, scanTransferRequest, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.Conflicts, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,ownership_type,ownership_id,claimant_reseller_ids::text[],status,resolution,created_by,COALESCE(resolved_by,''),created_at,resolved_at FROM conflict_records WHERE tenant_id=$1 ORDER BY created_at DESC,id`, scanConflict, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.CommissionRules, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,name,COALESCE(reseller_id::text,''),COALESCE(reseller_level_id::text,''),trigger_type,basis_points,status,version,effective_from,effective_until,created_by,created_at,updated_at FROM commission_rules WHERE tenant_id=$1 ORDER BY updated_at DESC,id`, scanCommissionRule, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.CommissionLocks, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,customer_charge_id,reseller_id,commission_rule_id,COALESCE(commission_id::text,''),currency,amount_minor,status,idempotency_key,created_by,created_at,updated_at FROM commission_locks WHERE tenant_id=$1 ORDER BY created_at DESC,id`, scanCommissionLock, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.SettlementCycles, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,reseller_id,name,period_start,period_end,status,created_by,created_at,updated_at FROM settlement_cycles WHERE tenant_id=$1 ORDER BY period_start DESC,id`, scanSettlementCycle, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.Suppliers, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,name,status,created_by,created_at,updated_at FROM suppliers WHERE tenant_id=$1 ORDER BY updated_at DESC,id`, scanSupplier, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.SupplierCapabilities, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,supplier_id,capability_id,status,created_at FROM supplier_capabilities WHERE tenant_id=$1 ORDER BY created_at DESC,id`, scanSupplierCapability, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.ProviderBindings, err = collectChannelRows(ctx, tx, `SELECT id,name,supplier_id FROM providers WHERE tenant_id=$1 AND supplier_id IS NOT NULL ORDER BY name,id`, scanProviderBinding, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.SupplierContracts, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,supplier_id,COALESCE(provider_id::text,''),name,status,currency,terms,version,starts_at,ends_at,created_by,COALESCE(approved_by,''),created_at,updated_at FROM supplier_contracts WHERE tenant_id=$1 ORDER BY updated_at DESC,id`, scanSupplierContract, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.SupplierRates, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,contract_id,capability_id,unit,rate_minor,version,status,created_by,created_at FROM supplier_rates WHERE tenant_id=$1 ORDER BY created_at DESC,id`, scanSupplierRate, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.SupplierQuality, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,supplier_id,COALESCE(provider_id::text,''),COALESCE(provider_endpoint_id::text,''),metric,score_bps,evidence,period_start,period_end,created_by,created_at FROM supplier_quality_records WHERE tenant_id=$1 ORDER BY period_end DESC,id`, scanSupplierQuality, scope.TenantID)
		if err != nil {
			return result, err
		}
		rows, queryErr := tx.Query(ctx, `SELECT pp.id,pp.tenant_id,pp.provider_cost_id,pp.provider_id,pp.currency,pp.amount_minor,pp.settled_minor,pp.status,pp.payable_account_id,pp.ledger_transaction_id,pp.idempotency_key,pp.created_by,pp.created_at,pp.updated_at FROM provider_payables pp JOIN providers p ON p.tenant_id=pp.tenant_id AND p.id=pp.provider_id WHERE pp.tenant_id=$1 AND p.supplier_id IS NOT NULL ORDER BY pp.created_at DESC`, scope.TenantID)
		if queryErr != nil {
			return result, queryErr
		}
		result.ProviderPayables, err = listRows(rows, scanProviderPayable)
		if err != nil {
			return result, err
		}
		rows, queryErr = tx.Query(ctx, `SELECT s.id,s.tenant_id,s.source_type,s.source_id,s.beneficiary_type,s.beneficiary_id,s.currency,s.amount_minor,s.status,s.ledger_transaction_id,s.idempotency_key,s.created_by,s.created_at FROM settlements s JOIN commissions c ON s.source_type='commission' AND c.tenant_id=s.tenant_id AND c.id=s.source_id WHERE s.tenant_id=$1 AND c.beneficiary_type='reseller' ORDER BY s.created_at DESC`, scope.TenantID)
		if queryErr != nil {
			return result, queryErr
		}
		result.ResellerSettlements, err = listRows(rows, scanSettlement)
		if err != nil {
			return result, err
		}
		rows, queryErr = tx.Query(ctx, `SELECT s.id,s.tenant_id,s.source_type,s.source_id,s.beneficiary_type,s.beneficiary_id,s.currency,s.amount_minor,s.status,s.ledger_transaction_id,s.idempotency_key,s.created_by,s.created_at FROM settlements s JOIN provider_payables pp ON s.source_type='provider_payable' AND pp.tenant_id=s.tenant_id AND pp.id=s.source_id JOIN providers p ON p.tenant_id=pp.tenant_id AND p.id=pp.provider_id WHERE s.tenant_id=$1 AND p.supplier_id IS NOT NULL ORDER BY s.created_at DESC`, scope.TenantID)
		if queryErr != nil {
			return result, queryErr
		}
		result.SupplierSettlements, err = listRows(rows, scanSettlement)
		if err != nil {
			return result, err
		}
		result.Marketplace, err = listMarketplace(ctx, tx, scope.TenantID)
		return result, err
	})
}

func (s *Store) CreateResellerLevel(ctx context.Context, scope tenancy.Scope, input application.ResellerLevelInput, key string) (channel.ResellerLevel, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || input.Rank < 1 || input.DefaultCommissionBPS < 0 || input.DefaultCommissionBPS > 10000 {
		return channel.ResellerLevel{}, platform.Invalid("invalid_reseller_level", "name, positive rank, and commission basis points are required")
	}
	return runCommand(ctx, s, scope, key, "reseller_level.create", func(tx pgx.Tx) (channel.ResellerLevel, string, error) {
		item, err := scanResellerLevel(tx.QueryRow(ctx, `INSERT INTO reseller_levels (tenant_id,name,rank,default_commission_bps,created_by) VALUES($1,$2,$3,$4,$5) RETURNING id,tenant_id,name,rank,default_commission_bps,status,created_by,created_at,updated_at`, scope.TenantID, input.Name, input.Rank, input.DefaultCommissionBPS, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"name": item.Name, "rank": item.Rank}
		if err = appendAudit(ctx, tx, scope, "reseller_level.create", "reseller_level", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "reseller_level", item.ID, "reseller_level.created", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) CreateReseller(ctx context.Context, scope tenancy.Scope, input application.ResellerInput, key string) (channel.Reseller, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return channel.Reseller{}, platform.Invalid("reseller_name_required", "reseller name is required")
	}
	return runCommand(ctx, s, scope, key, "reseller.create", func(tx pgx.Tx) (channel.Reseller, string, error) {
		if input.LevelID != "" {
			var active bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM reseller_levels WHERE tenant_id=$1 AND id=$2 AND status='active')`, scope.TenantID, input.LevelID).Scan(&active); err != nil {
				return channel.Reseller{}, "", err
			}
			if !active {
				return channel.Reseller{}, "", platform.Invalid("invalid_reseller_level", "reseller level must be active in the tenant")
			}
		}
		item, err := scanReseller(tx.QueryRow(ctx, `INSERT INTO resellers (tenant_id,level_id,name,created_by) VALUES($1,$2,$3,$4) RETURNING id,tenant_id,COALESCE(level_id::text,''),name,status,created_by,created_at,updated_at`, scope.TenantID, nullableUUID(input.LevelID), input.Name, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"name": item.Name, "level_id": item.LevelID}
		if err = appendAudit(ctx, tx, scope, "reseller.create", "reseller", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "reseller", item.ID, "reseller.created", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) CreateAttributionRule(ctx context.Context, scope tenancy.Scope, input application.AttributionRuleInput, key string) (channel.AttributionRule, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || input.Priority < 1 || len(input.Definition) == 0 {
		return channel.AttributionRule{}, platform.Invalid("invalid_attribution_rule", "name, priority, and definition are required")
	}
	definition, err := marshalChannelJSON(input.Definition)
	if err != nil {
		return channel.AttributionRule{}, err
	}
	return runCommand(ctx, s, scope, key, "attribution_rule.create", func(tx pgx.Tx) (channel.AttributionRule, string, error) {
		item, err := scanAttributionRule(tx.QueryRow(ctx, `INSERT INTO attribution_rules (tenant_id,name,priority,definition,created_by) VALUES($1,$2,$3,$4,$5) RETURNING id,tenant_id,name,priority,definition,status,version,created_by,created_at,updated_at`, scope.TenantID, input.Name, input.Priority, definition, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"name": item.Name, "priority": item.Priority}
		if err = appendAudit(ctx, tx, scope, "attribution_rule.create", "attribution_rule", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "attribution_rule", item.ID, "attribution_rule.created", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) AssignLeadOwnership(ctx context.Context, scope tenancy.Scope, input application.LeadOwnershipInput, key string) (channel.LeadOwnership, error) {
	if input.LeadID == "" || input.ResellerID == "" || input.AttributionRuleID == "" || input.ProtectionDays < 1 || input.ProtectionDays > 730 {
		return channel.LeadOwnership{}, platform.Invalid("invalid_lead_ownership", "lead, reseller, attribution rule, and protection days from 1 to 730 are required")
	}
	return runCommand(ctx, s, scope, key, "lead_ownership.assign", func(tx pgx.Tx) (channel.LeadOwnership, string, error) {
		var valid bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM leads l, resellers r, attribution_rules ar WHERE l.tenant_id=$1 AND l.id=$2 AND r.tenant_id=$1 AND r.id=$3 AND r.status='active' AND ar.tenant_id=$1 AND ar.id=$4 AND ar.status='active')`, scope.TenantID, input.LeadID, input.ResellerID, input.AttributionRuleID).Scan(&valid); err != nil {
			return channel.LeadOwnership{}, "", err
		}
		if !valid {
			return channel.LeadOwnership{}, "", platform.Invalid("ownership_reference_invalid", "lead, active reseller, and active rule must belong to the tenant")
		}
		expires := time.Now().UTC().AddDate(0, 0, input.ProtectionDays)
		item, err := scanLeadOwnership(tx.QueryRow(ctx, `INSERT INTO lead_ownerships (tenant_id,lead_id,reseller_id,attribution_rule_id,protection_expires_at,created_by) VALUES($1,$2,$3,$4,$5,$6) RETURNING id,tenant_id,lead_id,reseller_id,attribution_rule_id,status,protection_expires_at,version,created_by,acquired_at,updated_at`, scope.TenantID, input.LeadID, input.ResellerID, input.AttributionRuleID, expires, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"lead_id": item.LeadID, "reseller_id": item.ResellerID, "protection_expires_at": item.ProtectionExpiresAt}
		if err = appendAudit(ctx, tx, scope, "lead_ownership.assign", "lead_ownership", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "lead_ownership", item.ID, "lead_ownership.assigned", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) CreateCustomerOwnership(ctx context.Context, scope tenancy.Scope, input application.CustomerOwnershipInput, key string) (channel.CustomerOwnership, error) {
	input.CustomerID = strings.TrimSpace(input.CustomerID)
	if input.CustomerID == "" || input.ResellerID == "" || input.ProtectionDays < 1 || input.ProtectionDays > 730 {
		return channel.CustomerOwnership{}, platform.Invalid("invalid_customer_ownership", "customer, reseller, and protection days from 1 to 730 are required")
	}
	return runCommand(ctx, s, scope, key, "customer_ownership.create", func(tx pgx.Tx) (channel.CustomerOwnership, string, error) {
		var resellerActive bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM resellers WHERE tenant_id=$1 AND id=$2 AND status='active')`, scope.TenantID, input.ResellerID).Scan(&resellerActive); err != nil {
			return channel.CustomerOwnership{}, "", err
		}
		if !resellerActive {
			return channel.CustomerOwnership{}, "", platform.Invalid("reseller_inactive", "customer ownership requires an active tenant reseller")
		}
		if input.SourceLeadOwnershipID != "" {
			var matches bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM lead_ownerships WHERE tenant_id=$1 AND id=$2 AND reseller_id=$3 AND status='protected')`, scope.TenantID, input.SourceLeadOwnershipID, input.ResellerID).Scan(&matches); err != nil {
				return channel.CustomerOwnership{}, "", err
			}
			if !matches {
				return channel.CustomerOwnership{}, "", platform.Invalid("lead_ownership_mismatch", "source lead ownership must be protected by the same reseller")
			}
		}
		expires := time.Now().UTC().AddDate(0, 0, input.ProtectionDays)
		item, err := scanCustomerOwnership(tx.QueryRow(ctx, `INSERT INTO customer_ownerships (tenant_id,customer_id,reseller_id,source_lead_ownership_id,protection_expires_at,created_by) VALUES($1,$2,$3,$4,$5,$6) RETURNING id,tenant_id,customer_id,reseller_id,COALESCE(source_lead_ownership_id::text,''),status,protection_expires_at,version,created_by,acquired_at,updated_at`, scope.TenantID, input.CustomerID, input.ResellerID, nullableUUID(input.SourceLeadOwnershipID), expires, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"customer_id": item.CustomerID, "reseller_id": item.ResellerID, "source_lead_ownership_id": item.SourceLeadOwnershipID}
		if err = appendAudit(ctx, tx, scope, "customer_ownership.create", "customer_ownership", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "customer_ownership", item.ID, "customer_ownership.created", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) CreateTransferRequest(ctx context.Context, scope tenancy.Scope, input application.TransferRequestInput, key string) (channel.TransferRequest, error) {
	input.OwnershipType = strings.ToLower(strings.TrimSpace(input.OwnershipType))
	input.Rationale = strings.TrimSpace(input.Rationale)
	if (input.OwnershipType != "lead" && input.OwnershipType != "customer") || input.OwnershipID == "" || input.ToResellerID == "" || input.Rationale == "" {
		return channel.TransferRequest{}, platform.Invalid("invalid_transfer_request", "ownership, target reseller, and rationale are required")
	}
	return runCommand(ctx, s, scope, key, "ownership_transfer.request", func(tx pgx.Tx) (channel.TransferRequest, string, error) {
		var fromResellerID string
		table := "lead_ownerships"
		if input.OwnershipType == "customer" {
			table = "customer_ownerships"
		}
		if err := tx.QueryRow(ctx, `SELECT reseller_id FROM `+table+` WHERE tenant_id=$1 AND id=$2 AND status='protected' FOR UPDATE`, scope.TenantID, input.OwnershipID).Scan(&fromResellerID); err != nil {
			return channel.TransferRequest{}, "", err
		}
		var targetActive bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM resellers WHERE tenant_id=$1 AND id=$2 AND status='active')`, scope.TenantID, input.ToResellerID).Scan(&targetActive); err != nil {
			return channel.TransferRequest{}, "", err
		}
		if !targetActive {
			return channel.TransferRequest{}, "", platform.Invalid("target_reseller_inactive", "target reseller must be active")
		}
		item, err := scanTransferRequest(tx.QueryRow(ctx, `INSERT INTO transfer_requests (tenant_id,ownership_type,ownership_id,from_reseller_id,to_reseller_id,rationale,requested_by) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id,tenant_id,ownership_type,ownership_id,from_reseller_id,to_reseller_id,status,rationale,requested_by,COALESCE(reviewed_by,''),version,created_at,reviewed_at`, scope.TenantID, input.OwnershipType, input.OwnershipID, fromResellerID, input.ToResellerID, input.Rationale, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"ownership_type": item.OwnershipType, "ownership_id": item.OwnershipID, "from_reseller_id": item.FromResellerID, "to_reseller_id": item.ToResellerID}
		if err = appendAudit(ctx, tx, scope, "ownership_transfer.request", "transfer_request", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "transfer_request", item.ID, "ownership_transfer.requested", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) ReviewTransfer(ctx context.Context, scope tenancy.Scope, id string, input application.ReviewInput, key string) (channel.TransferRequest, error) {
	input.Decision = strings.ToLower(strings.TrimSpace(input.Decision))
	input.Rationale = strings.TrimSpace(input.Rationale)
	if input.Decision != "approved" && input.Decision != "rejected" {
		return channel.TransferRequest{}, platform.Invalid("invalid_transfer_decision", "decision must be approved or rejected")
	}
	return runCommand(ctx, s, scope, key, "ownership_transfer.review", func(tx pgx.Tx) (channel.TransferRequest, string, error) {
		item, err := scanTransferRequest(tx.QueryRow(ctx, `SELECT id,tenant_id,ownership_type,ownership_id,from_reseller_id,to_reseller_id,status,rationale,requested_by,COALESCE(reviewed_by,''),version,created_at,reviewed_at FROM transfer_requests WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, id))
		if err != nil {
			return item, "", err
		}
		if item.Status != "pending" {
			return item, "", platform.Invalid("transfer_not_pending", "only pending transfers can be reviewed")
		}
		if item.RequestedBy == scope.ActorID {
			return item, "", platform.ErrPermissionDenied
		}
		if input.Decision == "approved" {
			table := "lead_ownerships"
			if item.OwnershipType == "customer" {
				table = "customer_ownerships"
			}
			result, updateErr := tx.Exec(ctx, `UPDATE `+table+` SET reseller_id=$3,status='protected',version=version+1,updated_at=now() WHERE tenant_id=$1 AND id=$2 AND reseller_id=$4 AND status='protected'`, scope.TenantID, item.OwnershipID, item.ToResellerID, item.FromResellerID)
			if updateErr != nil {
				return item, "", updateErr
			}
			if result.RowsAffected() != 1 {
				return item, "", platform.Invalid("ownership_changed", "ownership changed before transfer review")
			}
		}
		item, err = scanTransferRequest(tx.QueryRow(ctx, `UPDATE transfer_requests SET status=$3,reviewed_by=$4,reviewed_at=now(),version=version+1 WHERE tenant_id=$1 AND id=$2 RETURNING id,tenant_id,ownership_type,ownership_id,from_reseller_id,to_reseller_id,status,rationale,requested_by,COALESCE(reviewed_by,''),version,created_at,reviewed_at`, scope.TenantID, id, input.Decision, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"decision": item.Status, "ownership_type": item.OwnershipType, "ownership_id": item.OwnershipID, "rationale": input.Rationale}
		if err = appendAudit(ctx, tx, scope, "ownership_transfer.review", "transfer_request", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "transfer_request", item.ID, "ownership_transfer."+item.Status, item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) CreateCommissionRule(ctx context.Context, scope tenancy.Scope, input application.CommissionRuleInput, key string) (channel.CommissionRule, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || (input.ResellerID == "" && input.ResellerLevelID == "") || input.BasisPoints < 0 || input.BasisPoints > 10000 {
		return channel.CommissionRule{}, platform.Invalid("invalid_commission_rule", "name, reseller or level, and basis points are required")
	}
	if input.EffectiveFrom.IsZero() {
		input.EffectiveFrom = time.Now().UTC()
	}
	if input.EffectiveUntil != nil && input.EffectiveUntil.Before(input.EffectiveFrom) {
		return channel.CommissionRule{}, platform.Invalid("invalid_effective_period", "effective end must not precede start")
	}
	return runCommand(ctx, s, scope, key, "commission_rule.create", func(tx pgx.Tx) (channel.CommissionRule, string, error) {
		var valid bool
		if err := tx.QueryRow(ctx, `SELECT ($2::uuid IS NULL OR EXISTS(SELECT 1 FROM resellers WHERE tenant_id=$1 AND id=$2 AND status='active')) AND ($3::uuid IS NULL OR EXISTS(SELECT 1 FROM reseller_levels WHERE tenant_id=$1 AND id=$3 AND status='active'))`, scope.TenantID, nullableUUID(input.ResellerID), nullableUUID(input.ResellerLevelID)).Scan(&valid); err != nil {
			return channel.CommissionRule{}, "", err
		}
		if !valid {
			return channel.CommissionRule{}, "", platform.Invalid("commission_scope_invalid", "commission scope must belong to the tenant and be active")
		}
		item, err := scanCommissionRule(tx.QueryRow(ctx, `INSERT INTO commission_rules (tenant_id,name,reseller_id,reseller_level_id,basis_points,effective_from,effective_until,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id,tenant_id,name,COALESCE(reseller_id::text,''),COALESCE(reseller_level_id::text,''),trigger_type,basis_points,status,version,effective_from,effective_until,created_by,created_at,updated_at`, scope.TenantID, input.Name, nullableUUID(input.ResellerID), nullableUUID(input.ResellerLevelID), input.BasisPoints, input.EffectiveFrom, input.EffectiveUntil, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"basis_points": item.BasisPoints, "reseller_id": item.ResellerID, "reseller_level_id": item.ResellerLevelID}
		if err = appendAudit(ctx, tx, scope, "commission_rule.create", "commission_rule", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "commission_rule", item.ID, "commission_rule.created", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) LockCommission(ctx context.Context, scope tenancy.Scope, input application.CommissionLockInput, key string) (channel.CommissionLock, error) {
	if input.CommissionID == "" || input.CommissionRuleID == "" || input.ResellerID == "" {
		return channel.CommissionLock{}, platform.Invalid("invalid_commission_lock", "commission, rule, and reseller are required")
	}
	return runCommand(ctx, s, scope, key, "commission_lock.create", func(tx pgx.Tx) (channel.CommissionLock, string, error) {
		var chargeID, beneficiaryType, beneficiaryID, currency, status string
		var amount int64
		var ruleMatches bool
		if err := tx.QueryRow(ctx, `SELECT customer_charge_id,beneficiary_type,beneficiary_id,currency,amount_minor,status FROM commissions WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, input.CommissionID).Scan(&chargeID, &beneficiaryType, &beneficiaryID, &currency, &amount, &status); err != nil {
			return channel.CommissionLock{}, "", err
		}
		if beneficiaryType != "reseller" || beneficiaryID != input.ResellerID || status == "reversed" {
			return channel.CommissionLock{}, "", platform.Invalid("commission_attribution_mismatch", "canonical commission must belong to the reseller and remain payable")
		}
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM commission_rules cr JOIN resellers r ON r.tenant_id=cr.tenant_id AND r.id=$3 WHERE cr.tenant_id=$1 AND cr.id=$2 AND cr.status='active' AND cr.effective_from<=now() AND (cr.effective_until IS NULL OR cr.effective_until>=now()) AND (cr.reseller_id=$3 OR cr.reseller_level_id=r.level_id))`, scope.TenantID, input.CommissionRuleID, input.ResellerID).Scan(&ruleMatches); err != nil {
			return channel.CommissionLock{}, "", err
		}
		if !ruleMatches {
			return channel.CommissionLock{}, "", platform.Invalid("commission_rule_mismatch", "active commission rule does not apply to the reseller")
		}
		item, err := scanCommissionLock(tx.QueryRow(ctx, `INSERT INTO commission_locks (tenant_id,customer_charge_id,reseller_id,commission_rule_id,commission_id,currency,amount_minor,status,idempotency_key,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,'posted',$8,$9) RETURNING id,tenant_id,customer_charge_id,reseller_id,commission_rule_id,COALESCE(commission_id::text,''),currency,amount_minor,status,idempotency_key,created_by,created_at,updated_at`, scope.TenantID, chargeID, input.ResellerID, input.CommissionRuleID, input.CommissionID, currency, amount, key, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"commission_id": item.CommissionID, "reseller_id": item.ResellerID, "amount_minor": item.AmountMinor}
		if err = appendAudit(ctx, tx, scope, "commission_lock.create", "commission_lock", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "commission_lock", item.ID, "commission_lock.created", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) CreateSettlementCycle(ctx context.Context, scope tenancy.Scope, input application.SettlementCycleInput, key string) (channel.SettlementCycle, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.ResellerID == "" || input.Name == "" || input.PeriodStart.IsZero() || input.PeriodEnd.Before(input.PeriodStart) {
		return channel.SettlementCycle{}, platform.Invalid("invalid_settlement_cycle", "reseller, name, and valid period are required")
	}
	return runCommand(ctx, s, scope, key, "settlement_cycle.create", func(tx pgx.Tx) (channel.SettlementCycle, string, error) {
		item, err := scanSettlementCycle(tx.QueryRow(ctx, `INSERT INTO settlement_cycles (tenant_id,reseller_id,name,period_start,period_end,created_by) SELECT $1,id,$3,$4,$5,$6 FROM resellers WHERE tenant_id=$1 AND id=$2 AND status='active' RETURNING id,tenant_id,reseller_id,name,period_start,period_end,status,created_by,created_at,updated_at`, scope.TenantID, input.ResellerID, input.Name, input.PeriodStart, input.PeriodEnd, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"reseller_id": item.ResellerID, "period_start": item.PeriodStart, "period_end": item.PeriodEnd}
		if err = appendAudit(ctx, tx, scope, "settlement_cycle.create", "settlement_cycle", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "settlement_cycle", item.ID, "settlement_cycle.created", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

var _ application.ChannelStore = (*Store)(nil)

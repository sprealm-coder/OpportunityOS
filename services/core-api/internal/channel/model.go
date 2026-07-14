package channel

import (
	"time"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/finance"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/marketplace"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/state"
)

type ResellerLevel struct {
	ID                   string    `json:"id"`
	TenantID             string    `json:"tenant_id"`
	Name                 string    `json:"name"`
	Rank                 int       `json:"rank"`
	DefaultCommissionBPS int       `json:"default_commission_bps"`
	Status               string    `json:"status"`
	CreatedBy            string    `json:"created_by"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type Reseller struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	LevelID   string    `json:"level_id,omitempty"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AttributionRule struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenant_id"`
	Name       string         `json:"name"`
	Priority   int            `json:"priority"`
	Definition map[string]any `json:"definition"`
	Status     string         `json:"status"`
	Version    int            `json:"version"`
	CreatedBy  string         `json:"created_by"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type LeadOwnership struct {
	ID                  string    `json:"id"`
	TenantID            string    `json:"tenant_id"`
	LeadID              string    `json:"lead_id"`
	ResellerID          string    `json:"reseller_id"`
	AttributionRuleID   string    `json:"attribution_rule_id"`
	Status              string    `json:"status"`
	ProtectionExpiresAt time.Time `json:"protection_expires_at"`
	Version             int       `json:"version"`
	CreatedBy           string    `json:"created_by"`
	AcquiredAt          time.Time `json:"acquired_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type CustomerOwnership struct {
	ID                    string    `json:"id"`
	TenantID              string    `json:"tenant_id"`
	CustomerID            string    `json:"customer_id"`
	ResellerID            string    `json:"reseller_id"`
	SourceLeadOwnershipID string    `json:"source_lead_ownership_id,omitempty"`
	Status                string    `json:"status"`
	ProtectionExpiresAt   time.Time `json:"protection_expires_at"`
	Version               int       `json:"version"`
	CreatedBy             string    `json:"created_by"`
	AcquiredAt            time.Time `json:"acquired_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type TransferRequest struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	OwnershipType  string     `json:"ownership_type"`
	OwnershipID    string     `json:"ownership_id"`
	FromResellerID string     `json:"from_reseller_id"`
	ToResellerID   string     `json:"to_reseller_id"`
	Status         string     `json:"status"`
	Rationale      string     `json:"rationale"`
	RequestedBy    string     `json:"requested_by"`
	ReviewedBy     string     `json:"reviewed_by,omitempty"`
	Version        int        `json:"version"`
	CreatedAt      time.Time  `json:"created_at"`
	ReviewedAt     *time.Time `json:"reviewed_at,omitempty"`
}

type ConflictRecord struct {
	ID                  string         `json:"id"`
	TenantID            string         `json:"tenant_id"`
	OwnershipType       string         `json:"ownership_type"`
	OwnershipID         string         `json:"ownership_id"`
	ClaimantResellerIDs []string       `json:"claimant_reseller_ids"`
	Status              string         `json:"status"`
	Resolution          map[string]any `json:"resolution"`
	CreatedBy           string         `json:"created_by"`
	ResolvedBy          string         `json:"resolved_by,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	ResolvedAt          *time.Time     `json:"resolved_at,omitempty"`
}

type CommissionRule struct {
	ID              string     `json:"id"`
	TenantID        string     `json:"tenant_id"`
	Name            string     `json:"name"`
	ResellerID      string     `json:"reseller_id,omitempty"`
	ResellerLevelID string     `json:"reseller_level_id,omitempty"`
	TriggerType     string     `json:"trigger_type"`
	BasisPoints     int        `json:"basis_points"`
	Status          string     `json:"status"`
	Version         int        `json:"version"`
	EffectiveFrom   time.Time  `json:"effective_from"`
	EffectiveUntil  *time.Time `json:"effective_until,omitempty"`
	CreatedBy       string     `json:"created_by"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type CommissionLock struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	CustomerChargeID string    `json:"customer_charge_id"`
	ResellerID       string    `json:"reseller_id"`
	CommissionRuleID string    `json:"commission_rule_id"`
	CommissionID     string    `json:"commission_id,omitempty"`
	Currency         string    `json:"currency"`
	AmountMinor      int64     `json:"amount_minor"`
	Status           string    `json:"status"`
	IdempotencyKey   string    `json:"idempotency_key"`
	CreatedBy        string    `json:"created_by"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type SettlementCycle struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	ResellerID  string    `json:"reseller_id"`
	Name        string    `json:"name"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	Status      string    `json:"status"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Supplier struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SupplierCapability struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	SupplierID   string    `json:"supplier_id"`
	CapabilityID string    `json:"capability_id"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type ProviderSupplierBinding struct {
	ProviderID   string `json:"provider_id"`
	ProviderName string `json:"provider_name"`
	SupplierID   string `json:"supplier_id"`
}

type SupplierContract struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenant_id"`
	SupplierID string         `json:"supplier_id"`
	ProviderID string         `json:"provider_id,omitempty"`
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	Currency   string         `json:"currency"`
	Terms      map[string]any `json:"terms"`
	Version    int            `json:"version"`
	StartsAt   *time.Time     `json:"starts_at,omitempty"`
	EndsAt     *time.Time     `json:"ends_at,omitempty"`
	CreatedBy  string         `json:"created_by"`
	ApprovedBy string         `json:"approved_by,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

func (c *SupplierContract) Transition(to, actorID string) error {
	if err := state.SupplierContract.Transition(c.Status, to); err != nil {
		return err
	}
	c.Status = to
	c.Version++
	c.UpdatedAt = time.Now().UTC()
	if to == "approved" {
		c.ApprovedBy = actorID
	}
	return nil
}

type SupplierRate struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	ContractID   string    `json:"contract_id"`
	CapabilityID string    `json:"capability_id"`
	Unit         string    `json:"unit"`
	RateMinor    int64     `json:"rate_minor"`
	Version      int       `json:"version"`
	Status       string    `json:"status"`
	CreatedBy    string    `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
}

type SupplierQualityRecord struct {
	ID                 string         `json:"id"`
	TenantID           string         `json:"tenant_id"`
	SupplierID         string         `json:"supplier_id"`
	ProviderID         string         `json:"provider_id,omitempty"`
	ProviderEndpointID string         `json:"provider_endpoint_id,omitempty"`
	Metric             string         `json:"metric"`
	ScoreBPS           int            `json:"score_bps"`
	Evidence           map[string]any `json:"evidence"`
	PeriodStart        time.Time      `json:"period_start"`
	PeriodEnd          time.Time      `json:"period_end"`
	CreatedBy          string         `json:"created_by"`
	CreatedAt          time.Time      `json:"created_at"`
}

type Overview struct {
	ResellerLevels       []ResellerLevel           `json:"reseller_levels"`
	Resellers            []Reseller                `json:"resellers"`
	AttributionRules     []AttributionRule         `json:"attribution_rules"`
	LeadOwnerships       []LeadOwnership           `json:"lead_ownerships"`
	CustomerOwnerships   []CustomerOwnership       `json:"customer_ownerships"`
	TransferRequests     []TransferRequest         `json:"transfer_requests"`
	Conflicts            []ConflictRecord          `json:"conflicts"`
	CommissionRules      []CommissionRule          `json:"commission_rules"`
	CommissionLocks      []CommissionLock          `json:"commission_locks"`
	SettlementCycles     []SettlementCycle         `json:"settlement_cycles"`
	Suppliers            []Supplier                `json:"suppliers"`
	SupplierCapabilities []SupplierCapability      `json:"supplier_capabilities"`
	ProviderBindings     []ProviderSupplierBinding `json:"provider_bindings"`
	SupplierContracts    []SupplierContract        `json:"supplier_contracts"`
	SupplierRates        []SupplierRate            `json:"supplier_rates"`
	SupplierQuality      []SupplierQualityRecord   `json:"supplier_quality"`
	ProviderPayables     []finance.ProviderPayable `json:"provider_payables"`
	ResellerSettlements  []finance.Settlement      `json:"reseller_settlements"`
	SupplierSettlements  []finance.Settlement      `json:"supplier_settlements"`
	Marketplace          marketplace.Overview      `json:"marketplace"`
}

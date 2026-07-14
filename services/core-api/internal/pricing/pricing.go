package pricing

import (
	"fmt"
	"math"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
)

type MeteringDefinition struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	Name     string `json:"name"`
	Unit     string `json:"unit"`
	Field    string `json:"field"`
	Version  int    `json:"version"`
}
type Rule struct {
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	FlatMinor     int64  `json:"flat_minor"`
	UnitMinor     int64  `json:"unit_minor"`
	IncludedUnits int64  `json:"included_units"`
}
type PriceBook struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	Currency string `json:"currency"`
	Version  int    `json:"version"`
	Rules    []Rule `json:"rules"`
}

func (m MeteringDefinition) Validate() error {
	if m.TenantID == "" {
		return platform.ErrTenantRequired
	}
	if m.Unit == "" || m.Field == "" {
		return fmt.Errorf("metering unit and field are required")
	}
	return nil
}
func (p PriceBook) Validate() error {
	if len(p.Currency) != 3 {
		return fmt.Errorf("invalid currency")
	}
	if p.Version < 1 || len(p.Rules) == 0 {
		return fmt.Errorf("price book requires a version and rules")
	}
	for _, r := range p.Rules {
		if r.Kind != "flat" && r.Kind != "per_unit" {
			return fmt.Errorf("unsupported price rule %s", r.Kind)
		}
		if r.FlatMinor < 0 || r.UnitMinor < 0 {
			return fmt.Errorf("price amounts cannot be negative")
		}
	}
	return nil
}
func (p PriceBook) Calculate(quantity int64) (platform.Money, error) {
	if err := p.Validate(); err != nil {
		return platform.Money{}, err
	}
	if quantity < 0 {
		return platform.Money{}, fmt.Errorf("quantity cannot be negative")
	}
	var total int64
	for _, rule := range p.Rules {
		var amount int64
		switch rule.Kind {
		case "flat":
			amount = rule.FlatMinor
		case "per_unit":
			billable := quantity - rule.IncludedUnits
			if billable > 0 {
				if rule.UnitMinor > 0 && billable > math.MaxInt64/rule.UnitMinor {
					return platform.Money{}, fmt.Errorf("calculated amount exceeds supported range")
				}
				amount = billable * rule.UnitMinor
			}
		}
		if amount > math.MaxInt64-total {
			return platform.Money{}, fmt.Errorf("calculated amount exceeds supported range")
		}
		total += amount
	}
	return platform.Money{Currency: p.Currency, Minor: total}, nil
}

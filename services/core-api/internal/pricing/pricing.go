package pricing

import (
	"fmt"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
)

type MeteringDefinition struct {
	ID, TenantID, Name, Unit, Field string
	Version                         int
}
type Rule struct {
	ID            string
	Kind          string
	FlatMinor     int64
	UnitMinor     int64
	IncludedUnits int64
}
type PriceBook struct {
	ID, TenantID, Currency string
	Version                int
	Rules                  []Rule
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
		switch rule.Kind {
		case "flat":
			total += rule.FlatMinor
		case "per_unit":
			billable := quantity - rule.IncludedUnits
			if billable > 0 {
				total += billable * rule.UnitMinor
			}
		}
	}
	return platform.Money{Currency: p.Currency, Minor: total}, nil
}

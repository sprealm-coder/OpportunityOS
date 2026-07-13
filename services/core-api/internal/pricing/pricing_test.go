package pricing

import "testing"

func TestPriceBookUsesIntegerMinorUnits(t *testing.T) {
	book := PriceBook{ID: "price", TenantID: "tenant", Currency: "USD", Version: 1, Rules: []Rule{{Kind: "flat", FlatMinor: 500}, {Kind: "per_unit", UnitMinor: 25, IncludedUnits: 2}}}
	charge, err := book.Calculate(6)
	if err != nil {
		t.Fatal(err)
	}
	if charge.Minor != 600 {
		t.Fatalf("expected 600 minor units, got %d", charge.Minor)
	}
}

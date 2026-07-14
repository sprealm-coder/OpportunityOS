package permission

import "testing"

func TestPermissionIsTenantScoped(t *testing.T) {
	auth := New()
	if err := auth.Grant(Grant{TenantID: "tenant-a", ActorID: "actor", Permission: "opportunity.review"}); err != nil {
		t.Fatal(err)
	}
	if err := auth.Require("tenant-a", "actor", "opportunity.review"); err != nil {
		t.Fatal(err)
	}
	if err := auth.Require("tenant-b", "actor", "opportunity.review"); err == nil {
		t.Fatal("cross-tenant permission leak")
	}
}

func TestFinanceRoleSeparation(t *testing.T) {
	if err := RequireRole("operator", FinanceRead); err != nil {
		t.Fatal(err)
	}
	if err := RequireRole("operator", LedgerPost); err != nil {
		t.Fatal(err)
	}
	if err := RequireRole("operator", FinanceAdjust); err == nil {
		t.Fatal("operator received administrative wallet adjustment permission")
	}
	if err := RequireRole("auditor", FinanceRead); err != nil {
		t.Fatal(err)
	}
	if err := RequireRole("auditor", LedgerPost); err == nil {
		t.Fatal("auditor received ledger posting permission")
	}
}

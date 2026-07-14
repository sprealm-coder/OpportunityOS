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

func TestGrowthApprovalSeparation(t *testing.T) {
	for _, allowed := range []string{GrowthRead, GrowthWrite, LeadWrite, ProofWrite, CampaignWrite, OutreachWrite, DealWrite, ExperimentWrite} {
		if err := RequireRole("operator", allowed); err != nil {
			t.Fatalf("operator missing %s: %v", allowed, err)
		}
	}
	for _, denied := range []string{ProofReview, CampaignApprove} {
		if err := RequireRole("operator", denied); err == nil {
			t.Fatalf("operator received approval permission %s", denied)
		}
		if err := RequireRole("reviewer", denied); err != nil {
			t.Fatalf("reviewer missing %s: %v", denied, err)
		}
	}
	if err := RequireRole("auditor", GrowthRead); err != nil {
		t.Fatal(err)
	}
	if err := RequireRole("auditor", LeadWrite); err == nil {
		t.Fatal("auditor received lead write permission")
	}
}

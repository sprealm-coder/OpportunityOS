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

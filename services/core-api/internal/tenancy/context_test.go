package tenancy

import (
	"context"
	"testing"
)

func TestTenantScopeRequired(t *testing.T) {
	if _, err := FromContext(context.Background()); err == nil {
		t.Fatal("expected tenant error")
	}
	scope := Scope{TenantID: "tenant-a", ActorID: "actor-a"}
	actual, err := FromContext(WithScope(context.Background(), scope))
	if err != nil || actual != scope {
		t.Fatalf("unexpected scope: %#v %v", actual, err)
	}
}
func TestCacheKeyContainsTenant(t *testing.T) {
	key, err := CacheKey("tenant-a", "opportunity", "1")
	if err != nil {
		t.Fatal(err)
	}
	if key != "tenant:tenant-a:opportunity:1" {
		t.Fatalf("unexpected key %s", key)
	}
}

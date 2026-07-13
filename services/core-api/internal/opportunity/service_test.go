package opportunity

import (
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/audit"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/outbox"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
	"testing"
)

func TestServiceTenantIsolationAndTransitions(t *testing.T) {
	repo := NewMemoryRepository()
	events := &outbox.Memory{}
	log := &audit.Log{}
	service := NewService(repo, log, events)
	a := tenancy.Scope{TenantID: "tenant-a", ActorID: "actor"}
	b := tenancy.Scope{TenantID: "tenant-b", ActorID: "actor"}
	item, err := service.Create(a, "Neutral opportunity", "Evidence-led hypothesis", "req-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(b, item.ID); err == nil {
		t.Fatal("cross-tenant read should fail")
	}
	item, err = service.AddEvidence(a, item.ID, Evidence{Kind: "observation", Summary: "Neutral evidence", Confidence: 80}, "req-2")
	if err != nil || item.Status != "enriched" {
		t.Fatalf("evidence: %#v %v", item, err)
	}
	item, err = service.Score(a, item.ID, 75, "req-3")
	if err != nil || item.Status != "scored" {
		t.Fatalf("score: %#v %v", item, err)
	}
	if len(log.ForTenant("tenant-a")) != 3 {
		t.Fatal("expected three audit records")
	}
	if len(events.Pending("tenant-a")) == 0 {
		t.Fatal("expected outbox events")
	}
}

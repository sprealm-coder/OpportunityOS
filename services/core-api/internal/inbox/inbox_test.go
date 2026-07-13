package inbox

import "testing"

func TestInboxDeduplicatesPerTenant(t *testing.T) {
	box := New()
	event := Event{TenantID: "tenant", ExternalEventID: "external-1", IdempotencyKey: "idem-1"}
	accepted, err := box.Accept(event)
	if err != nil || !accepted {
		t.Fatalf("first event rejected: %v", err)
	}
	accepted, err = box.Accept(event)
	if err != nil || accepted {
		t.Fatal("duplicate event accepted")
	}
	event.TenantID = "other"
	accepted, err = box.Accept(event)
	if err != nil || !accepted {
		t.Fatal("other tenant should have an independent keyspace")
	}
}

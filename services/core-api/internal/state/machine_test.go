package state

import "testing"

func TestRequiredStateMachines(t *testing.T) {
	tests := []struct {
		name     string
		machine  Machine
		from, to string
	}{{"opportunity", Opportunity, "under_review", "approved"}, {"incubation", Incubation, "validating", "approved"}, {"blueprint", Blueprint, "configuring", "ready"}, {"product", Product, "ready", "published"}, {"lead", Lead, "proposal", "won"}, {"quote", Quote, "sent", "accepted"}, {"order", Order, "paid", "provisioning"}, {"execution", Execution, "processing", "succeeded"}, {"delivery", Delivery, "in_progress", "completed"}, {"listing", Listing, "sandbox_testing", "limited_release"}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.machine.Transition(test.from, test.to); err != nil {
				t.Fatal(err)
			}
			if err := test.machine.Transition(test.to, test.from); err == nil {
				t.Fatalf("expected reverse transition to fail")
			}
		})
	}
}

package workflow

import (
	"context"
	"testing"
)

func testDefinition() Definition {
	return Definition{ID: "wf-1", TenantID: "tenant-a", Name: "Test Workflow", Version: 1, Nodes: []Node{{ID: "start", Type: Start}, {ID: "validate", Type: Validate}, {ID: "meter", Type: Meter}, {ID: "end", Type: End}}, Edges: []Edge{{From: "start", To: "validate"}, {From: "validate", To: "meter"}, {From: "meter", To: "end"}}}
}
func TestEngineExecutesAndDeduplicates(t *testing.T) {
	engine := NewEngine()
	definition := testDefinition()
	run, err := engine.Execute(context.Background(), definition, "idem-1", map[string]any{"units": int64(2)})
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "succeeded" || len(run.Steps) != 4 {
		t.Fatalf("unexpected run %#v", run)
	}
	duplicate, err := engine.Execute(context.Background(), definition, "idem-1", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.ID != run.ID {
		t.Fatal("idempotent run was duplicated")
	}
}
func TestWorkflowRejectsCycles(t *testing.T) {
	definition := testDefinition()
	definition.Edges = append(definition.Edges, Edge{From: "end", To: "start"})
	if err := definition.Validate(); err == nil {
		t.Fatal("expected cycle validation error")
	}
}

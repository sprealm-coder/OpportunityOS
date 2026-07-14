package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/application"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/opportunity"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

func TestApplicationStorePersistsCommercialControlChain(t *testing.T) {
	ctx := context.Background()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://opportunity:opportunity@localhost:5432/opportunity?sslmode=disable"
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	var tenantID string
	tenantName := fmt.Sprintf("Store Test %d", time.Now().UnixNano())
	if err := pool.QueryRow(ctx, `INSERT INTO tenants(name) VALUES($1) RETURNING id`, tenantName).Scan(&tenantID); err != nil {
		t.Fatal(err)
	}
	defer cleanupApplicationTenant(t, pool, tenantID)

	store := NewStore(pool)
	scope := tenancy.Scope{TenantID: tenantID, ActorID: "store-test", TraceID: "trace-store-test"}
	created, err := store.CreateOpportunity(ctx, scope, "Persistent Opportunity", "Repository integration", "create")
	mustStore(t, err)
	replayed, err := store.CreateOpportunity(ctx, scope, "Ignored duplicate", "Ignored", "create")
	mustStore(t, err)
	if replayed.ID != created.ID {
		t.Fatalf("idempotency replay created another opportunity: %s != %s", replayed.ID, created.ID)
	}

	item, err := store.AddEvidence(ctx, scope, created.ID, opportunity.Evidence{Kind: "customer_interview", Summary: "Validated demand", Confidence: 90}, "evidence")
	mustStore(t, err)
	if item.Status != "enriched" || len(item.Evidence) != 1 {
		t.Fatalf("unexpected evidence result: %#v", item)
	}
	item, err = store.ScoreOpportunity(ctx, scope, created.ID, 84, "score")
	mustStore(t, err)
	item, err = store.TransitionOpportunity(ctx, scope, created.ID, "under_review", "review-start")
	mustStore(t, err)
	item, err = store.ReviewOpportunity(ctx, scope, created.ID, "approved", "Evidence and economics meet the gate", "review")
	mustStore(t, err)
	if item.Status != "approved" {
		t.Fatalf("review status=%s", item.Status)
	}

	project, err := store.CreateIncubation(ctx, scope, created.ID, "Persistent Incubation", "incubate")
	mustStore(t, err)
	project, err = store.TransitionIncubation(ctx, scope, project.ID, "researching", "incubation-research")
	mustStore(t, err)
	if project.Status != "researching" {
		t.Fatalf("incubation status=%s", project.Status)
	}

	bp, err := store.CreateBlueprint(ctx, scope, created.ID, application.BlueprintInput{
		Name: "Persistent Blueprint", ValueProposition: "Verified value",
		RequiredCapabilities: []string{"Test Capability"},
		ProductDefinitions:   []map[string]any{{"name": "Test Product"}},
		WorkflowDefinitions:  []map[string]any{{"name": "Test Workflow"}},
		MeteringDefinitions:  []map[string]any{{"unit": "test_unit"}},
		PricingDefinitions:   []map[string]any{{"currency": "USD"}},
		ComplianceProfile:    map[string]any{"classification": "test"},
	}, "blueprint")
	mustStore(t, err)
	for index, next := range []string{"analyzing", "validating", "approved"} {
		bp, err = store.TransitionBlueprint(ctx, scope, bp.ID, next, fmt.Sprintf("blueprint-transition-%d", index))
		mustStore(t, err)
	}
	if bp.Status != "approved" || bp.ApprovedBy != scope.ActorID {
		t.Fatalf("unexpected blueprint approval: %#v", bp)
	}

	auditRecords, err := store.ListAudit(ctx, scope)
	mustStore(t, err)
	if len(auditRecords) < 9 {
		t.Fatalf("expected persisted audit chain, got %d records", len(auditRecords))
	}
	var outboxCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1`, tenantID).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if outboxCount < 9 {
		t.Fatalf("expected persisted outbox chain, got %d events", outboxCount)
	}

	var otherTenant string
	if err := pool.QueryRow(ctx, `INSERT INTO tenants(name) VALUES($1) RETURNING id`, tenantName+" Other").Scan(&otherTenant); err != nil {
		t.Fatal(err)
	}
	defer cleanupApplicationTenant(t, pool, otherTenant)
	if _, err := store.GetOpportunity(ctx, tenancy.Scope{TenantID: otherTenant, ActorID: "other"}, created.ID); err == nil {
		t.Fatal("cross-tenant opportunity read succeeded")
	}
}

func cleanupApplicationTenant(t *testing.T, pool *pgxpool.Pool, tenantID string) {
	t.Helper()
	ctx := context.Background()
	tables := []string{
		"command_idempotency", "opportunity_reviews", "business_blueprints",
		"incubation_projects", "opportunity_evidence", "audit_log", "outbox_events",
		"opportunities", "brands", "memberships",
	}
	for _, table := range tables {
		if _, err := pool.Exec(ctx, `DELETE FROM `+table+` WHERE tenant_id=$1`, tenantID); err != nil {
			t.Errorf("cleanup %s: %v", table, err)
		}
	}
	if _, err := pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1`, tenantID); err != nil {
		t.Errorf("cleanup tenant: %v", err)
	}
}

func mustStore(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

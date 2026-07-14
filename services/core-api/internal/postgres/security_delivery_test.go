package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/auth"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/inbox"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://opportunity:opportunity@localhost:5432/opportunity?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestDatabaseRoleEnforcesTenantRLS(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	tenantA := createSecurityTenant(t, pool, "RLS A")
	tenantB := createSecurityTenant(t, pool, "RLS B")
	defer cleanupSecurityTenant(t, pool, tenantA)
	defer cleanupSecurityTenant(t, pool, tenantB)
	var opportunityID string
	if err := pool.QueryRow(ctx, `INSERT INTO opportunities(tenant_id,name,status,created_by) VALUES($1,'RLS Opportunity','detected','test') RETURNING id`, tenantA).Scan(&opportunityID); err != nil {
		t.Fatal(err)
	}
	tx, err := beginTenantTx(ctx, pool, tenantB)
	if err != nil {
		t.Fatal(err)
	}
	var visible int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM opportunities WHERE id=$1`, opportunityID).Scan(&visible); err != nil {
		t.Fatal(err)
	}
	if visible != 0 {
		t.Fatalf("tenant B saw tenant A record through RLS: %d", visible)
	}
	if err = tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}

	tx, err = beginTenantTx(ctx, pool, tenantB)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO opportunities(tenant_id,name,status,created_by) VALUES($1,'Cross Tenant','detected','test')`, tenantA); err == nil {
		_ = tx.Rollback(ctx)
		t.Fatal("RLS allowed a cross-tenant insert")
	}
	_ = tx.Rollback(ctx)
}

func TestPersistentSessionOutboxAndInbox(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	tenantID := createSecurityTenant(t, pool, "Delivery")
	var userID string
	defer func() {
		cleanupSecurityTenant(t, pool, tenantID)
		if userID != "" {
			_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID)
		}
	}()
	passwordHash, err := auth.HashPassword("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	email := fmt.Sprintf("security-%d@example.test", time.Now().UnixNano())
	if err = pool.QueryRow(ctx, `INSERT INTO users(email,display_name,password_hash) VALUES($1,'Security Test',$2) RETURNING id`, email, passwordHash).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO memberships(tenant_id,user_id,role) VALUES($1,$2,'operator')`, tenantID, userID); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool)
	if _, err = store.CreateSession(ctx, email, "wrong-password"); err == nil {
		t.Fatal("invalid password created a session")
	}
	session, err := store.CreateSession(ctx, email, "correct-password")
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := store.ResolveSession(ctx, session.Token)
	if err != nil || resolved.TenantID != tenantID || resolved.Role != "operator" {
		t.Fatalf("unexpected resolved session: %#v err=%v", resolved, err)
	}
	if err = store.RevokeSession(ctx, session.Token); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ResolveSession(ctx, session.Token); err == nil {
		t.Fatal("revoked session resolved")
	}

	var eventID string
	if err = pool.QueryRow(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,aggregate_version,trace_id,payload) VALUES($1,'test','aggregate','test.created',1,'trace','{"ok":true}') RETURNING id`, tenantID).Scan(&eventID); err != nil {
		t.Fatal(err)
	}
	events, err := store.LeaseOutbox(ctx, "worker-a", 1000, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	var leased bool
	for _, event := range events {
		if event.ID == eventID {
			leased = true
		}
	}
	if !leased {
		t.Fatalf("new outbox event was not leased: %#v", events)
	}
	secondLease, err := store.LeaseOutbox(ctx, "worker-b", 100, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range secondLease {
		if event.ID == eventID {
			t.Fatal("active outbox lease was acquired by another worker")
		}
	}
	if err = store.MarkOutboxPublished(ctx, "worker-b", eventID); err == nil {
		t.Fatal("worker without the lease marked the event published")
	}
	if err = store.MarkOutboxPublished(ctx, "worker-a", eventID); err != nil {
		t.Fatal(err)
	}

	scope := tenancy.Scope{TenantID: tenantID, ActorID: userID}
	message := inbox.Event{ExternalEventID: "external-1", IdempotencyKey: "idem-1", Payload: map[string]any{"ok": true}}
	accepted, err := store.AcceptInbox(ctx, scope, message)
	if err != nil || !accepted {
		t.Fatalf("first inbox event was not accepted: accepted=%v err=%v", accepted, err)
	}
	accepted, err = store.AcceptInbox(ctx, scope, message)
	if err != nil || accepted {
		t.Fatalf("duplicate inbox event was accepted: accepted=%v err=%v", accepted, err)
	}
}

func createSecurityTenant(t *testing.T, pool *pgxpool.Pool, prefix string) string {
	t.Helper()
	var tenantID string
	if err := pool.QueryRow(context.Background(), `INSERT INTO tenants(name) VALUES($1) RETURNING id`, fmt.Sprintf("%s %d", prefix, time.Now().UnixNano())).Scan(&tenantID); err != nil {
		t.Fatal(err)
	}
	return tenantID
}

func cleanupSecurityTenant(t *testing.T, pool *pgxpool.Pool, tenantID string) {
	t.Helper()
	ctx := context.Background()
	for _, table := range []string{"auth_sessions", "inbox_events", "outbox_events", "opportunities", "memberships"} {
		if _, err := pool.Exec(ctx, `DELETE FROM `+table+` WHERE tenant_id=$1`, tenantID); err != nil {
			t.Errorf("cleanup %s: %v", table, err)
		}
	}
	if _, err := pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1`, tenantID); err != nil {
		t.Errorf("cleanup tenant: %v", err)
	}
}

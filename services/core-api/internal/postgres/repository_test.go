package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTenantScopeAndTransactionalOutbox(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://opportunity:opportunity@localhost:5432/opportunity?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var tenantA, tenantB string
	if err := tx.QueryRow(context.Background(), "INSERT INTO tenants(name) VALUES('Repository Test A ' || gen_random_uuid()) RETURNING id").Scan(&tenantA); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(context.Background(), "INSERT INTO tenants(name) VALUES('Repository Test B ' || gen_random_uuid()) RETURNING id").Scan(&tenantB); err != nil {
		t.Fatal(err)
	}
	var opportunityID string
	if err := tx.QueryRow(context.Background(), "INSERT INTO opportunities(tenant_id,name,status,created_by) VALUES($1,'Test Opportunity','detected','test-actor') RETURNING id", tenantA).Scan(&opportunityID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(context.Background(), "INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,aggregate_version,trace_id,payload) VALUES($1,'opportunity',$2,'opportunity.created',1,'test-trace','{}')", tenantA, opportunityID); err != nil {
		t.Fatal(err)
	}
	var countA, countB int
	if err := tx.QueryRow(context.Background(), "SELECT count(*) FROM opportunities WHERE tenant_id=$1 AND id=$2", tenantA, opportunityID).Scan(&countA); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(context.Background(), "SELECT count(*) FROM opportunities WHERE tenant_id=$1 AND id=$2", tenantB, opportunityID).Scan(&countB); err != nil {
		t.Fatal(err)
	}
	if countA != 1 || countB != 0 {
		t.Fatalf("tenant scope failed: tenantA=%d tenantB=%d", countA, countB)
	}
	if err := tx.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	var persisted int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM opportunities WHERE id=$1", opportunityID).Scan(&persisted); err != nil {
		t.Fatal(err)
	}
	if persisted != 0 {
		t.Fatal("business record persisted after transaction rollback")
	}
	var outboxPersisted int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM outbox_events WHERE aggregate_id=$1", opportunityID).Scan(&outboxPersisted); err != nil {
		t.Fatal(err)
	}
	if outboxPersisted != 0 {
		t.Fatal("outbox event persisted independently of business transaction")
	}
}

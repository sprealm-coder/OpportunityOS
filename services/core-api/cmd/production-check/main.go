package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	postgresstore "github.com/opportunity-os/opportunity-os/services/core-api/internal/postgres"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

type check struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	tenantID := os.Getenv("PRODUCTION_CHECK_TENANT_ID")
	checks := []check{
		require("app_environment", os.Getenv("APP_ENV") == "production", "APP_ENV must be production"),
		require("secure_session_cookie", strings.EqualFold(os.Getenv("SESSION_COOKIE_SECURE"), "true"), "SESSION_COOKIE_SECURE must be true"),
		require("database_tls", strings.Contains(databaseURL, "sslmode=require") || strings.Contains(databaseURL, "sslmode=verify-"), "DATABASE_URL must require TLS"),
		require("tenant_scope", tenantID != "", "PRODUCTION_CHECK_TENANT_ID is required"),
	}
	if databaseURL == "" || tenantID == "" {
		finish(checks)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		checks = append(checks, check{Name: "database", Status: "failed", Message: err.Error()})
		finish(checks)
	}
	defer pool.Close()
	if err = pool.Ping(ctx); err != nil {
		checks = append(checks, check{Name: "database", Status: "failed", Message: err.Error()})
		finish(checks)
	}
	checks = append(checks, check{Name: "database", Status: "passed", Message: "database connection succeeded"})
	overview, err := postgresstore.NewStore(pool).ListOperations(ctx, tenancy.Scope{TenantID: tenantID, ActorID: "production-check", Role: "admin", TraceID: "production-check"})
	if err != nil {
		checks = append(checks, check{Name: "operations", Status: "failed", Message: err.Error()})
		finish(checks)
	}
	for _, item := range overview.DeploymentChecks {
		checks = append(checks, check{Name: item.Name, Status: item.Status, Message: item.Message})
	}
	activeAlerts := 0
	for _, alert := range overview.Alerts {
		if alert.Status != "resolved" {
			activeAlerts++
		}
	}
	checks = append(checks, require("outbox_dead_letter", overview.Outbox.DeadLetter == 0, fmt.Sprintf("dead-letter events: %d", overview.Outbox.DeadLetter)))
	checks = append(checks, require("operational_alerts", activeAlerts == 0, fmt.Sprintf("active operational alerts: %d", activeAlerts)))
	finish(checks)
}

func require(name string, passed bool, message string) check {
	status := "failed"
	if passed {
		status = "passed"
	}
	return check{Name: name, Status: status, Message: message}
}

func finish(checks []check) {
	encoded, _ := json.MarshalIndent(map[string]any{"checks": checks}, "", "  ")
	fmt.Println(string(encoded))
	for _, item := range checks {
		if item.Status != "passed" {
			os.Exit(1)
		}
	}
}

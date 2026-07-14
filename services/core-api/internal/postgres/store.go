package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	generateddb "github.com/opportunity-os/opportunity-os/services/core-api/generated/db"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/application"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/audit"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/blueprint"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/incubation"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/opportunity"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/state"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

type Store struct {
	pool    *pgxpool.Pool
	queries *generateddb.Queries
}

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool, queries: generateddb.New(pool)} }

func runCommand[T any](ctx context.Context, store *Store, scope tenancy.Scope, key, operation string, command func(pgx.Tx) (T, string, error)) (T, error) {
	var zero T
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return zero, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	response, replay, err := reserveCommand(ctx, tx, scope.TenantID, key, operation)
	if err != nil {
		return zero, mapError(err)
	}
	if replay {
		var result T
		if err := json.Unmarshal(response, &result); err != nil {
			return zero, fmt.Errorf("decode idempotent response: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return zero, err
		}
		return result, nil
	}

	result, aggregateID, err := command(tx)
	if err != nil {
		return zero, mapError(err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return zero, err
	}
	if _, err = tx.Exec(ctx, `UPDATE command_idempotency SET aggregate_id=$3, response=$4 WHERE tenant_id=$1 AND idempotency_key=$2`, scope.TenantID, key, nullableUUID(aggregateID), encoded); err != nil {
		return zero, mapError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return zero, mapError(err)
	}
	return result, nil
}

func reserveCommand(ctx context.Context, tx pgx.Tx, tenantID, key, operation string) (json.RawMessage, bool, error) {
	var inserted string
	err := tx.QueryRow(ctx, `INSERT INTO command_idempotency (tenant_id,idempotency_key,operation) VALUES($1,$2,$3) ON CONFLICT DO NOTHING RETURNING idempotency_key`, tenantID, key, operation).Scan(&inserted)
	if err == nil {
		return nil, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}
	var existingOperation string
	var response []byte
	if err := tx.QueryRow(ctx, `SELECT operation,response FROM command_idempotency WHERE tenant_id=$1 AND idempotency_key=$2`, tenantID, key).Scan(&existingOperation, &response); err != nil {
		return nil, false, err
	}
	if existingOperation != operation {
		return nil, false, platform.Invalid("idempotency_conflict", "idempotency key was used for another command")
	}
	if len(response) == 0 {
		return nil, false, platform.Invalid("command_incomplete", "previous command has not completed")
	}
	return response, true, nil
}

func nullableUUID(id string) any {
	if id == "" {
		return nil
	}
	return id
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return platform.Invalid("not_found", "record not found")
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "22P02":
			return platform.Invalid("invalid_id", "identifier must be a UUID")
		case "23503":
			return platform.Invalid("invalid_reference", "referenced tenant or object does not exist")
		case "23505":
			return platform.Invalid("conflict", "record already exists")
		}
	}
	return err
}

func appendAudit(ctx context.Context, tx pgx.Tx, scope tenancy.Scope, action, objectType, objectID, requestID string, metadata map[string]any) error {
	if metadata == nil {
		metadata = map[string]any{}
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_log (tenant_id,actor_id,action,object_type,object_id,request_id,trace_id,metadata) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, scope.TenantID, scope.ActorID, action, objectType, objectID, requestID, scope.TraceID, encoded)
	return err
}

func appendEvent(ctx context.Context, tx pgx.Tx, scope tenancy.Scope, aggregateType, aggregateID, eventType string, version int, payload map[string]any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events (tenant_id,aggregate_type,aggregate_id,event_type,aggregate_version,trace_id,payload) VALUES($1,$2,$3,$4,$5,$6,$7)`, scope.TenantID, aggregateType, aggregateID, eventType, version, scope.TraceID, encoded)
	return err
}

type rowScanner interface{ Scan(...any) error }

func scanOpportunity(row rowScanner) (opportunity.Opportunity, error) {
	var item opportunity.Opportunity
	err := row.Scan(&item.ID, &item.TenantID, &item.Name, &item.Description, &item.Status, &item.Score, &item.Version, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func parseUUID(value string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(value); err != nil {
		return pgtype.UUID{}, platform.Invalid("invalid_id", "identifier must be a UUID")
	}
	return id, nil
}

func uuidString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	b := value.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func opportunityFromDB(item generateddb.Opportunity) opportunity.Opportunity {
	return opportunity.Opportunity{
		ID: uuidString(item.ID), TenantID: uuidString(item.TenantID), Name: item.Name,
		Description: item.Description, Status: item.Status, Score: int(item.Score), Version: int(item.Version),
		CreatedBy: item.CreatedBy, CreatedAt: item.CreatedAt.Time, UpdatedAt: item.UpdatedAt.Time,
		Evidence: []opportunity.Evidence{},
	}
}

func evidenceFromDB(item generateddb.OpportunityEvidence) opportunity.Evidence {
	return opportunity.Evidence{
		ID: uuidString(item.ID), TenantID: uuidString(item.TenantID), OpportunityID: uuidString(item.OpportunityID),
		Kind: item.Kind, Summary: item.Summary, Confidence: int(item.Confidence), CreatedAt: item.CreatedAt.Time,
	}
}

func (s *Store) sqlcEvidence(ctx context.Context, tenantID, opportunityID string) ([]opportunity.Evidence, error) {
	tenantUUID, err := parseUUID(tenantID)
	if err != nil {
		return nil, err
	}
	opportunityUUID, err := parseUUID(opportunityID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListOpportunityEvidence(ctx, generateddb.ListOpportunityEvidenceParams{TenantID: tenantUUID, OpportunityID: opportunityUUID})
	if err != nil {
		return nil, mapError(err)
	}
	items := make([]opportunity.Evidence, 0, len(rows))
	for _, row := range rows {
		items = append(items, evidenceFromDB(row))
	}
	return items, nil
}

func loadEvidence(ctx context.Context, queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, tenantID, opportunityID string) ([]opportunity.Evidence, error) {
	rows, err := queryer.Query(ctx, `SELECT id,tenant_id,opportunity_id,kind,summary,confidence,created_at FROM opportunity_evidence WHERE tenant_id=$1 AND opportunity_id=$2 ORDER BY created_at,id`, tenantID, opportunityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []opportunity.Evidence{}
	for rows.Next() {
		var item opportunity.Evidence
		if err := rows.Scan(&item.ID, &item.TenantID, &item.OpportunityID, &item.Kind, &item.Summary, &item.Confidence, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func getOpportunity(ctx context.Context, tx pgx.Tx, tenantID, id string, lock bool) (opportunity.Opportunity, error) {
	query := `SELECT id,tenant_id,name,description,status,score,version,created_by,created_at,updated_at FROM opportunities WHERE tenant_id=$1 AND id=$2`
	if lock {
		query += ` FOR UPDATE`
	}
	item, err := scanOpportunity(tx.QueryRow(ctx, query, tenantID, id))
	if err != nil {
		return item, err
	}
	item.Evidence, err = loadEvidence(ctx, tx, tenantID, id)
	return item, err
}

func (s *Store) CreateOpportunity(ctx context.Context, scope tenancy.Scope, name, description, key string) (opportunity.Opportunity, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return opportunity.Opportunity{}, platform.Invalid("invalid_name", "opportunity name is required")
	}
	return runCommand(ctx, s, scope, key, "opportunity.create", func(tx pgx.Tx) (opportunity.Opportunity, string, error) {
		item, err := scanOpportunity(tx.QueryRow(ctx, `INSERT INTO opportunities (tenant_id,name,description,status,created_by) VALUES($1,$2,$3,'detected',$4) RETURNING id,tenant_id,name,description,status,score,version,created_by,created_at,updated_at`, scope.TenantID, name, description, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		if err = appendAudit(ctx, tx, scope, "opportunity.create", "opportunity", item.ID, key, nil); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "opportunity", item.ID, "opportunity.created", item.Version, map[string]any{"status": item.Status}); err != nil {
			return item, "", err
		}
		item.Evidence = []opportunity.Evidence{}
		return item, item.ID, nil
	})
}

func (s *Store) ListOpportunities(ctx context.Context, scope tenancy.Scope) ([]opportunity.Opportunity, error) {
	tenantID, err := parseUUID(scope.TenantID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListOpportunities(ctx, generateddb.ListOpportunitiesParams{TenantID: tenantID, Limit: 200})
	if err != nil {
		return nil, mapError(err)
	}
	items := make([]opportunity.Opportunity, 0, len(rows))
	for _, row := range rows {
		item := opportunityFromDB(row)
		item.Evidence, err = s.sqlcEvidence(ctx, scope.TenantID, item.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) GetOpportunity(ctx context.Context, scope tenancy.Scope, id string) (opportunity.Opportunity, error) {
	tenantID, err := parseUUID(scope.TenantID)
	if err != nil {
		return opportunity.Opportunity{}, err
	}
	opportunityID, err := parseUUID(id)
	if err != nil {
		return opportunity.Opportunity{}, err
	}
	row, err := s.queries.GetOpportunity(ctx, generateddb.GetOpportunityParams{TenantID: tenantID, ID: opportunityID})
	if err != nil {
		return opportunity.Opportunity{}, mapError(err)
	}
	item := opportunityFromDB(row)
	item.Evidence, err = s.sqlcEvidence(ctx, scope.TenantID, id)
	return item, err
}

func (s *Store) AddEvidence(ctx context.Context, scope tenancy.Scope, id string, evidence opportunity.Evidence, key string) (opportunity.Opportunity, error) {
	evidence.Kind = strings.TrimSpace(evidence.Kind)
	evidence.Summary = strings.TrimSpace(evidence.Summary)
	if evidence.Kind == "" || evidence.Summary == "" || evidence.Confidence < 0 || evidence.Confidence > 100 {
		return opportunity.Opportunity{}, platform.Invalid("invalid_evidence", "kind, summary, and confidence from 0 to 100 are required")
	}
	return runCommand(ctx, s, scope, key, "opportunity.add_evidence", func(tx pgx.Tx) (opportunity.Opportunity, string, error) {
		item, err := getOpportunity(ctx, tx, scope.TenantID, id, true)
		if err != nil {
			return item, "", err
		}
		var added opportunity.Evidence
		err = tx.QueryRow(ctx, `INSERT INTO opportunity_evidence (tenant_id,opportunity_id,kind,summary,confidence) VALUES($1,$2,$3,$4,$5) RETURNING id,tenant_id,opportunity_id,kind,summary,confidence,created_at`, scope.TenantID, id, evidence.Kind, evidence.Summary, evidence.Confidence).Scan(&added.ID, &added.TenantID, &added.OpportunityID, &added.Kind, &added.Summary, &added.Confidence, &added.CreatedAt)
		if err != nil {
			return item, "", err
		}
		if item.Status == "detected" {
			if err = state.Opportunity.Transition(item.Status, "enriched"); err != nil {
				return item, "", err
			}
			item.Status = "enriched"
		}
		item.Version++
		item, err = updateOpportunity(ctx, tx, scope.TenantID, item)
		if err != nil {
			return item, "", err
		}
		item.Evidence = append(item.Evidence, added)
		if err = appendAudit(ctx, tx, scope, "opportunity.add_evidence", "opportunity", id, key, map[string]any{"evidence_id": added.ID}); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "opportunity", id, "opportunity.evidence_added", item.Version, map[string]any{"evidence_id": added.ID, "status": item.Status}); err != nil {
			return item, "", err
		}
		return item, id, nil
	})
}

func updateOpportunity(ctx context.Context, tx pgx.Tx, tenantID string, item opportunity.Opportunity) (opportunity.Opportunity, error) {
	updated, err := scanOpportunity(tx.QueryRow(ctx, `UPDATE opportunities SET status=$3,score=$4,version=$5,updated_at=now() WHERE tenant_id=$1 AND id=$2 RETURNING id,tenant_id,name,description,status,score,version,created_by,created_at,updated_at`, tenantID, item.ID, item.Status, item.Score, item.Version))
	if err == nil {
		updated.Evidence = item.Evidence
	}
	return updated, err
}

func (s *Store) ScoreOpportunity(ctx context.Context, scope tenancy.Scope, id string, score int, key string) (opportunity.Opportunity, error) {
	if score < 0 || score > 100 {
		return opportunity.Opportunity{}, platform.Invalid("invalid_score", "score must be between 0 and 100")
	}
	return runCommand(ctx, s, scope, key, "opportunity.score", func(tx pgx.Tx) (opportunity.Opportunity, string, error) {
		item, err := getOpportunity(ctx, tx, scope.TenantID, id, true)
		if err != nil {
			return item, "", err
		}
		if item.Status != "enriched" {
			return item, "", platform.Invalid("score_not_ready", "opportunity must be enriched before scoring")
		}
		if err = state.Opportunity.Transition(item.Status, "scored"); err != nil {
			return item, "", err
		}
		item.Status, item.Score, item.Version = "scored", score, item.Version+1
		item, err = updateOpportunity(ctx, tx, scope.TenantID, item)
		if err != nil {
			return item, "", err
		}
		if err = appendAudit(ctx, tx, scope, "opportunity.score", "opportunity", id, key, map[string]any{"score": score}); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "opportunity", id, "opportunity.scored", item.Version, map[string]any{"score": score}); err != nil {
			return item, "", err
		}
		return item, id, nil
	})
}

func (s *Store) TransitionOpportunity(ctx context.Context, scope tenancy.Scope, id, to, key string) (opportunity.Opportunity, error) {
	return runCommand(ctx, s, scope, key, "opportunity.transition", func(tx pgx.Tx) (opportunity.Opportunity, string, error) {
		item, err := getOpportunity(ctx, tx, scope.TenantID, id, true)
		if err != nil {
			return item, "", err
		}
		from := item.Status
		if err = state.Opportunity.Transition(from, to); err != nil {
			return item, "", err
		}
		item.Status, item.Version = to, item.Version+1
		item, err = updateOpportunity(ctx, tx, scope.TenantID, item)
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"from": from, "to": to}
		if err = appendAudit(ctx, tx, scope, "opportunity.transition", "opportunity", id, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "opportunity", id, "opportunity.transitioned", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, id, nil
	})
}

func (s *Store) ReviewOpportunity(ctx context.Context, scope tenancy.Scope, id, decision, rationale, key string) (opportunity.Opportunity, error) {
	decision, rationale = strings.TrimSpace(decision), strings.TrimSpace(rationale)
	if decision != "approved" && decision != "rejected" {
		return opportunity.Opportunity{}, platform.Invalid("invalid_review_decision", "decision must be approved or rejected")
	}
	if rationale == "" {
		return opportunity.Opportunity{}, platform.Invalid("invalid_rationale", "review rationale is required")
	}
	return runCommand(ctx, s, scope, key, "opportunity.review", func(tx pgx.Tx) (opportunity.Opportunity, string, error) {
		item, err := getOpportunity(ctx, tx, scope.TenantID, id, true)
		if err != nil {
			return item, "", err
		}
		if item.Status != "under_review" {
			return item, "", platform.Invalid("review_not_ready", "opportunity must be under review")
		}
		if err = state.Opportunity.Transition(item.Status, decision); err != nil {
			return item, "", err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO opportunity_reviews (tenant_id,opportunity_id,actor_id,decision,rationale) VALUES($1,$2,$3,$4,$5)`, scope.TenantID, id, scope.ActorID, decision, rationale); err != nil {
			return item, "", err
		}
		item.Status, item.Version = decision, item.Version+1
		item, err = updateOpportunity(ctx, tx, scope.TenantID, item)
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"decision": decision, "rationale": rationale}
		if err = appendAudit(ctx, tx, scope, "opportunity.review", "opportunity", id, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "opportunity", id, "opportunity.reviewed", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, id, nil
	})
}

func (s *Store) ListAudit(ctx context.Context, scope tenancy.Scope) ([]audit.Record, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,tenant_id,actor_id,action,object_type,object_id,request_id,trace_id,metadata,created_at FROM audit_log WHERE tenant_id=$1 ORDER BY created_at DESC,id LIMIT 200`, scope.TenantID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	items := []audit.Record{}
	for rows.Next() {
		var item audit.Record
		var metadata []byte
		if err := rows.Scan(&item.ID, &item.TenantID, &item.ActorID, &item.Action, &item.ObjectType, &item.ObjectID, &item.RequestID, &item.TraceID, &metadata, &item.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(metadata, &item.Metadata)
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanIncubation(row rowScanner) (incubation.Project, error) {
	var item incubation.Project
	err := row.Scan(&item.ID, &item.TenantID, &item.OpportunityID, &item.Name, &item.Status, &item.Version, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) CreateIncubation(ctx context.Context, scope tenancy.Scope, opportunityID, name, key string) (incubation.Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return incubation.Project{}, platform.Invalid("invalid_name", "incubation name is required")
	}
	return runCommand(ctx, s, scope, key, "incubation.create", func(tx pgx.Tx) (incubation.Project, string, error) {
		opp, err := getOpportunity(ctx, tx, scope.TenantID, opportunityID, true)
		if err != nil {
			return incubation.Project{}, "", err
		}
		if opp.Status != "approved" {
			return incubation.Project{}, "", platform.Invalid("incubation_not_ready", "opportunity must be approved")
		}
		if err = state.Opportunity.Transition(opp.Status, "incubating"); err != nil {
			return incubation.Project{}, "", err
		}
		opp.Status, opp.Version = "incubating", opp.Version+1
		opp, err = updateOpportunity(ctx, tx, scope.TenantID, opp)
		if err != nil {
			return incubation.Project{}, "", err
		}
		project, err := scanIncubation(tx.QueryRow(ctx, `INSERT INTO incubation_projects (tenant_id,opportunity_id,name,status) VALUES($1,$2,$3,'draft') RETURNING id,tenant_id,opportunity_id,name,status,version,created_at,updated_at`, scope.TenantID, opportunityID, name))
		if err != nil {
			return project, "", err
		}
		if err = appendAudit(ctx, tx, scope, "incubation.create", "incubation_project", project.ID, key, map[string]any{"opportunity_id": opportunityID}); err != nil {
			return project, "", err
		}
		if err = appendEvent(ctx, tx, scope, "incubation_project", project.ID, "incubation.created", project.Version, map[string]any{"opportunity_id": opportunityID}); err != nil {
			return project, "", err
		}
		if err = appendEvent(ctx, tx, scope, "opportunity", opportunityID, "opportunity.transitioned", opp.Version, map[string]any{"from": "approved", "to": "incubating"}); err != nil {
			return project, "", err
		}
		return project, project.ID, nil
	})
}

func (s *Store) ListIncubations(ctx context.Context, scope tenancy.Scope) ([]incubation.Project, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,tenant_id,opportunity_id,name,status,version,created_at,updated_at FROM incubation_projects WHERE tenant_id=$1 ORDER BY updated_at DESC,id LIMIT 200`, scope.TenantID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	items := []incubation.Project{}
	for rows.Next() {
		item, err := scanIncubation(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) TransitionIncubation(ctx context.Context, scope tenancy.Scope, id, to, key string) (incubation.Project, error) {
	return runCommand(ctx, s, scope, key, "incubation.transition", func(tx pgx.Tx) (incubation.Project, string, error) {
		item, err := scanIncubation(tx.QueryRow(ctx, `SELECT id,tenant_id,opportunity_id,name,status,version,created_at,updated_at FROM incubation_projects WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, id))
		if err != nil {
			return item, "", err
		}
		from := item.Status
		if err = item.Transition(to); err != nil {
			return item, "", err
		}
		item, err = scanIncubation(tx.QueryRow(ctx, `UPDATE incubation_projects SET status=$3,version=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2 RETURNING id,tenant_id,opportunity_id,name,status,version,created_at,updated_at`, scope.TenantID, id, item.Status, item.Version))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"from": from, "to": to}
		if err = appendAudit(ctx, tx, scope, "incubation.transition", "incubation_project", id, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "incubation_project", id, "incubation.transitioned", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, id, nil
	})
}

func scanBlueprint(row rowScanner) (blueprint.BusinessBlueprint, error) {
	var item blueprint.BusinessBlueprint
	var definition []byte
	var approvedBy pgtype.Text
	var id, tenantID, opportunityID, name, description, status, createdBy string
	var version int
	var createdAt, updatedAt time.Time
	err := row.Scan(&id, &tenantID, &opportunityID, &name, &description, &version, &status, &definition, &createdBy, &approvedBy, &createdAt, &updatedAt)
	if err != nil {
		return item, err
	}
	if len(definition) > 0 {
		if err := json.Unmarshal(definition, &item); err != nil {
			return item, err
		}
	}
	item.ID = id
	item.TenantID = tenantID
	item.SourceOpportunityID = opportunityID
	item.Name = name
	item.Description = description
	item.Version = version
	item.Status = status
	item.CreatedBy = createdBy
	item.CreatedAt = createdAt
	item.UpdatedAt = updatedAt
	if approvedBy.Valid {
		item.ApprovedBy = approvedBy.String
	} else {
		item.ApprovedBy = ""
	}
	return item, nil
}

func (s *Store) CreateBlueprint(ctx context.Context, scope tenancy.Scope, opportunityID string, input application.BlueprintInput, key string) (blueprint.BusinessBlueprint, error) {
	item := application.BuildBlueprint(scope, opportunityID, input)
	if strings.TrimSpace(item.Name) == "" {
		return blueprint.BusinessBlueprint{}, platform.Invalid("invalid_name", "blueprint name is required")
	}
	return runCommand(ctx, s, scope, key, "blueprint.create", func(tx pgx.Tx) (blueprint.BusinessBlueprint, string, error) {
		if _, err := getOpportunity(ctx, tx, scope.TenantID, opportunityID, false); err != nil {
			return item, "", err
		}
		definition, err := json.Marshal(item)
		if err != nil {
			return item, "", err
		}
		item, err = scanBlueprint(tx.QueryRow(ctx, `INSERT INTO business_blueprints (tenant_id,source_opportunity_id,name,description,version,status,definition,created_by) VALUES($1,$2,$3,$4,1,'draft',$5,$6) RETURNING id,tenant_id,source_opportunity_id,name,description,version,status,definition,created_by,approved_by,created_at,updated_at`, scope.TenantID, opportunityID, item.Name, item.Description, definition, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		if err = appendAudit(ctx, tx, scope, "blueprint.create", "business_blueprint", item.ID, key, map[string]any{"opportunity_id": opportunityID}); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "business_blueprint", item.ID, "blueprint.created", item.Version, map[string]any{"opportunity_id": opportunityID}); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) ListBlueprints(ctx context.Context, scope tenancy.Scope) ([]blueprint.BusinessBlueprint, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,tenant_id,source_opportunity_id,name,description,version,status,definition,created_by,approved_by,created_at,updated_at FROM business_blueprints WHERE tenant_id=$1 ORDER BY updated_at DESC,id LIMIT 200`, scope.TenantID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	items := []blueprint.BusinessBlueprint{}
	for rows.Next() {
		item, err := scanBlueprint(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) TransitionBlueprint(ctx context.Context, scope tenancy.Scope, id, to, key string) (blueprint.BusinessBlueprint, error) {
	return runCommand(ctx, s, scope, key, "blueprint.transition", func(tx pgx.Tx) (blueprint.BusinessBlueprint, string, error) {
		item, err := scanBlueprint(tx.QueryRow(ctx, `SELECT id,tenant_id,source_opportunity_id,name,description,version,status,definition,created_by,approved_by,created_at,updated_at FROM business_blueprints WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, id))
		if err != nil {
			return item, "", err
		}
		from := item.Status
		if to == "approved" {
			if err = item.ValidateCompleteness(); err != nil {
				return item, "", platform.Invalid("blueprint_incomplete", err.Error())
			}
		}
		if err = item.Transition(to, scope.ActorID); err != nil {
			return item, "", err
		}
		definition, err := json.Marshal(item)
		if err != nil {
			return item, "", err
		}
		item, err = scanBlueprint(tx.QueryRow(ctx, `UPDATE business_blueprints SET status=$3,version=$4,definition=$5,approved_by=$6,updated_at=now() WHERE tenant_id=$1 AND id=$2 RETURNING id,tenant_id,source_opportunity_id,name,description,version,status,definition,created_by,approved_by,created_at,updated_at`, scope.TenantID, id, item.Status, item.Version, definition, nullableText(item.ApprovedBy)))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"from": from, "to": to}
		if err = appendAudit(ctx, tx, scope, "blueprint.transition", "business_blueprint", id, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "business_blueprint", id, "blueprint.transitioned", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, id, nil
	})
}

func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}

var _ application.Store = (*Store)(nil)

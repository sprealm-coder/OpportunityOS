package postgres

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/analytics"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/application"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/intelligence"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/operations"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/opportunity"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/workflow"
)

const adapterClockTolerance = 5 * time.Minute

func decodeObject(raw []byte) map[string]any {
	result := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &result)
	}
	return result
}

func scanSource(row rowScanner) (intelligence.Source, error) {
	var item intelligence.Source
	var config []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.Name, &item.ConnectorType, &item.Status, &config, &item.Version, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	item.Config = decodeObject(config)
	return item, err
}

func scanSignal(row rowScanner) (intelligence.Signal, error) {
	var item intelligence.Signal
	var payload, normalized []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.SourceID, &item.OpportunityID, &item.ExternalID, &item.Fingerprint, &item.Status, &payload, &normalized, &item.OccurredAt, &item.ImportedBy, &item.CreatedAt, &item.UpdatedAt)
	item.Payload, item.Normalized = decodeObject(payload), decodeObject(normalized)
	return item, err
}

func (s *Store) ListIntelligence(ctx context.Context, scope tenancy.Scope) (intelligence.Overview, error) {
	return runTenantRead(ctx, s, scope.TenantID, func(tx pgx.Tx) (intelligence.Overview, error) {
		result := intelligence.Overview{Sources: []intelligence.Source{}, Signals: []intelligence.Signal{}}
		rows, err := tx.Query(ctx, `SELECT id,tenant_id,name,connector_type,status,config,version,created_by,created_at,updated_at FROM sources WHERE tenant_id=$1 ORDER BY created_at DESC`, scope.TenantID)
		if err != nil {
			return result, err
		}
		for rows.Next() {
			item, scanErr := scanSource(rows)
			if scanErr != nil {
				rows.Close()
				return result, scanErr
			}
			result.Sources = append(result.Sources, item)
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return result, err
		}
		rows.Close()
		rows, err = tx.Query(ctx, `
			SELECT signal.id,signal.tenant_id,signal.source_id,COALESCE(link.opportunity_id::text,''),COALESCE(signal.external_id,''),
			       signal.fingerprint,signal.status,signal.payload,signal.normalized,signal.occurred_at,signal.imported_by,signal.created_at,signal.updated_at
			FROM signals signal LEFT JOIN opportunity_signals link ON link.tenant_id=signal.tenant_id AND link.signal_id=signal.id
			WHERE signal.tenant_id=$1 ORDER BY signal.created_at DESC LIMIT 500`, scope.TenantID)
		if err != nil {
			return result, err
		}
		defer rows.Close()
		for rows.Next() {
			item, scanErr := scanSignal(rows)
			if scanErr != nil {
				return result, scanErr
			}
			result.Signals = append(result.Signals, item)
		}
		return result, rows.Err()
	})
}

func (s *Store) CreateSource(ctx context.Context, scope tenancy.Scope, input application.SourceInput, key string) (intelligence.Source, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.ConnectorType = strings.TrimSpace(input.ConnectorType)
	if input.Name == "" || input.ConnectorType == "" {
		return intelligence.Source{}, platform.Invalid("invalid_source", "source name and connector type are required")
	}
	if input.Config == nil {
		input.Config = map[string]any{}
	}
	config, err := json.Marshal(input.Config)
	if err != nil {
		return intelligence.Source{}, platform.Invalid("invalid_source_config", err.Error())
	}
	return runCommand(ctx, s, scope, key, "source.create", func(tx pgx.Tx) (intelligence.Source, string, error) {
		item, createErr := scanSource(tx.QueryRow(ctx, `
			INSERT INTO sources (tenant_id,name,connector_type,config,created_by)
			VALUES($1,$2,$3,$4,$5)
			RETURNING id,tenant_id,name,connector_type,status,config,version,created_by,created_at,updated_at`, scope.TenantID, input.Name, input.ConnectorType, config, scope.ActorID))
		if createErr != nil {
			return item, "", createErr
		}
		metadata := map[string]any{"connector_type": item.ConnectorType}
		if createErr = appendAudit(ctx, tx, scope, "source.create", "source", item.ID, key, metadata); createErr != nil {
			return item, "", createErr
		}
		if createErr = appendEvent(ctx, tx, scope, "source", item.ID, "source.created", item.Version, metadata); createErr != nil {
			return item, "", createErr
		}
		return item, item.ID, nil
	})
}

func signalFingerprint(sourceID string, payload []byte) string {
	digest := sha256.Sum256(append([]byte(sourceID+"\n"), payload...))
	return hex.EncodeToString(digest[:])
}

func (s *Store) ImportSignal(ctx context.Context, scope tenancy.Scope, sourceID string, input application.SignalInput, key string) (intelligence.Signal, error) {
	if sourceID == "" || len(input.Payload) == 0 {
		return intelligence.Signal{}, platform.Invalid("invalid_signal", "source and non-empty payload are required")
	}
	if input.Normalized == nil {
		input.Normalized = map[string]any{}
	}
	payload, err := json.Marshal(input.Payload)
	if err != nil {
		return intelligence.Signal{}, platform.Invalid("invalid_signal_payload", err.Error())
	}
	normalized, err := json.Marshal(input.Normalized)
	if err != nil {
		return intelligence.Signal{}, platform.Invalid("invalid_signal_normalized", err.Error())
	}
	fingerprint := strings.ToLower(strings.TrimSpace(input.Fingerprint))
	if fingerprint == "" {
		fingerprint = signalFingerprint(sourceID, payload)
	}
	if input.OccurredAt.IsZero() {
		input.OccurredAt = time.Now().UTC()
	}
	return runCommand(ctx, s, scope, key, "signal.import", func(tx pgx.Tx) (intelligence.Signal, string, error) {
		var active bool
		if queryErr := tx.QueryRow(ctx, `SELECT status='active' FROM sources WHERE tenant_id=$1 AND id=$2`, scope.TenantID, sourceID).Scan(&active); queryErr != nil {
			return intelligence.Signal{}, "", queryErr
		}
		if !active {
			return intelligence.Signal{}, "", platform.Invalid("source_inactive", "signals can only be imported from an active source")
		}
		var id string
		insertErr := tx.QueryRow(ctx, `
			INSERT INTO signals (tenant_id,source_id,external_id,fingerprint,payload,normalized,status,occurred_at,imported_by)
			VALUES($1,$2,NULLIF($3,''),$4,$5,$6,CASE WHEN $6::jsonb='{}'::jsonb THEN 'imported' ELSE 'normalized' END,$7,$8)
			ON CONFLICT (tenant_id,fingerprint) DO NOTHING RETURNING id`, scope.TenantID, sourceID, strings.TrimSpace(input.ExternalID), fingerprint, payload, normalized, input.OccurredAt.UTC(), scope.ActorID).Scan(&id)
		created := insertErr == nil
		if errors.Is(insertErr, pgx.ErrNoRows) {
			insertErr = tx.QueryRow(ctx, `SELECT id FROM signals WHERE tenant_id=$1 AND fingerprint=$2`, scope.TenantID, fingerprint).Scan(&id)
		}
		if insertErr != nil {
			return intelligence.Signal{}, "", insertErr
		}
		item, scanErr := scanSignal(tx.QueryRow(ctx, `
			SELECT signal.id,signal.tenant_id,signal.source_id,COALESCE(link.opportunity_id::text,''),COALESCE(signal.external_id,''),signal.fingerprint,signal.status,signal.payload,signal.normalized,signal.occurred_at,signal.imported_by,signal.created_at,signal.updated_at
			FROM signals signal LEFT JOIN opportunity_signals link ON link.tenant_id=signal.tenant_id AND link.signal_id=signal.id
			WHERE signal.tenant_id=$1 AND signal.id=$2`, scope.TenantID, id))
		if scanErr != nil {
			return item, "", scanErr
		}
		metadata := map[string]any{"source_id": sourceID, "fingerprint": fingerprint, "deduplicated": !created}
		if scanErr = appendAudit(ctx, tx, scope, "signal.import", "signal", item.ID, key, metadata); scanErr != nil {
			return item, "", scanErr
		}
		if created {
			scanErr = appendEvent(ctx, tx, scope, "signal", item.ID, "signal.imported", 1, metadata)
		}
		return item, item.ID, scanErr
	})
}

func (s *Store) PromoteSignal(ctx context.Context, scope tenancy.Scope, signalID string, input application.SignalPromotionInput, key string) (opportunity.Opportunity, error) {
	input.Name, input.Description, input.Summary = strings.TrimSpace(input.Name), strings.TrimSpace(input.Description), strings.TrimSpace(input.Summary)
	if input.Name == "" || input.Summary == "" || input.Confidence < 0 || input.Confidence > 100 {
		return opportunity.Opportunity{}, platform.Invalid("invalid_signal_promotion", "name, summary, and confidence between 0 and 100 are required")
	}
	return runCommand(ctx, s, scope, key, "signal.promote", func(tx pgx.Tx) (opportunity.Opportunity, string, error) {
		var status string
		if err := tx.QueryRow(ctx, `SELECT status FROM signals WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, signalID).Scan(&status); err != nil {
			return opportunity.Opportunity{}, "", err
		}
		if status == "promoted" {
			return opportunity.Opportunity{}, "", platform.Invalid("signal_already_promoted", "signal already belongs to an opportunity")
		}
		item, err := scanOpportunity(tx.QueryRow(ctx, `
			INSERT INTO opportunities (tenant_id,name,description,status,created_by)
			VALUES($1,$2,$3,'enriched',$4)
			RETURNING id,tenant_id,name,description,status,score,version,created_by,created_at,updated_at`, scope.TenantID, input.Name, input.Description, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		var evidence opportunity.Evidence
		err = tx.QueryRow(ctx, `
			INSERT INTO opportunity_evidence (tenant_id,opportunity_id,kind,summary,confidence)
			VALUES($1,$2,'signal',$3,$4)
			RETURNING id,tenant_id,opportunity_id,kind,summary,confidence,created_at`, scope.TenantID, item.ID, input.Summary, input.Confidence).
			Scan(&evidence.ID, &evidence.TenantID, &evidence.OpportunityID, &evidence.Kind, &evidence.Summary, &evidence.Confidence, &evidence.CreatedAt)
		if err != nil {
			return item, "", err
		}
		item.Evidence = []opportunity.Evidence{evidence}
		if _, err = tx.Exec(ctx, `INSERT INTO opportunity_signals (tenant_id,opportunity_id,signal_id,linked_by) VALUES($1,$2,$3,$4)`, scope.TenantID, item.ID, signalID, scope.ActorID); err != nil {
			return item, "", err
		}
		if _, err = tx.Exec(ctx, `UPDATE signals SET status='promoted',updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, signalID); err != nil {
			return item, "", err
		}
		metadata := map[string]any{"signal_id": signalID, "evidence_id": evidence.ID}
		if err = appendAudit(ctx, tx, scope, "signal.promote", "opportunity", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "opportunity", item.ID, "opportunity.created_from_signal", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func scanOutcomeFeedback(row rowScanner) (analytics.OutcomeFeedback, error) {
	var item analytics.OutcomeFeedback
	var metrics, evidence []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.OpportunityID, &item.OrderID, &item.ExecutionOrderID, &item.Status, &metrics, &evidence, &item.IdempotencyKey, &item.Version, &item.CreatedBy, &item.ValidatedAt, &item.CreatedAt)
	item.Metrics, item.Evidence = decodeObject(metrics), decodeObject(evidence)
	return item, err
}

func (s *Store) ListAnalytics(ctx context.Context, scope tenancy.Scope) (analytics.Overview, error) {
	return runTenantRead(ctx, s, scope.TenantID, func(tx pgx.Tx) (analytics.Overview, error) {
		result := analytics.Overview{Feedback: []analytics.OutcomeFeedback{}, Projections: []analytics.OpportunityProjection{}}
		rows, err := tx.Query(ctx, `SELECT id,tenant_id,opportunity_id,order_id,COALESCE(execution_order_id::text,''),status,metrics,evidence,COALESCE(idempotency_key,''),version,created_by,validated_at,created_at FROM outcome_feedback WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT 200`, scope.TenantID)
		if err != nil {
			return result, err
		}
		for rows.Next() {
			item, scanErr := scanOutcomeFeedback(rows)
			if scanErr != nil {
				rows.Close()
				return result, scanErr
			}
			result.Feedback = append(result.Feedback, item)
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return result, err
		}
		rows.Close()
		rows, err = tx.Query(ctx, `SELECT id,outcome_feedback_count,outcome_metrics,outcome_updated_at FROM opportunities WHERE tenant_id=$1 AND outcome_feedback_count>0 ORDER BY outcome_updated_at DESC`, scope.TenantID)
		if err != nil {
			return result, err
		}
		defer rows.Close()
		for rows.Next() {
			var projection analytics.OpportunityProjection
			var metrics []byte
			if err = rows.Scan(&projection.OpportunityID, &projection.FeedbackCount, &metrics, &projection.UpdatedAt); err != nil {
				return result, err
			}
			projection.LatestMetrics = decodeObject(metrics)
			result.Projections = append(result.Projections, projection)
		}
		return result, rows.Err()
	})
}

func (s *Store) RecordOutcomeFeedback(ctx context.Context, scope tenancy.Scope, input application.OutcomeFeedbackInput, key string) (analytics.OutcomeFeedback, error) {
	if input.OpportunityID == "" || input.OrderID == "" {
		return analytics.OutcomeFeedback{}, platform.Invalid("invalid_outcome_feedback", "opportunity and order are required")
	}
	if input.Metrics == nil {
		input.Metrics = map[string]any{}
	}
	if input.Evidence == nil {
		input.Evidence = map[string]any{}
	}
	return runCommand(ctx, s, scope, key, "outcome_feedback.record", func(tx pgx.Tx) (analytics.OutcomeFeedback, string, error) {
		var orderStatus, currency string
		if err := tx.QueryRow(ctx, `SELECT status,trim(currency) FROM orders WHERE tenant_id=$1 AND id=$2 FOR SHARE`, scope.TenantID, input.OrderID).Scan(&orderStatus, &currency); err != nil {
			return analytics.OutcomeFeedback{}, "", err
		}
		if orderStatus != "completed" {
			return analytics.OutcomeFeedback{}, "", platform.Invalid("order_not_completed", "outcome feedback requires a completed order")
		}
		var lineage bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS(
			  SELECT 1 FROM order_items item
			  JOIN product_versions version ON version.tenant_id=item.tenant_id AND version.id=item.product_version_id
			  JOIN products product ON product.tenant_id=version.tenant_id AND product.id=version.product_id
			  JOIN business_blueprints blueprint ON blueprint.tenant_id=product.tenant_id AND blueprint.id=product.blueprint_id
			  WHERE item.tenant_id=$1 AND item.order_id=$2 AND blueprint.source_opportunity_id=$3
			)`, scope.TenantID, input.OrderID, input.OpportunityID).Scan(&lineage); err != nil {
			return analytics.OutcomeFeedback{}, "", err
		}
		if !lineage {
			return analytics.OutcomeFeedback{}, "", platform.Invalid("outcome_lineage_mismatch", "order does not originate from the opportunity blueprint")
		}
		if input.ExecutionOrderID != "" {
			var settled bool
			if err := tx.QueryRow(ctx, `SELECT status='settled' FROM execution_orders WHERE tenant_id=$1 AND id=$2 AND order_id=$3`, scope.TenantID, input.ExecutionOrderID, input.OrderID).Scan(&settled); err != nil {
				return analytics.OutcomeFeedback{}, "", err
			}
			if !settled {
				return analytics.OutcomeFeedback{}, "", platform.Invalid("execution_not_settled", "feedback execution must be settled")
			}
		}
		var incomplete int
		if err := tx.QueryRow(ctx, `
			SELECT
			  (SELECT count(*) FROM execution_orders WHERE tenant_id=$1 AND order_id=$2 AND status<>'settled')
			+ (SELECT count(*) FROM customer_charges WHERE tenant_id=$1 AND order_id=$2 AND status NOT IN ('posted','reversed'))
			+ (SELECT count(*) FROM provider_costs cost LEFT JOIN provider_payables payable ON payable.tenant_id=cost.tenant_id AND payable.provider_cost_id=cost.id WHERE cost.tenant_id=$1 AND cost.order_id=$2 AND (payable.id IS NULL OR payable.status<>'settled'))
			+ (SELECT count(*) FROM commissions commission JOIN customer_charges charge ON charge.tenant_id=commission.tenant_id AND charge.id=commission.customer_charge_id WHERE commission.tenant_id=$1 AND charge.order_id=$2 AND commission.status<>'settled')`, scope.TenantID, input.OrderID).Scan(&incomplete); err != nil {
			return analytics.OutcomeFeedback{}, "", err
		}
		if incomplete != 0 {
			return analytics.OutcomeFeedback{}, "", platform.Invalid("financial_settlement_incomplete", "executions, charges, payables, and commissions must be settled")
		}
		var matched bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM reconciliation_runs WHERE tenant_id=$1 AND order_id=$2 AND status='matched')`, scope.TenantID, input.OrderID).Scan(&matched); err != nil {
			return analytics.OutcomeFeedback{}, "", err
		}
		if !matched {
			return analytics.OutcomeFeedback{}, "", platform.Invalid("reconciliation_required", "a matched reconciliation is required before feedback")
		}
		var chargeMinor, refundMinor, providerCostMinor, commissionMinor int64
		if err := tx.QueryRow(ctx, `
			SELECT
			  COALESCE((SELECT sum(amount_minor) FROM customer_charges WHERE tenant_id=$1 AND order_id=$2 AND status='posted'),0),
			  COALESCE((SELECT sum(refund.amount_minor) FROM refunds refund JOIN customer_charges charge ON charge.tenant_id=refund.tenant_id AND charge.id=refund.customer_charge_id WHERE refund.tenant_id=$1 AND charge.order_id=$2 AND refund.status='posted'),0),
			  COALESCE((SELECT sum(amount_minor) FROM provider_costs WHERE tenant_id=$1 AND order_id=$2),0),
			  COALESCE((SELECT sum(commission.amount_minor) FROM commissions commission JOIN customer_charges charge ON charge.tenant_id=commission.tenant_id AND charge.id=commission.customer_charge_id WHERE commission.tenant_id=$1 AND charge.order_id=$2 AND commission.status<>'reversed'),0)`, scope.TenantID, input.OrderID).Scan(&chargeMinor, &refundMinor, &providerCostMinor, &commissionMinor); err != nil {
			return analytics.OutcomeFeedback{}, "", err
		}
		metrics := map[string]any{}
		for name, value := range input.Metrics {
			metrics[name] = value
		}
		metrics["currency"] = currency
		metrics["customer_charge_minor"] = chargeMinor
		metrics["refund_minor"] = refundMinor
		metrics["net_revenue_minor"] = chargeMinor - refundMinor
		metrics["provider_cost_minor"] = providerCostMinor
		metrics["commission_minor"] = commissionMinor
		metrics["gross_margin_minor"] = chargeMinor - refundMinor - providerCostMinor - commissionMinor
		metrics["order_completed"] = true
		metricsJSON, err := json.Marshal(metrics)
		if err != nil {
			return analytics.OutcomeFeedback{}, "", err
		}
		evidenceJSON, err := json.Marshal(input.Evidence)
		if err != nil {
			return analytics.OutcomeFeedback{}, "", err
		}
		var version int
		if err = tx.QueryRow(ctx, `SELECT COALESCE(max(version),0)+1 FROM outcome_feedback WHERE tenant_id=$1 AND opportunity_id=$2 AND order_id=$3`, scope.TenantID, input.OpportunityID, input.OrderID).Scan(&version); err != nil {
			return analytics.OutcomeFeedback{}, "", err
		}
		item, err := scanOutcomeFeedback(tx.QueryRow(ctx, `
			INSERT INTO outcome_feedback (tenant_id,opportunity_id,order_id,execution_order_id,status,metrics,evidence,idempotency_key,version,created_by)
			VALUES($1,$2,$3,NULLIF($4,'')::uuid,'accepted',$5,$6,$7,$8,$9)
			RETURNING id,tenant_id,opportunity_id,order_id,COALESCE(execution_order_id::text,''),status,metrics,evidence,idempotency_key,version,created_by,validated_at,created_at`, scope.TenantID, input.OpportunityID, input.OrderID, input.ExecutionOrderID, metricsJSON, evidenceJSON, key, version, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		if _, err = tx.Exec(ctx, `UPDATE opportunities SET outcome_feedback_count=outcome_feedback_count+1,outcome_metrics=$3,outcome_updated_at=now(),updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, input.OpportunityID, metricsJSON); err != nil {
			return item, "", err
		}
		metadata := map[string]any{"order_id": input.OrderID, "version": item.Version, "gross_margin_minor": metrics["gross_margin_minor"]}
		if err = appendAudit(ctx, tx, scope, "outcome_feedback.record", "outcome_feedback", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "opportunity", input.OpportunityID, "opportunity.outcome_recorded", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func scanAdapterIdentity(row rowScanner) (operations.AdapterIdentity, error) {
	var item operations.AdapterIdentity
	err := row.Scan(&item.ID, &item.TenantID, &item.Name, &item.KeyID, &item.ProviderEndpointID, &item.SecretRef, &item.Status, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func validAdapterSecretRef(value string) bool {
	const prefix = "OPPORTUNITY_ADAPTER_SECRET_"
	if !strings.HasPrefix(value, prefix) || len(value) == len(prefix) {
		return false
	}
	for _, character := range strings.TrimPrefix(value, prefix) {
		if (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '_' {
			return false
		}
	}
	return true
}

func (s *Store) RegisterAdapterIdentity(ctx context.Context, scope tenancy.Scope, input application.AdapterIdentityInput, key string) (operations.AdapterIdentity, error) {
	input.Name, input.KeyID, input.SecretRef = strings.TrimSpace(input.Name), strings.TrimSpace(input.KeyID), strings.TrimSpace(input.SecretRef)
	if input.Name == "" || input.KeyID == "" || input.ProviderEndpointID == "" || !validAdapterSecretRef(input.SecretRef) {
		return operations.AdapterIdentity{}, platform.Invalid("invalid_adapter_identity", "name, key ID, provider endpoint, and an OPPORTUNITY_ADAPTER_SECRET_ environment reference are required")
	}
	return runCommand(ctx, s, scope, key, "adapter_identity.register", func(tx pgx.Tx) (operations.AdapterIdentity, string, error) {
		var healthy bool
		if err := tx.QueryRow(ctx, `SELECT status='healthy' FROM provider_endpoints WHERE tenant_id=$1 AND id=$2`, scope.TenantID, input.ProviderEndpointID).Scan(&healthy); err != nil {
			return operations.AdapterIdentity{}, "", err
		}
		if !healthy {
			return operations.AdapterIdentity{}, "", platform.Invalid("provider_unavailable", "adapter identity requires a healthy provider endpoint")
		}
		item, err := scanAdapterIdentity(tx.QueryRow(ctx, `
			INSERT INTO adapter_identities (tenant_id,name,key_id,provider_endpoint_id,secret_ref,created_by)
			VALUES($1,$2,$3,$4,$5,$6)
			RETURNING id,tenant_id,name,key_id,provider_endpoint_id,secret_ref,status,created_by,created_at,updated_at`, scope.TenantID, input.Name, input.KeyID, input.ProviderEndpointID, input.SecretRef, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"key_id": item.KeyID, "provider_endpoint_id": item.ProviderEndpointID, "secret_ref": item.SecretRef}
		if err = appendAudit(ctx, tx, scope, "adapter_identity.register", "adapter_identity", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "adapter_identity", item.ID, "adapter_identity.registered", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func workflowSideEffectNodes(definition workflow.Definition) []workflow.Node {
	result := []workflow.Node{}
	for _, node := range definition.Nodes {
		switch node.Type {
		case workflow.RealtimeCall, workflow.AsyncSubmit, workflow.AsyncWait, workflow.Provision, workflow.ManualTask, workflow.WebhookWait:
			result = append(result, node)
		}
	}
	if len(result) == 0 {
		result = append(result, workflow.Node{ID: "execute", Type: workflow.RealtimeCall})
	}
	return result
}

func scanWorkflowStep(row rowScanner) (operations.WorkflowStep, error) {
	var item operations.WorkflowStep
	var output, stepError []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.RunID, &item.ExecutionOrderID, &item.NodeID, &item.NodeType, &item.Status, &item.Attempt, &item.MaxAttempts, &item.LockedBy, &item.LockedUntil, &item.NextRetryAt, &output, &stepError, &item.StartedAt, &item.CompletedAt)
	item.Output, item.Error = decodeObject(output), decodeObject(stepError)
	return item, err
}

func getWorkflowRun(ctx context.Context, tx pgx.Tx, tenantID, runID string) (operations.WorkflowRun, error) {
	var item operations.WorkflowRun
	var variables []byte
	err := tx.QueryRow(ctx, `SELECT id,tenant_id,definition_id,COALESCE(execution_order_id::text,''),idempotency_key,status,variables,created_by,started_at,completed_at,created_at,updated_at FROM workflow_runs WHERE tenant_id=$1 AND id=$2`, tenantID, runID).
		Scan(&item.ID, &item.TenantID, &item.DefinitionID, &item.ExecutionOrderID, &item.IdempotencyKey, &item.Status, &variables, &item.CreatedBy, &item.StartedAt, &item.CompletedAt, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	item.Variables, item.Steps = decodeObject(variables), []operations.WorkflowStep{}
	rows, err := tx.Query(ctx, `SELECT id,tenant_id,run_id,COALESCE(execution_order_id::text,''),node_id,node_type,status,attempt,max_attempts,COALESCE(locked_by,''),locked_until,next_retry_at,COALESCE(output,'{}'),COALESCE(last_error,error,'{}'),started_at,completed_at FROM workflow_step_runs WHERE tenant_id=$1 AND run_id=$2 ORDER BY id`, tenantID, runID)
	if err != nil {
		return item, err
	}
	defer rows.Close()
	for rows.Next() {
		step, scanErr := scanWorkflowStep(rows)
		if scanErr != nil {
			return item, scanErr
		}
		item.Steps = append(item.Steps, step)
	}
	return item, rows.Err()
}

func (s *Store) StartWorkflowRun(ctx context.Context, scope tenancy.Scope, executionID string, input application.WorkflowRunInput, key string) (operations.WorkflowRun, error) {
	if input.MaxAttempts <= 0 {
		input.MaxAttempts = 3
	}
	if input.MaxAttempts > 20 {
		return operations.WorkflowRun{}, platform.Invalid("invalid_max_attempts", "workflow max attempts cannot exceed 20")
	}
	return runCommand(ctx, s, scope, key, "workflow_run.start", func(tx pgx.Tx) (operations.WorkflowRun, string, error) {
		var definitionID, executionStatus string
		var variables []byte
		if err := tx.QueryRow(ctx, `SELECT workflow_version_id,status,input FROM execution_orders WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, executionID).Scan(&definitionID, &executionStatus, &variables); err != nil {
			return operations.WorkflowRun{}, "", err
		}
		if executionStatus != "queued" {
			return operations.WorkflowRun{}, "", platform.Invalid("execution_not_queued", "workflow run requires a queued execution")
		}
		var definitionJSON []byte
		if err := tx.QueryRow(ctx, `SELECT definition FROM workflow_definitions WHERE tenant_id=$1 AND id=$2`, scope.TenantID, definitionID).Scan(&definitionJSON); err != nil {
			return operations.WorkflowRun{}, "", err
		}
		var definition workflow.Definition
		if err := json.Unmarshal(definitionJSON, &definition); err != nil {
			return operations.WorkflowRun{}, "", platform.Invalid("invalid_workflow_definition", err.Error())
		}
		var runID string
		if err := tx.QueryRow(ctx, `
			INSERT INTO workflow_runs (tenant_id,definition_id,execution_order_id,idempotency_key,status,variables,created_by,started_at)
			VALUES($1,$2,$3,$4,'running',$5,$6,now()) RETURNING id`, scope.TenantID, definitionID, executionID, key, variables, scope.ActorID).Scan(&runID); err != nil {
			return operations.WorkflowRun{}, "", err
		}
		for _, node := range workflowSideEffectNodes(definition) {
			if _, err := tx.Exec(ctx, `INSERT INTO workflow_step_runs (tenant_id,run_id,execution_order_id,node_id,node_type,status,attempt,max_attempts) VALUES($1,$2,$3,$4,$5,'pending',0,$6)`, scope.TenantID, runID, executionID, node.ID, string(node.Type), input.MaxAttempts); err != nil {
				return operations.WorkflowRun{}, "", err
			}
		}
		metadata := map[string]any{"execution_order_id": executionID, "definition_id": definitionID, "max_attempts": input.MaxAttempts}
		if err := appendAudit(ctx, tx, scope, "workflow_run.start", "workflow_run", runID, key, metadata); err != nil {
			return operations.WorkflowRun{}, "", err
		}
		if err := appendEvent(ctx, tx, scope, "workflow_run", runID, "workflow_run.started", 1, metadata); err != nil {
			return operations.WorkflowRun{}, "", err
		}
		item, err := getWorkflowRun(ctx, tx, scope.TenantID, runID)
		return item, runID, err
	})
}

func (s *Store) LeaseWorkflowStep(ctx context.Context, scope tenancy.Scope, input application.WorkflowLeaseInput, key string) (operations.WorkflowStep, error) {
	if input.AdapterIdentityID == "" {
		return operations.WorkflowStep{}, platform.Invalid("adapter_identity_required", "adapter identity is required")
	}
	if input.LeaseSeconds <= 0 {
		input.LeaseSeconds = 30
	}
	if input.LeaseSeconds > 300 {
		return operations.WorkflowStep{}, platform.Invalid("invalid_lease", "workflow lease cannot exceed 300 seconds")
	}
	return runCommand(ctx, s, scope, key, "workflow_step.lease", func(tx pgx.Tx) (operations.WorkflowStep, string, error) {
		var active bool
		if err := tx.QueryRow(ctx, `SELECT status='active' FROM adapter_identities WHERE tenant_id=$1 AND id=$2`, scope.TenantID, input.AdapterIdentityID).Scan(&active); err != nil {
			return operations.WorkflowStep{}, "", err
		}
		if !active {
			return operations.WorkflowStep{}, "", platform.Invalid("adapter_identity_inactive", "adapter identity is not active")
		}
		item, err := scanWorkflowStep(tx.QueryRow(ctx, `
			WITH candidate AS (
			  SELECT step.id FROM workflow_step_runs step
			  JOIN execution_orders execution ON execution.tenant_id=step.tenant_id AND execution.id=step.execution_order_id
			  JOIN adapter_identities identity ON identity.tenant_id=step.tenant_id AND identity.id=$2 AND identity.provider_endpoint_id=execution.provider_endpoint_id
			  WHERE step.tenant_id=$1 AND step.status IN ('pending','retry_wait','leased')
			    AND (step.next_retry_at IS NULL OR step.next_retry_at<=now())
			    AND (step.locked_until IS NULL OR step.locked_until<now())
			    AND step.attempt<step.max_attempts
			  ORDER BY step.next_retry_at NULLS FIRST,step.id FOR UPDATE OF step SKIP LOCKED LIMIT 1
			)
			UPDATE workflow_step_runs step SET status='leased',attempt=attempt+1,locked_by=$2::text,locked_until=now()+make_interval(secs=>$3),started_at=COALESCE(started_at,now()),next_retry_at=NULL
			FROM candidate WHERE step.id=candidate.id
			RETURNING step.id,step.tenant_id,step.run_id,COALESCE(step.execution_order_id::text,''),step.node_id,step.node_type,step.status,step.attempt,step.max_attempts,COALESCE(step.locked_by,''),step.locked_until,step.next_retry_at,COALESCE(step.output,'{}'),COALESCE(step.last_error,step.error,'{}'),step.started_at,step.completed_at`, scope.TenantID, input.AdapterIdentityID, input.LeaseSeconds))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"adapter_identity_id": input.AdapterIdentityID, "execution_order_id": item.ExecutionOrderID, "attempt": item.Attempt, "locked_until": item.LockedUntil}
		if err = appendAudit(ctx, tx, scope, "workflow_step.lease", "workflow_step_run", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "workflow_run", item.RunID, "workflow_step.leased", item.Attempt, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func parseAdapterTimestamp(value string) (time.Time, error) {
	if unix, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Unix(unix, 0).UTC(), nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	return parsed.UTC(), err
}

func adapterSignature(secret, timestamp, nonce string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp + "\n" + nonce + "\n"))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func scanAdapterReceipt(row rowScanner) (operations.AdapterReceipt, error) {
	var item operations.AdapterReceipt
	var payload []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.AdapterIdentityID, &item.ExecutionOrderID, &item.WorkflowStepID, &item.ExternalEventID, &item.Nonce, &item.ResultStatus, &payload, &item.ReceivedAt, &item.ProcessedAt)
	item.Payload = decodeObject(payload)
	return item, err
}

func executionProgress(status string) int {
	return map[string]int{"created": 0, "validating": 1, "reserved": 2, "queued": 3, "submitted": 4, "processing": 5, "succeeded": 6, "failed": 6, "reconciling": 7, "settled": 8, "cancelled": 8}[status]
}

func advanceAdapterExecution(ctx context.Context, tx pgx.Tx, scope tenancy.Scope, executionID, current, resultStatus, externalID string, output, resultError map[string]any) (string, error) {
	target := resultStatus
	if target == "unknown" {
		target = "reconciling"
	}
	if executionProgress(current) > executionProgress(target) || current == target {
		return current, nil
	}
	paths := map[string][]string{
		"submitted":   {"submitted"},
		"processing":  {"submitted", "processing"},
		"succeeded":   {"submitted", "processing", "succeeded"},
		"failed":      {"submitted", "failed"},
		"reconciling": {"submitted", "processing", "failed", "reconciling"},
	}
	status := current
	submittedAdvanced := false
	for _, next := range paths[target] {
		if executionProgress(next) <= executionProgress(status) {
			continue
		}
		var allowed bool
		switch status + ">" + next {
		case "reserved>submitted", "queued>submitted", "submitted>processing", "processing>succeeded", "submitted>failed", "processing>failed", "failed>reconciling":
			allowed = true
		}
		if !allowed {
			return status, platform.Invalid("adapter_transition_invalid", fmt.Sprintf("adapter cannot advance execution from %s to %s", status, next))
		}
		if next == "submitted" {
			submittedAdvanced = true
		}
		status = next
	}
	outputJSON, _ := json.Marshal(output)
	errorJSON, _ := json.Marshal(resultError)
	_, err := tx.Exec(ctx, `UPDATE execution_orders SET status=$3,external_id=COALESCE(NULLIF($4,''),external_id),output=CASE WHEN $5::jsonb='null'::jsonb THEN output ELSE $5 END,error=CASE WHEN $6::jsonb='null'::jsonb THEN error ELSE $6 END,attempt=CASE WHEN $7 THEN attempt+1 ELSE attempt END,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, executionID, status, externalID, outputJSON, errorJSON, submittedAdvanced)
	return status, err
}

func (s *Store) IngestAdapterResult(ctx context.Context, request application.AdapterIngressRequest) (operations.AdapterReceipt, error) {
	if request.KeyID == "" || request.Timestamp == "" || len(request.Nonce) < 16 || request.Signature == "" || len(request.Body) == 0 {
		return operations.AdapterReceipt{}, platform.Invalid("adapter_auth_required", "adapter key, timestamp, nonce, signature, and body are required")
	}
	receivedAt, err := parseAdapterTimestamp(request.Timestamp)
	if err != nil || time.Since(receivedAt) > adapterClockTolerance || time.Until(receivedAt) > adapterClockTolerance {
		return operations.AdapterReceipt{}, platform.Invalid("adapter_timestamp_invalid", "adapter timestamp is outside the allowed clock tolerance")
	}
	identity, err := scanAdapterIdentity(s.pool.QueryRow(ctx, `SELECT id,tenant_id,name,key_id,provider_endpoint_id,secret_ref,status,created_by,created_at,updated_at FROM adapter_identities WHERE key_id=$1`, request.KeyID))
	if err != nil || identity.Status != "active" {
		return operations.AdapterReceipt{}, platform.Invalid("adapter_identity_invalid", "adapter identity is unknown or inactive")
	}
	secret := os.Getenv(identity.SecretRef)
	if len(secret) < 32 {
		return operations.AdapterReceipt{}, platform.Invalid("adapter_secret_unavailable", "adapter secret reference is unavailable")
	}
	provided := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(request.Signature)), "sha256=")
	expected := adapterSignature(secret, request.Timestamp, request.Nonce, request.Body)
	if !hmac.Equal([]byte(provided), []byte(expected)) {
		return operations.AdapterReceipt{}, platform.Invalid("adapter_signature_invalid", "adapter signature is invalid")
	}
	var input operations.AdapterResultInput
	if err = json.Unmarshal(request.Body, &input); err != nil {
		return operations.AdapterReceipt{}, platform.Invalid("invalid_adapter_result", err.Error())
	}
	if input.ExternalEventID == "" || input.ExecutionID == "" || !map[string]bool{"submitted": true, "processing": true, "succeeded": true, "failed": true, "unknown": true}[input.Status] {
		return operations.AdapterReceipt{}, platform.Invalid("invalid_adapter_result", "external event ID, execution ID, and a supported result status are required")
	}
	scope := tenancy.Scope{TenantID: identity.TenantID, ActorID: "adapter:" + identity.KeyID, Role: "adapter", TraceID: "adapter:" + input.ExternalEventID}
	tx, err := beginTenantTx(ctx, s.pool, scope.TenantID)
	if err != nil {
		return operations.AdapterReceipt{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	payloadJSON := json.RawMessage(request.Body)
	var inboxID string
	err = tx.QueryRow(ctx, `INSERT INTO inbox_events (tenant_id,external_event_id,idempotency_key,payload,processed_at) VALUES($1,$2,$3,$4,now()) ON CONFLICT DO NOTHING RETURNING id`, scope.TenantID, input.ExternalEventID, "adapter:"+identity.ID+":"+input.ExternalEventID, payloadJSON).Scan(&inboxID)
	if errors.Is(err, pgx.ErrNoRows) {
		item, findErr := scanAdapterReceipt(tx.QueryRow(ctx, `SELECT id,tenant_id,adapter_identity_id,execution_order_id,workflow_step_id,external_event_id,nonce,result_status,payload,received_at,processed_at FROM adapter_result_receipts WHERE tenant_id=$1 AND adapter_identity_id=$2 AND external_event_id=$3`, scope.TenantID, identity.ID, input.ExternalEventID))
		if findErr != nil {
			return operations.AdapterReceipt{}, platform.Invalid("adapter_replay_detected", "adapter nonce or event was already received")
		}
		if err = tx.Commit(ctx); err != nil {
			return operations.AdapterReceipt{}, err
		}
		return item, nil
	}
	if err != nil {
		return operations.AdapterReceipt{}, mapError(err)
	}
	var step operations.WorkflowStep
	step, err = scanWorkflowStep(tx.QueryRow(ctx, `
		SELECT step.id,step.tenant_id,step.run_id,COALESCE(step.execution_order_id::text,''),step.node_id,step.node_type,step.status,step.attempt,step.max_attempts,COALESCE(step.locked_by,''),step.locked_until,step.next_retry_at,COALESCE(step.output,'{}'),COALESCE(step.last_error,step.error,'{}'),step.started_at,step.completed_at
		FROM workflow_step_runs step JOIN execution_orders execution ON execution.tenant_id=step.tenant_id AND execution.id=step.execution_order_id
		WHERE step.tenant_id=$1 AND step.execution_order_id=$2 AND step.locked_by=$3::text AND step.status='leased' AND step.locked_until>=now() AND execution.provider_endpoint_id=$4
		ORDER BY step.started_at DESC LIMIT 1 FOR UPDATE OF step`, scope.TenantID, input.ExecutionID, identity.ID, identity.ProviderEndpointID))
	if err != nil {
		return operations.AdapterReceipt{}, platform.Invalid("workflow_lease_invalid", "adapter does not own an active workflow step lease")
	}
	var currentStatus string
	if err = tx.QueryRow(ctx, `SELECT status FROM execution_orders WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, input.ExecutionID).Scan(&currentStatus); err != nil {
		return operations.AdapterReceipt{}, mapError(err)
	}
	newStatus := currentStatus
	effectiveResultStatus := input.Status
	if input.Status == "succeeded" {
		var unfinishedOtherSteps int
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM workflow_step_runs WHERE tenant_id=$1 AND run_id=$2 AND id<>$3 AND status<>'succeeded'`, scope.TenantID, step.RunID, step.ID).Scan(&unfinishedOtherSteps); err != nil {
			return operations.AdapterReceipt{}, err
		}
		if unfinishedOtherSteps > 0 {
			effectiveResultStatus = "processing"
		}
	}
	if input.Status != "failed" || step.Attempt >= step.MaxAttempts {
		newStatus, err = advanceAdapterExecution(ctx, tx, scope, input.ExecutionID, currentStatus, effectiveResultStatus, input.ExternalID, input.Output, input.Error)
		if err != nil {
			return operations.AdapterReceipt{}, err
		}
	}
	digest := sha256.Sum256(request.Body)
	item, err := scanAdapterReceipt(tx.QueryRow(ctx, `
		INSERT INTO adapter_result_receipts (tenant_id,adapter_identity_id,execution_order_id,workflow_step_id,external_event_id,nonce,signature_digest,result_status,payload,processed_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,now())
		RETURNING id,tenant_id,adapter_identity_id,execution_order_id,workflow_step_id,external_event_id,nonce,result_status,payload,received_at,processed_at`, scope.TenantID, identity.ID, input.ExecutionID, step.ID, input.ExternalEventID, request.Nonce, hex.EncodeToString(digest[:]), input.Status, payloadJSON))
	if err != nil {
		return operations.AdapterReceipt{}, mapError(err)
	}
	outputJSON, _ := json.Marshal(input.Output)
	errorJSON, _ := json.Marshal(input.Error)
	stepStatus, runStatus := "leased", "running"
	var retryAt any
	switch input.Status {
	case "succeeded":
		stepStatus, runStatus = "succeeded", "succeeded"
	case "failed":
		if step.Attempt < step.MaxAttempts {
			stepStatus, runStatus = "retry_wait", "retry_wait"
			delay := time.Duration(1<<min(step.Attempt-1, 8)) * time.Second
			retryAt = time.Now().UTC().Add(delay)
		} else {
			stepStatus, runStatus = "failed", "failed"
		}
	case "unknown":
		stepStatus, runStatus = "failed", "reconciling"
	}
	if _, err = tx.Exec(ctx, `UPDATE workflow_step_runs SET status=$3,output=CASE WHEN $4::jsonb='null'::jsonb THEN output ELSE $4 END,error=CASE WHEN $5::jsonb='null'::jsonb THEN error ELSE $5 END,last_error=CASE WHEN $5::jsonb='null'::jsonb THEN last_error ELSE $5 END,next_retry_at=$6,locked_by=CASE WHEN $3='leased' THEN locked_by ELSE NULL END,locked_until=CASE WHEN $3='leased' THEN locked_until ELSE NULL END,completed_at=CASE WHEN $3 IN ('succeeded','failed') THEN now() ELSE completed_at END WHERE tenant_id=$1 AND id=$2`, scope.TenantID, step.ID, stepStatus, outputJSON, errorJSON, retryAt); err != nil {
		return operations.AdapterReceipt{}, err
	}
	if input.Status == "succeeded" {
		var remaining int
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM workflow_step_runs WHERE tenant_id=$1 AND run_id=$2 AND id<>$3 AND status<>'succeeded'`, scope.TenantID, step.RunID, step.ID).Scan(&remaining); err != nil {
			return operations.AdapterReceipt{}, err
		}
		if remaining > 0 {
			runStatus = "running"
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE workflow_runs SET status=$3,last_error=CASE WHEN $4::jsonb='null'::jsonb THEN last_error ELSE $4 END,completed_at=CASE WHEN $3 IN ('succeeded','failed') THEN now() ELSE NULL END,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, step.RunID, runStatus, errorJSON); err != nil {
		return operations.AdapterReceipt{}, err
	}
	if (input.Status == "failed" && step.Attempt >= step.MaxAttempts) || input.Status == "unknown" {
		severity := "critical"
		alertType := "workflow_retry_exhausted"
		if input.Status == "unknown" {
			alertType = "adapter_result_unknown"
		}
		details, _ := json.Marshal(map[string]any{"receipt_id": item.ID, "attempt": step.Attempt, "max_attempts": step.MaxAttempts})
		_, err = tx.Exec(ctx, `INSERT INTO operational_alerts (tenant_id,alert_type,severity,object_type,object_id,message,details) VALUES($1,$2,$3,'execution_order',$4,$5,$6) ON CONFLICT (tenant_id,alert_type,object_type,object_id) WHERE status IN ('open','acknowledged') DO UPDATE SET severity=EXCLUDED.severity,message=EXCLUDED.message,details=EXCLUDED.details`, scope.TenantID, alertType, severity, input.ExecutionID, "trusted Adapter result requires operator attention", details)
		if err != nil {
			return operations.AdapterReceipt{}, err
		}
	}
	metadata := map[string]any{"adapter_identity_id": identity.ID, "execution_order_id": input.ExecutionID, "workflow_step_id": step.ID, "result_status": input.Status, "execution_status": newStatus}
	if err = appendAudit(ctx, tx, scope, "adapter_result.ingest", "adapter_result_receipt", item.ID, input.ExternalEventID, metadata); err != nil {
		return operations.AdapterReceipt{}, err
	}
	if err = appendEvent(ctx, tx, scope, "execution_order", input.ExecutionID, "adapter_result.received", step.Attempt, metadata); err != nil {
		return operations.AdapterReceipt{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return operations.AdapterReceipt{}, mapError(err)
	}
	return item, nil
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func scanAlert(row rowScanner) (operations.Alert, error) {
	var item operations.Alert
	var details []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.AlertType, &item.Severity, &item.Status, &item.ObjectType, &item.ObjectID, &item.Message, &details, &item.CreatedAt, &item.AcknowledgedAt, &item.ResolvedAt)
	item.Details = decodeObject(details)
	return item, err
}

func (s *Store) ListOperations(ctx context.Context, scope tenancy.Scope) (operations.Overview, error) {
	result, err := runTenantRead(ctx, s, scope.TenantID, func(tx pgx.Tx) (operations.Overview, error) {
		result := operations.Overview{WorkflowRuns: []operations.WorkflowRun{}, AdapterIdentities: []operations.AdapterIdentity{}, AdapterReceipts: []operations.AdapterReceipt{}, Alerts: []operations.Alert{}, DeploymentChecks: []operations.DeploymentCheck{}}
		if err := tx.QueryRow(ctx, `
			SELECT count(*) FILTER (WHERE processed_at IS NULL AND dead_letter_reason IS NULL),
			       count(*) FILTER (WHERE processed_at IS NULL AND dead_letter_reason IS NULL AND next_retry_at IS NOT NULL),
			       count(*) FILTER (WHERE dead_letter_reason IS NOT NULL),
			       min(occurred_at) FILTER (WHERE processed_at IS NULL AND dead_letter_reason IS NULL),max(published_at)
			FROM outbox_events WHERE tenant_id=$1`, scope.TenantID).Scan(&result.Outbox.Pending, &result.Outbox.RetryScheduled, &result.Outbox.DeadLetter, &result.Outbox.OldestPendingAt, &result.Outbox.LastPublishedAt); err != nil {
			return result, err
		}
		rows, err := tx.Query(ctx, `SELECT id FROM workflow_runs WHERE tenant_id=$1 ORDER BY updated_at DESC LIMIT 100`, scope.TenantID)
		if err != nil {
			return result, err
		}
		runIDs := []string{}
		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				rows.Close()
				return result, err
			}
			runIDs = append(runIDs, id)
		}
		rows.Close()
		for _, id := range runIDs {
			run, runErr := getWorkflowRun(ctx, tx, scope.TenantID, id)
			if runErr != nil {
				return result, runErr
			}
			result.WorkflowRuns = append(result.WorkflowRuns, run)
		}
		rows, err = tx.Query(ctx, `SELECT id,tenant_id,name,key_id,provider_endpoint_id,secret_ref,status,created_by,created_at,updated_at FROM adapter_identities WHERE tenant_id=$1 ORDER BY created_at DESC`, scope.TenantID)
		if err != nil {
			return result, err
		}
		for rows.Next() {
			item, scanErr := scanAdapterIdentity(rows)
			if scanErr != nil {
				rows.Close()
				return result, scanErr
			}
			result.AdapterIdentities = append(result.AdapterIdentities, item)
		}
		rows.Close()
		rows, err = tx.Query(ctx, `SELECT id,tenant_id,adapter_identity_id,execution_order_id,workflow_step_id,external_event_id,nonce,result_status,payload,received_at,processed_at FROM adapter_result_receipts WHERE tenant_id=$1 ORDER BY received_at DESC LIMIT 200`, scope.TenantID)
		if err != nil {
			return result, err
		}
		for rows.Next() {
			item, scanErr := scanAdapterReceipt(rows)
			if scanErr != nil {
				rows.Close()
				return result, scanErr
			}
			result.AdapterReceipts = append(result.AdapterReceipts, item)
		}
		rows.Close()
		rows, err = tx.Query(ctx, `SELECT id,tenant_id,alert_type,severity,status,object_type,object_id,message,details,created_at,acknowledged_at,resolved_at FROM operational_alerts WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT 200`, scope.TenantID)
		if err != nil {
			return result, err
		}
		for rows.Next() {
			item, scanErr := scanAlert(rows)
			if scanErr != nil {
				rows.Close()
				return result, scanErr
			}
			result.Alerts = append(result.Alerts, item)
		}
		rows.Close()
		checks := []struct {
			name, query, message string
		}{
			{"phase_h_migration", `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version='000011_phase_h_integration')`, "phase H schema migration is applied"},
			{"outbound_delivery_disabled", `SELECT NOT COALESCE((SELECT COALESCE(tenant.enabled,definition.default_enabled) FROM feature_definitions definition LEFT JOIN tenant_features tenant ON tenant.feature_key=definition.key AND tenant.tenant_id=$1 WHERE definition.key='growth.outbound_delivery'),false)`, "external outreach remains disabled"},
			{"marketplace_payout_disabled", `SELECT NOT COALESCE((SELECT COALESCE(tenant.enabled,definition.default_enabled) FROM feature_definitions definition LEFT JOIN tenant_features tenant ON tenant.feature_key=definition.key AND tenant.tenant_id=$1 WHERE definition.key='marketplace.payout'),false)`, "marketplace payout remains disabled"},
			{"phase_h_rls", `SELECT count(*)=5 FROM pg_class WHERE relname=ANY(ARRAY['opportunity_signals','adapter_identities','adapter_result_receipts','outbox_replays','operational_alerts']) AND relrowsecurity`, "phase H tenant tables have RLS enabled"},
		}
		for _, check := range checks {
			var passed bool
			var row pgx.Row
			if strings.Contains(check.query, "$1") {
				row = tx.QueryRow(ctx, check.query, scope.TenantID)
			} else {
				row = tx.QueryRow(ctx, check.query)
			}
			if err = row.Scan(&passed); err != nil {
				return result, err
			}
			status := "failed"
			if passed {
				status = "passed"
			}
			result.DeploymentChecks = append(result.DeploymentChecks, operations.DeploymentCheck{Name: check.name, Status: status, Message: check.message})
		}
		return result, nil
	})
	if err != nil {
		return result, err
	}
	if result.Outbox.OldestPendingAt != nil {
		result.Outbox.OldestPendingAge = int64(time.Since(*result.Outbox.OldestPendingAt).Seconds())
	}
	maxLagSeconds := int64(300)
	if configured, parseErr := strconv.ParseInt(os.Getenv("OUTBOX_MAX_LAG_SECONDS"), 10, 64); parseErr == nil && configured > 0 {
		maxLagSeconds = configured
	}
	lagStatus := "passed"
	lagMessage := fmt.Sprintf("oldest pending Outbox event is within %d seconds", maxLagSeconds)
	if result.Outbox.Pending > 0 && result.Outbox.OldestPendingAge > maxLagSeconds {
		lagStatus = "failed"
		lagMessage = fmt.Sprintf("oldest pending Outbox event is %d seconds old", result.Outbox.OldestPendingAge)
	}
	result.DeploymentChecks = append(result.DeploymentChecks, operations.DeploymentCheck{Name: "outbox_delivery_lag", Status: lagStatus, Message: lagMessage})
	for _, identity := range result.AdapterIdentities {
		status := "passed"
		message := "adapter secret reference is available"
		if os.Getenv(identity.SecretRef) == "" {
			status, message = "failed", "adapter secret reference "+identity.SecretRef+" is unavailable"
		}
		result.DeploymentChecks = append(result.DeploymentChecks, operations.DeploymentCheck{Name: "adapter_secret:" + identity.KeyID, Status: status, Message: message})
	}
	return result, nil
}

func (s *Store) ReplayOutbox(ctx context.Context, scope tenancy.Scope, eventID, reason, key string) (operations.OutboxReplay, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return operations.OutboxReplay{}, platform.Invalid("replay_reason_required", "an audited replay reason is required")
	}
	return runCommand(ctx, s, scope, key, "outbox.replay", func(tx pgx.Tx) (operations.OutboxReplay, string, error) {
		var deadLetterReason string
		var retryCount int
		if err := tx.QueryRow(ctx, `SELECT COALESCE(dead_letter_reason,''),retry_count FROM outbox_events WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, eventID).Scan(&deadLetterReason, &retryCount); err != nil {
			return operations.OutboxReplay{}, "", err
		}
		if deadLetterReason == "" {
			return operations.OutboxReplay{}, "", platform.Invalid("outbox_not_dead_letter", "only dead-letter events can be replayed")
		}
		var item operations.OutboxReplay
		if err := tx.QueryRow(ctx, `INSERT INTO outbox_replays (tenant_id,outbox_event_id,reason,previous_retry_count,requested_by) VALUES($1,$2,$3,$4,$5) RETURNING id,tenant_id,outbox_event_id,reason,previous_retry_count,requested_by,requested_at`, scope.TenantID, eventID, reason, retryCount, scope.ActorID).Scan(&item.ID, &item.TenantID, &item.OutboxEventID, &item.Reason, &item.PreviousRetryCount, &item.RequestedBy, &item.RequestedAt); err != nil {
			return item, "", err
		}
		if _, err := tx.Exec(ctx, `UPDATE outbox_events SET retry_count=0,next_retry_at=now(),dead_letter_reason=NULL,last_error=NULL,locked_by=NULL,locked_until=NULL WHERE tenant_id=$1 AND id=$2`, scope.TenantID, eventID); err != nil {
			return item, "", err
		}
		if _, err := tx.Exec(ctx, `UPDATE operational_alerts SET status='resolved',resolved_at=now() WHERE tenant_id=$1 AND alert_type='outbox_dead_letter' AND object_type='outbox_event' AND object_id=$2 AND status IN ('open','acknowledged')`, scope.TenantID, eventID); err != nil {
			return item, "", err
		}
		metadata := map[string]any{"outbox_event_id": eventID, "reason": reason, "previous_retry_count": retryCount}
		if err := appendAudit(ctx, tx, scope, "outbox.replay", "outbox_event", eventID, key, metadata); err != nil {
			return item, "", err
		}
		if err := appendEvent(ctx, tx, scope, "outbox_replay", item.ID, "outbox.replay_requested", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) AcknowledgeOperationalAlert(ctx context.Context, scope tenancy.Scope, alertID, key string) (operations.Alert, error) {
	return runCommand(ctx, s, scope, key, "operational_alert.acknowledge", func(tx pgx.Tx) (operations.Alert, string, error) {
		item, err := scanAlert(tx.QueryRow(ctx, `UPDATE operational_alerts SET status='acknowledged',acknowledged_by=$3,acknowledged_at=now() WHERE tenant_id=$1 AND id=$2 AND status='open' RETURNING id,tenant_id,alert_type,severity,status,object_type,object_id,message,details,created_at,acknowledged_at,resolved_at`, scope.TenantID, alertID, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		if err = appendAudit(ctx, tx, scope, "operational_alert.acknowledge", "operational_alert", item.ID, key, map[string]any{"alert_type": item.AlertType}); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

var _ application.IntegrationStore = (*Store)(nil)

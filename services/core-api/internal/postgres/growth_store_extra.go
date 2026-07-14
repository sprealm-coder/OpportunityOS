package postgres

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/application"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/growth"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

func scanConversation(row rowScanner) (growth.Conversation, error) {
	var item growth.Conversation
	err := row.Scan(&item.ID, &item.TenantID, &item.LeadID, &item.DealID, &item.Channel, &item.Status, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt, &item.LastMessageAt)
	item.Messages = []growth.ConversationMessage{}
	return item, err
}

func scanConversationMessage(row rowScanner) (growth.ConversationMessage, error) {
	var item growth.ConversationMessage
	var content []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.ConversationID, &item.Direction, &item.Status, &content, &item.IdempotencyKey, &item.CreatedBy, &item.CreatedAt)
	if err == nil {
		item.Content, err = unmarshalGrowthJSON(content)
	}
	return item, err
}

func (s *Store) CreateConversation(ctx context.Context, scope tenancy.Scope, input application.ConversationInput, key string) (growth.Conversation, error) {
	input.Channel = strings.ToLower(strings.TrimSpace(input.Channel))
	if input.LeadID == "" || !validGrowthChannel(input.Channel) {
		return growth.Conversation{}, platform.Invalid("invalid_conversation", "lead and a valid channel are required")
	}
	return runCommand(ctx, s, scope, key, "conversation.create", func(tx pgx.Tx) (growth.Conversation, string, error) {
		if _, err := getGrowthLead(ctx, tx, scope.TenantID, input.LeadID, false); err != nil {
			return growth.Conversation{}, "", err
		}
		if input.DealID != "" {
			var matches bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM deals WHERE tenant_id=$1 AND id=$2 AND lead_id=$3)`, scope.TenantID, input.DealID, input.LeadID).Scan(&matches); err != nil {
				return growth.Conversation{}, "", err
			}
			if !matches {
				return growth.Conversation{}, "", platform.Invalid("deal_lead_mismatch", "conversation deal must belong to the lead")
			}
		}
		item, err := scanConversation(tx.QueryRow(ctx, `INSERT INTO conversations (tenant_id,lead_id,deal_id,channel,created_by) VALUES($1,$2,$3,$4,$5) RETURNING id,tenant_id,lead_id,COALESCE(deal_id::text,''),channel,status,created_by,created_at,updated_at,last_message_at`, scope.TenantID, input.LeadID, nullableUUID(input.DealID), input.Channel, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"lead_id": input.LeadID, "deal_id": input.DealID, "channel": input.Channel}
		if err = appendAudit(ctx, tx, scope, "conversation.create", "conversation", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "conversation", item.ID, "conversation.created", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) AddConversationMessage(ctx context.Context, scope tenancy.Scope, conversationID string, input application.ConversationMessageInput, key string) (growth.ConversationMessage, error) {
	input.Direction = strings.ToLower(strings.TrimSpace(input.Direction))
	input.Status = strings.ToLower(strings.TrimSpace(input.Status))
	valid := (input.Direction == "inbound" && input.Status == "received") ||
		(input.Direction == "outbound" && (input.Status == "draft" || input.Status == "planned")) ||
		(input.Direction == "system" && (input.Status == "received" || input.Status == "draft"))
	if !valid || len(input.Content) == 0 {
		return growth.ConversationMessage{}, platform.Invalid("invalid_conversation_message", "direction, non-delivery status, and content are required")
	}
	content, err := marshalGrowthJSON(input.Content)
	if err != nil {
		return growth.ConversationMessage{}, err
	}
	return runCommand(ctx, s, scope, key, "conversation.message", func(tx pgx.Tx) (growth.ConversationMessage, string, error) {
		conversation, err := scanConversation(tx.QueryRow(ctx, `SELECT id,tenant_id,lead_id,COALESCE(deal_id::text,''),channel,status,created_by,created_at,updated_at,last_message_at FROM conversations WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, conversationID))
		if err != nil {
			return growth.ConversationMessage{}, "", err
		}
		if conversation.Status != "open" {
			return growth.ConversationMessage{}, "", platform.Invalid("conversation_closed", "closed conversation cannot accept messages")
		}
		item, err := scanConversationMessage(tx.QueryRow(ctx, `INSERT INTO conversation_messages (tenant_id,conversation_id,direction,status,content,idempotency_key,created_by) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id,tenant_id,conversation_id,direction,status,content,idempotency_key,created_by,created_at`, scope.TenantID, conversationID, input.Direction, input.Status, content, key, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		if _, err = tx.Exec(ctx, `UPDATE conversations SET last_message_at=$3,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, conversationID, item.CreatedAt); err != nil {
			return item, "", err
		}
		if input.Direction == "inbound" {
			lead, leadErr := getGrowthLead(ctx, tx, scope.TenantID, conversation.LeadID, true)
			if leadErr != nil {
				return item, "", leadErr
			}
			if lead.Status == "contacted" {
				if leadErr = lead.Transition("replied"); leadErr != nil {
					return item, "", leadErr
				}
				if _, leadErr = tx.Exec(ctx, `UPDATE leads SET status=$3,version=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, lead.ID, lead.Status, lead.Version); leadErr != nil {
					return item, "", leadErr
				}
			}
		}
		metadata := map[string]any{"conversation_id": conversationID, "direction": input.Direction, "status": input.Status}
		if err = appendAudit(ctx, tx, scope, "conversation.message", "conversation_message", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "conversation", conversationID, "conversation.message_added", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func scanDeal(row rowScanner) (growth.Deal, error) {
	var item growth.Deal
	err := row.Scan(&item.ID, &item.TenantID, &item.LeadID, &item.Name, &item.CustomerID, &item.Status, &item.ValueMinor, &item.Currency, &item.Version, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt, &item.ClosedAt)
	item.Currency = strings.TrimSpace(item.Currency)
	return item, err
}

func getDeal(ctx context.Context, tx pgx.Tx, tenantID, id string, lock bool) (growth.Deal, error) {
	query := `SELECT id,tenant_id,lead_id,name,customer_id,status,value_minor,currency,version,created_by,created_at,updated_at,closed_at FROM deals WHERE tenant_id=$1 AND id=$2`
	if lock {
		query += ` FOR UPDATE`
	}
	return scanDeal(tx.QueryRow(ctx, query, tenantID, id))
}

func (s *Store) CreateDeal(ctx context.Context, scope tenancy.Scope, input application.DealInput, key string) (growth.Deal, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.CustomerID = strings.TrimSpace(input.CustomerID)
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	if input.LeadID == "" || input.Name == "" || input.CustomerID == "" || len(input.Currency) != 3 || input.ValueMinor < 0 {
		return growth.Deal{}, platform.Invalid("invalid_deal", "lead, name, customer, non-negative value, and three-letter currency are required")
	}
	return runCommand(ctx, s, scope, key, "deal.create", func(tx pgx.Tx) (growth.Deal, string, error) {
		lead, err := getGrowthLead(ctx, tx, scope.TenantID, input.LeadID, false)
		if err != nil {
			return growth.Deal{}, "", err
		}
		if lead.Status == "discovered" || lead.Status == "enriched" || lead.Status == "won" || lead.Status == "lost" || lead.Status == "suppressed" {
			return growth.Deal{}, "", platform.Invalid("lead_not_deal_ready", "lead must be qualified and open before creating a deal")
		}
		item, err := scanDeal(tx.QueryRow(ctx, `INSERT INTO deals (tenant_id,lead_id,name,customer_id,currency,value_minor,created_by) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id,tenant_id,lead_id,name,customer_id,status,value_minor,currency,version,created_by,created_at,updated_at,closed_at`, scope.TenantID, input.LeadID, input.Name, input.CustomerID, input.Currency, input.ValueMinor, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"lead_id": input.LeadID, "customer_id": input.CustomerID, "value_minor": input.ValueMinor, "currency": input.Currency}
		if err = appendAudit(ctx, tx, scope, "deal.create", "deal", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "deal", item.ID, "deal.created", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) GetDeal(ctx context.Context, scope tenancy.Scope, id string) (growth.Deal, error) {
	return runTenantRead(ctx, s, scope.TenantID, func(tx pgx.Tx) (growth.Deal, error) {
		return getDeal(ctx, tx, scope.TenantID, id, false)
	})
}

func (s *Store) TransitionDeal(ctx context.Context, scope tenancy.Scope, id, to, key string) (growth.Deal, error) {
	to = strings.ToLower(strings.TrimSpace(to))
	return runCommand(ctx, s, scope, key, "deal.transition", func(tx pgx.Tx) (growth.Deal, string, error) {
		item, err := getDeal(ctx, tx, scope.TenantID, id, true)
		if err != nil {
			return item, "", err
		}
		if to == "proposal" || to == "won" {
			var quoteReady bool
			status := "draft"
			if to == "won" {
				status = "accepted"
			}
			if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM quotes WHERE tenant_id=$1 AND growth_deal_id=$2 AND status=$3)`, scope.TenantID, id, status).Scan(&quoteReady); err != nil {
				return item, "", err
			}
			if !quoteReady {
				return item, "", platform.Invalid("deal_quote_gate", "deal transition requires the corresponding canonical quote state")
			}
		}
		from := item.Status
		if err = item.Transition(to); err != nil {
			return item, "", err
		}
		item, err = scanDeal(tx.QueryRow(ctx, `UPDATE deals SET status=$3,version=$4,closed_at=$5,updated_at=now() WHERE tenant_id=$1 AND id=$2 RETURNING id,tenant_id,lead_id,name,customer_id,status,value_minor,currency,version,created_by,created_at,updated_at,closed_at`, scope.TenantID, id, item.Status, item.Version, item.ClosedAt))
		if err != nil {
			return item, "", err
		}
		lead, err := getGrowthLead(ctx, tx, scope.TenantID, item.LeadID, true)
		if err != nil {
			return item, "", err
		}
		leadTarget := ""
		if to == "proposal" && lead.Status == "meeting" {
			leadTarget = "proposal"
		} else if to == "won" && lead.Status == "proposal" {
			leadTarget = "won"
		} else if to == "lost" && lead.Status != "won" && lead.Status != "lost" && lead.Status != "suppressed" {
			leadTarget = "lost"
		}
		if leadTarget != "" {
			if err = lead.Transition(leadTarget); err != nil {
				return item, "", err
			}
			if _, err = tx.Exec(ctx, `UPDATE leads SET status=$3,version=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, lead.ID, lead.Status, lead.Version); err != nil {
				return item, "", err
			}
		}
		metadata := map[string]any{"from": from, "to": to, "lead_id": item.LeadID}
		if err = appendAudit(ctx, tx, scope, "deal.transition", "deal", id, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "deal", id, "deal.transitioned", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func scanExperiment(row rowScanner) (growth.Experiment, error) {
	var item growth.Experiment
	var metrics, result []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.Name, &item.EntityType, &item.EntityID, &item.Hypothesis, &item.Status, &item.AllocationBasisPoints, &metrics, &result, &item.Version, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	if item.MetricsDefinition, err = unmarshalGrowthJSON(metrics); err != nil {
		return item, err
	}
	item.Result, err = unmarshalGrowthJSON(result)
	return item, err
}

func growthEntityExists(ctx context.Context, tx pgx.Tx, tenantID, entityType, entityID string) (bool, error) {
	tables := map[string]string{"market_segment": "market_segments", "lead": "leads", "proof": "proof_instances", "campaign": "campaigns", "deal": "deals"}
	table, ok := tables[entityType]
	if !ok {
		return false, platform.Invalid("invalid_experiment_entity", "unsupported experiment entity type")
	}
	var exists bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM `+table+` WHERE tenant_id=$1 AND id=$2)`, tenantID, entityID).Scan(&exists)
	return exists, err
}

func (s *Store) CreateExperiment(ctx context.Context, scope tenancy.Scope, input application.ExperimentInput, key string) (growth.Experiment, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.EntityType = strings.ToLower(strings.TrimSpace(input.EntityType))
	input.Hypothesis = strings.TrimSpace(input.Hypothesis)
	if input.Name == "" || input.EntityID == "" || input.Hypothesis == "" || input.AllocationBasisPoints < 0 || input.AllocationBasisPoints > 10000 || len(input.MetricsDefinition) == 0 {
		return growth.Experiment{}, platform.Invalid("invalid_experiment", "name, entity, hypothesis, allocation, and metrics are required")
	}
	metrics, err := marshalGrowthJSON(input.MetricsDefinition)
	if err != nil {
		return growth.Experiment{}, err
	}
	return runCommand(ctx, s, scope, key, "experiment.create", func(tx pgx.Tx) (growth.Experiment, string, error) {
		exists, err := growthEntityExists(ctx, tx, scope.TenantID, input.EntityType, input.EntityID)
		if err != nil {
			return growth.Experiment{}, "", err
		}
		if !exists {
			return growth.Experiment{}, "", platform.Invalid("experiment_entity_not_found", "experiment entity must belong to the tenant")
		}
		item, err := scanExperiment(tx.QueryRow(ctx, `INSERT INTO experiments (tenant_id,name,entity_type,entity_id,hypothesis,allocation_basis_points,metrics_definition,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id,tenant_id,name,entity_type,entity_id,hypothesis,status,allocation_basis_points,metrics_definition,result,version,created_by,created_at,updated_at`, scope.TenantID, input.Name, input.EntityType, input.EntityID, input.Hypothesis, input.AllocationBasisPoints, metrics, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"entity_type": input.EntityType, "entity_id": input.EntityID, "allocation_basis_points": input.AllocationBasisPoints}
		if err = appendAudit(ctx, tx, scope, "experiment.create", "experiment", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "experiment", item.ID, "experiment.created", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) TransitionExperiment(ctx context.Context, scope tenancy.Scope, id string, input application.ExperimentTransitionInput, key string) (growth.Experiment, error) {
	input.To = strings.ToLower(strings.TrimSpace(input.To))
	if input.To == "completed" && len(input.Result) == 0 {
		return growth.Experiment{}, platform.Invalid("experiment_result_required", "completed experiment requires a result")
	}
	result, err := marshalGrowthJSON(input.Result)
	if err != nil {
		return growth.Experiment{}, err
	}
	return runCommand(ctx, s, scope, key, "experiment.transition", func(tx pgx.Tx) (growth.Experiment, string, error) {
		item, err := scanExperiment(tx.QueryRow(ctx, `SELECT id,tenant_id,name,entity_type,entity_id,hypothesis,status,allocation_basis_points,metrics_definition,result,version,created_by,created_at,updated_at FROM experiments WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, id))
		if err != nil {
			return item, "", err
		}
		from := item.Status
		if err = item.Transition(input.To); err != nil {
			return item, "", err
		}
		if input.To == "completed" {
			item.Result = input.Result
		}
		item, err = scanExperiment(tx.QueryRow(ctx, `UPDATE experiments SET status=$3,result=$4,version=$5,updated_at=now() WHERE tenant_id=$1 AND id=$2 RETURNING id,tenant_id,name,entity_type,entity_id,hypothesis,status,allocation_basis_points,metrics_definition,result,version,created_by,created_at,updated_at`, scope.TenantID, id, item.Status, result, item.Version))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"from": from, "to": input.To}
		if err = appendAudit(ctx, tx, scope, "experiment.transition", "experiment", id, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "experiment", id, "experiment.transitioned", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func collectGrowthRows[T any](ctx context.Context, tx pgx.Tx, query string, scan func(rowScanner) (T, error), args ...any) ([]T, error) {
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []T{}
	for rows.Next() {
		item, scanErr := scan(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ListGrowth(ctx context.Context, scope tenancy.Scope) (growth.Overview, error) {
	return runTenantRead(ctx, s, scope.TenantID, func(tx pgx.Tx) (growth.Overview, error) {
		var result growth.Overview
		var err error
		result.Segments, err = collectGrowthRows(ctx, tx, `SELECT id,tenant_id,name,status,definition,version,created_by,created_at,updated_at FROM market_segments WHERE tenant_id=$1 ORDER BY updated_at DESC,id LIMIT 200`, scanMarketSegment, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.ICPs, err = collectGrowthRows(ctx, tx, `SELECT id,tenant_id,market_segment_id,name,status,definition,version,created_by,created_at,updated_at FROM icp_definitions WHERE tenant_id=$1 ORDER BY updated_at DESC,id LIMIT 200`, scanICPDefinition, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.Leads, err = collectGrowthRows(ctx, tx, `SELECT id,tenant_id,market_segment_id,COALESCE(icp_definition_id::text,''),name,status,score,version,created_by,created_at,updated_at FROM leads WHERE tenant_id=$1 ORDER BY updated_at DESC,id LIMIT 500`, scanLead, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.Evidence, err = collectGrowthRows(ctx, tx, `SELECT id,tenant_id,lead_id,kind,summary,confidence,source_ref,created_by,created_at FROM lead_evidence WHERE tenant_id=$1 ORDER BY created_at DESC,id LIMIT 1000`, scanLeadEvidence, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.Contacts, err = collectGrowthRows(ctx, tx, `SELECT id,tenant_id,lead_id,channel,value,normalized_value,status,consent_status,created_by,created_at,updated_at FROM contacts WHERE tenant_id=$1 ORDER BY updated_at DESC,id LIMIT 500`, scanContact, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.ProofTemplates, err = collectGrowthRows(ctx, tx, `SELECT id,tenant_id,name,proof_type,workflow_version_id,input_schema,output_schema,access_policy,retention_days,status,version,created_by,created_at,updated_at FROM proof_templates WHERE tenant_id=$1 ORDER BY updated_at DESC,id LIMIT 200`, scanProofTemplate, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.ProofRequests, err = collectGrowthRows(ctx, tx, `SELECT id,tenant_id,lead_id,COALESCE(deal_id::text,''),template_id,status,input,requested_by,expires_at,created_at,updated_at FROM proof_requests WHERE tenant_id=$1 ORDER BY updated_at DESC,id LIMIT 500`, scanProofRequest, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.ProofInstances, err = collectGrowthRows(ctx, tx, `SELECT id,tenant_id,proof_request_id,status,result,artifact_ref,review_rationale,generated_by,COALESCE(reviewed_by,''),created_at,reviewed_at,expires_at FROM proof_instances WHERE tenant_id=$1 ORDER BY created_at DESC,id LIMIT 500`, scanProofInstance, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.Campaigns, err = collectGrowthRows(ctx, tx, `SELECT id,tenant_id,COALESCE(market_segment_id::text,''),name,channel,purpose,status,version,created_by,created_at,updated_at FROM campaigns WHERE tenant_id=$1 ORDER BY updated_at DESC,id LIMIT 200`, scanCampaign, scope.TenantID)
		if err != nil {
			return result, err
		}
		for index := range result.Campaigns {
			campaign := &result.Campaigns[index]
			campaign.Steps, err = collectGrowthRows(ctx, tx, `SELECT id,tenant_id,campaign_id,position,kind,definition,created_at FROM campaign_steps WHERE tenant_id=$1 AND campaign_id=$2 ORDER BY position,id`, scanCampaignStep, scope.TenantID, campaign.ID)
			if err != nil {
				return result, err
			}
			approval, approvalErr := scanCampaignApproval(tx.QueryRow(ctx, `SELECT id,tenant_id,campaign_id,campaign_version,decision,rationale,reviewed_by,created_at FROM campaign_approvals WHERE tenant_id=$1 AND campaign_id=$2 ORDER BY campaign_version DESC LIMIT 1`, scope.TenantID, campaign.ID))
			if approvalErr == nil {
				campaign.Approval = &approval
			} else if approvalErr != pgx.ErrNoRows {
				return result, approvalErr
			}
		}
		result.Suppressions, err = collectGrowthRows(ctx, tx, `SELECT id,tenant_id,subject_type,subject_id,channel,reason,status,source_ref,created_by,created_at,released_at FROM suppression_entries WHERE tenant_id=$1 ORDER BY created_at DESC,id LIMIT 500`, scanSuppression, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.Outreach, err = collectGrowthRows(ctx, tx, `SELECT id,tenant_id,campaign_id,campaign_step_id,lead_id,COALESCE(contact_id::text,''),status,content,block_reason,COALESCE(external_message_id,''),idempotency_key,created_by,created_at,updated_at FROM outreach_messages WHERE tenant_id=$1 ORDER BY created_at DESC,id LIMIT 1000`, scanOutreach, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.Conversations, err = collectGrowthRows(ctx, tx, `SELECT id,tenant_id,lead_id,COALESCE(deal_id::text,''),channel,status,created_by,created_at,updated_at,last_message_at FROM conversations WHERE tenant_id=$1 ORDER BY updated_at DESC,id LIMIT 500`, scanConversation, scope.TenantID)
		if err != nil {
			return result, err
		}
		for index := range result.Conversations {
			conversation := &result.Conversations[index]
			conversation.Messages, err = collectGrowthRows(ctx, tx, `SELECT id,tenant_id,conversation_id,direction,status,content,idempotency_key,created_by,created_at FROM conversation_messages WHERE tenant_id=$1 AND conversation_id=$2 ORDER BY created_at,id`, scanConversationMessage, scope.TenantID, conversation.ID)
			if err != nil {
				return result, err
			}
		}
		result.Deals, err = collectGrowthRows(ctx, tx, `SELECT id,tenant_id,lead_id,name,customer_id,status,value_minor,currency,version,created_by,created_at,updated_at,closed_at FROM deals WHERE tenant_id=$1 ORDER BY updated_at DESC,id LIMIT 500`, scanDeal, scope.TenantID)
		if err != nil {
			return result, err
		}
		result.Experiments, err = collectGrowthRows(ctx, tx, `SELECT id,tenant_id,name,entity_type,entity_id,hypothesis,status,allocation_basis_points,metrics_definition,result,version,created_by,created_at,updated_at FROM experiments WHERE tenant_id=$1 ORDER BY updated_at DESC,id LIMIT 500`, scanExperiment, scope.TenantID)
		return result, err
	})
}

var _ application.GrowthStore = (*Store)(nil)

package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/application"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/growth"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/schema"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

func marshalGrowthJSON(value map[string]any) ([]byte, error) {
	if value == nil {
		value = map[string]any{}
	}
	return json.Marshal(value)
}

func unmarshalGrowthJSON(raw []byte) (map[string]any, error) {
	value := map[string]any{}
	if len(raw) == 0 {
		return value, nil
	}
	return value, json.Unmarshal(raw, &value)
}

func suppressionKey(tenantID, subjectType, subjectID string) string {
	sum := sha256.Sum256([]byte(tenantID + "|" + subjectType + "|" + strings.ToLower(strings.TrimSpace(subjectID))))
	return hex.EncodeToString(sum[:])
}

func validGrowthChannel(channel string) bool {
	return channel == "email" || channel == "phone" || channel == "web" || channel == "custom"
}

func scanMarketSegment(row rowScanner) (growth.MarketSegment, error) {
	var item growth.MarketSegment
	var definition []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.Name, &item.Status, &definition, &item.Version, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	if err == nil {
		item.Definition, err = unmarshalGrowthJSON(definition)
	}
	return item, err
}

func scanICPDefinition(row rowScanner) (growth.ICPDefinition, error) {
	var item growth.ICPDefinition
	var definition []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.MarketSegmentID, &item.Name, &item.Status, &definition, &item.Version, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	if err == nil {
		item.Definition, err = unmarshalGrowthJSON(definition)
	}
	return item, err
}

func scanLead(row rowScanner) (growth.Lead, error) {
	var item growth.Lead
	err := row.Scan(&item.ID, &item.TenantID, &item.SegmentID, &item.ICPDefinitionID, &item.Name, &item.Status, &item.Score, &item.Version, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	item.Evidence = []string{}
	return item, err
}

func scanLeadEvidence(row rowScanner) (growth.LeadEvidence, error) {
	var item growth.LeadEvidence
	err := row.Scan(&item.ID, &item.TenantID, &item.LeadID, &item.Kind, &item.Summary, &item.Confidence, &item.SourceRef, &item.CreatedBy, &item.CreatedAt)
	return item, err
}

func scanContact(row rowScanner) (growth.Contact, error) {
	var item growth.Contact
	err := row.Scan(&item.ID, &item.TenantID, &item.LeadID, &item.Channel, &item.Value, &item.NormalizedValue, &item.Status, &item.ConsentStatus, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func getGrowthLead(ctx context.Context, tx pgx.Tx, tenantID, id string, lock bool) (growth.Lead, error) {
	query := `SELECT id,tenant_id,market_segment_id,COALESCE(icp_definition_id::text,''),name,status,score,version,created_by,created_at,updated_at FROM leads WHERE tenant_id=$1 AND id=$2`
	if lock {
		query += ` FOR UPDATE`
	}
	return scanLead(tx.QueryRow(ctx, query, tenantID, id))
}

func (s *Store) CreateMarketSegment(ctx context.Context, scope tenancy.Scope, input application.MarketSegmentInput, key string) (growth.MarketSegment, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return growth.MarketSegment{}, platform.Invalid("segment_name_required", "market segment name is required")
	}
	definition, err := marshalGrowthJSON(input.Definition)
	if err != nil {
		return growth.MarketSegment{}, err
	}
	return runCommand(ctx, s, scope, key, "market_segment.create", func(tx pgx.Tx) (growth.MarketSegment, string, error) {
		item, err := scanMarketSegment(tx.QueryRow(ctx, `INSERT INTO market_segments (tenant_id,name,definition,created_by) VALUES($1,$2,$3,$4) RETURNING id,tenant_id,name,status,definition,version,created_by,created_at,updated_at`, scope.TenantID, input.Name, definition, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"name": item.Name, "status": item.Status}
		if err = appendAudit(ctx, tx, scope, "market_segment.create", "market_segment", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "market_segment", item.ID, "market_segment.created", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) CreateICPDefinition(ctx context.Context, scope tenancy.Scope, segmentID string, input application.ICPDefinitionInput, key string) (growth.ICPDefinition, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return growth.ICPDefinition{}, platform.Invalid("icp_name_required", "ICP definition name is required")
	}
	definition, err := marshalGrowthJSON(input.Definition)
	if err != nil {
		return growth.ICPDefinition{}, err
	}
	return runCommand(ctx, s, scope, key, "icp.create", func(tx pgx.Tx) (growth.ICPDefinition, string, error) {
		var segmentStatus string
		if err := tx.QueryRow(ctx, `SELECT status FROM market_segments WHERE tenant_id=$1 AND id=$2`, scope.TenantID, segmentID).Scan(&segmentStatus); err != nil {
			return growth.ICPDefinition{}, "", err
		}
		if segmentStatus != "active" {
			return growth.ICPDefinition{}, "", platform.Invalid("segment_inactive", "ICP definition requires an active market segment")
		}
		item, err := scanICPDefinition(tx.QueryRow(ctx, `INSERT INTO icp_definitions (tenant_id,market_segment_id,name,definition,created_by) VALUES($1,$2,$3,$4,$5) RETURNING id,tenant_id,market_segment_id,name,status,definition,version,created_by,created_at,updated_at`, scope.TenantID, segmentID, input.Name, definition, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"market_segment_id": segmentID, "name": item.Name}
		if err = appendAudit(ctx, tx, scope, "icp.create", "icp_definition", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "icp_definition", item.ID, "icp.created", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) CreateLead(ctx context.Context, scope tenancy.Scope, input application.LeadInput, key string) (growth.Lead, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || input.MarketSegmentID == "" {
		return growth.Lead{}, platform.Invalid("lead_incomplete", "lead name and market segment are required")
	}
	return runCommand(ctx, s, scope, key, "lead.create", func(tx pgx.Tx) (growth.Lead, string, error) {
		var segmentActive, icpMatches bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM market_segments WHERE tenant_id=$1 AND id=$2 AND status='active')`, scope.TenantID, input.MarketSegmentID).Scan(&segmentActive); err != nil {
			return growth.Lead{}, "", err
		}
		if !segmentActive {
			return growth.Lead{}, "", platform.Invalid("invalid_segment", "lead requires an active tenant market segment")
		}
		if input.ICPDefinitionID != "" {
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM icp_definitions WHERE tenant_id=$1 AND id=$2 AND market_segment_id=$3 AND status='active')`, scope.TenantID, input.ICPDefinitionID, input.MarketSegmentID).Scan(&icpMatches); err != nil {
				return growth.Lead{}, "", err
			}
			if !icpMatches {
				return growth.Lead{}, "", platform.Invalid("invalid_icp", "ICP must be active and belong to the lead segment")
			}
		}
		item, err := scanLead(tx.QueryRow(ctx, `INSERT INTO leads (tenant_id,market_segment_id,icp_definition_id,name,created_by) VALUES($1,$2,$3,$4,$5) RETURNING id,tenant_id,market_segment_id,COALESCE(icp_definition_id::text,''),name,status,score,version,created_by,created_at,updated_at`, scope.TenantID, input.MarketSegmentID, nullableUUID(input.ICPDefinitionID), input.Name, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"market_segment_id": item.SegmentID, "icp_definition_id": item.ICPDefinitionID, "name": item.Name}
		if err = appendAudit(ctx, tx, scope, "lead.create", "lead", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "lead", item.ID, "lead.created", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) AddLeadEvidence(ctx context.Context, scope tenancy.Scope, leadID string, input application.LeadEvidenceInput, key string) (growth.LeadEvidence, error) {
	input.Kind, input.Summary, input.SourceRef = strings.TrimSpace(input.Kind), strings.TrimSpace(input.Summary), strings.TrimSpace(input.SourceRef)
	if input.Kind == "" || input.Summary == "" || input.Confidence < 0 || input.Confidence > 100 {
		return growth.LeadEvidence{}, platform.Invalid("invalid_lead_evidence", "kind, summary, and confidence from 0 to 100 are required")
	}
	return runCommand(ctx, s, scope, key, "lead.evidence", func(tx pgx.Tx) (growth.LeadEvidence, string, error) {
		lead, err := getGrowthLead(ctx, tx, scope.TenantID, leadID, true)
		if err != nil {
			return growth.LeadEvidence{}, "", err
		}
		if lead.Status == "suppressed" || lead.Status == "won" || lead.Status == "lost" {
			return growth.LeadEvidence{}, "", platform.Invalid("lead_closed", "closed or suppressed lead cannot accept evidence")
		}
		var item growth.LeadEvidence
		err = tx.QueryRow(ctx, `INSERT INTO lead_evidence (tenant_id,lead_id,kind,summary,confidence,source_ref,created_by) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id,tenant_id,lead_id,kind,summary,confidence,source_ref,created_by,created_at`, scope.TenantID, leadID, input.Kind, input.Summary, input.Confidence, input.SourceRef, scope.ActorID).Scan(&item.ID, &item.TenantID, &item.LeadID, &item.Kind, &item.Summary, &item.Confidence, &item.SourceRef, &item.CreatedBy, &item.CreatedAt)
		if err != nil {
			return item, "", err
		}
		if lead.Status == "discovered" {
			if err = lead.Transition("enriched"); err != nil {
				return item, "", err
			}
		}
		if input.Confidence > lead.Score {
			lead.Score = input.Confidence
		}
		if _, err = tx.Exec(ctx, `UPDATE leads SET status=$3,score=$4,version=$5,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, leadID, lead.Status, lead.Score, lead.Version); err != nil {
			return item, "", err
		}
		metadata := map[string]any{"lead_id": leadID, "kind": item.Kind, "confidence": item.Confidence}
		if err = appendAudit(ctx, tx, scope, "lead.evidence", "lead_evidence", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "lead", leadID, "lead.evidence_added", lead.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) TransitionLead(ctx context.Context, scope tenancy.Scope, leadID, to, key string) (growth.Lead, error) {
	to = strings.TrimSpace(to)
	return runCommand(ctx, s, scope, key, "lead.transition", func(tx pgx.Tx) (growth.Lead, string, error) {
		item, err := getGrowthLead(ctx, tx, scope.TenantID, leadID, true)
		if err != nil {
			return item, "", err
		}
		from := item.Status
		if to == "approved_for_outreach" {
			var suppressed bool
			keyHash := suppressionKey(scope.TenantID, "lead", leadID)
			if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM suppression_entries WHERE tenant_id=$1 AND subject_key=$2 AND status='active')`, scope.TenantID, keyHash).Scan(&suppressed); err != nil {
				return item, "", err
			}
			if suppressed {
				return item, "", platform.Invalid("lead_suppressed", "suppressed lead cannot be approved for outreach")
			}
		}
		if err = item.Transition(to); err != nil {
			return item, "", err
		}
		if _, err = tx.Exec(ctx, `UPDATE leads SET status=$3,version=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, leadID, item.Status, item.Version); err != nil {
			return item, "", err
		}
		metadata := map[string]any{"from": from, "to": to}
		if err = appendAudit(ctx, tx, scope, "lead.transition", "lead", leadID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "lead", leadID, "lead.transitioned", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) CreateContact(ctx context.Context, scope tenancy.Scope, leadID string, input application.ContactInput, key string) (growth.Contact, error) {
	input.Channel = strings.ToLower(strings.TrimSpace(input.Channel))
	input.Value = strings.TrimSpace(input.Value)
	input.ConsentStatus = strings.ToLower(strings.TrimSpace(input.ConsentStatus))
	if input.ConsentStatus == "" {
		input.ConsentStatus = "unknown"
	}
	if !validGrowthChannel(input.Channel) || input.Value == "" || (input.ConsentStatus != "unknown" && input.ConsentStatus != "opted_in" && input.ConsentStatus != "opted_out") {
		return growth.Contact{}, platform.Invalid("invalid_contact", "valid channel, value, and consent status are required")
	}
	normalized := strings.ToLower(input.Value)
	evidence, err := marshalGrowthJSON(input.Evidence)
	if err != nil {
		return growth.Contact{}, err
	}
	return runCommand(ctx, s, scope, key, "contact.create", func(tx pgx.Tx) (growth.Contact, string, error) {
		if _, err := getGrowthLead(ctx, tx, scope.TenantID, leadID, false); err != nil {
			return growth.Contact{}, "", err
		}
		item, err := scanContact(tx.QueryRow(ctx, `INSERT INTO contacts (tenant_id,lead_id,channel,value,normalized_value,consent_status,created_by) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id,tenant_id,lead_id,channel,value,normalized_value,status,consent_status,created_by,created_at,updated_at`, scope.TenantID, leadID, input.Channel, input.Value, normalized, input.ConsentStatus, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		if strings.TrimSpace(input.SourceType) != "" {
			if _, err = tx.Exec(ctx, `INSERT INTO contact_sources (tenant_id,contact_id,source_type,source_ref,evidence) VALUES($1,$2,$3,$4,$5)`, scope.TenantID, item.ID, strings.TrimSpace(input.SourceType), strings.TrimSpace(input.SourceRef), evidence); err != nil {
				return item, "", err
			}
		}
		metadata := map[string]any{"lead_id": leadID, "channel": item.Channel, "consent_status": item.ConsentStatus}
		if err = appendAudit(ctx, tx, scope, "contact.create", "contact", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "lead", leadID, "contact.created", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func scanProofTemplate(row rowScanner) (growth.ProofTemplate, error) {
	var item growth.ProofTemplate
	var inputSchema, outputSchema, accessPolicy []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.Name, &item.Type, &item.WorkflowVersionID, &inputSchema, &outputSchema, &accessPolicy, &item.RetentionDays, &item.Status, &item.Version, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	if item.InputSchema, err = unmarshalGrowthJSON(inputSchema); err != nil {
		return item, err
	}
	if item.OutputSchema, err = unmarshalGrowthJSON(outputSchema); err != nil {
		return item, err
	}
	item.AccessPolicy, err = unmarshalGrowthJSON(accessPolicy)
	return item, err
}

func scanProofRequest(row rowScanner) (growth.ProofRequest, error) {
	var item growth.ProofRequest
	var input []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.LeadID, &item.DealID, &item.TemplateID, &item.Status, &input, &item.RequestedBy, &item.ExpiresAt, &item.CreatedAt, &item.UpdatedAt)
	if err == nil {
		item.Input, err = unmarshalGrowthJSON(input)
	}
	return item, err
}

func scanProofInstance(row rowScanner) (growth.ProofInstance, error) {
	var item growth.ProofInstance
	var result []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.ProofRequestID, &item.Status, &result, &item.ArtifactRef, &item.ReviewRationale, &item.GeneratedBy, &item.ReviewedBy, &item.CreatedAt, &item.ReviewedAt, &item.ExpiresAt)
	if err == nil {
		item.Result, err = unmarshalGrowthJSON(result)
	}
	return item, err
}

func getProofRequest(ctx context.Context, tx pgx.Tx, tenantID, id string, lock bool) (growth.ProofRequest, error) {
	query := `SELECT id,tenant_id,lead_id,COALESCE(deal_id::text,''),template_id,status,input,requested_by,expires_at,created_at,updated_at FROM proof_requests WHERE tenant_id=$1 AND id=$2`
	if lock {
		query += ` FOR UPDATE`
	}
	return scanProofRequest(tx.QueryRow(ctx, query, tenantID, id))
}

func (s *Store) CreateProofTemplate(ctx context.Context, scope tenancy.Scope, input application.ProofTemplateInput, key string) (growth.ProofTemplate, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.ProofType = strings.ToLower(strings.TrimSpace(input.ProofType))
	if input.RetentionDays == 0 {
		input.RetentionDays = 30
	}
	template := growth.ProofTemplate{Name: input.Name, Type: input.ProofType, WorkflowVersionID: input.WorkflowVersionID}
	if input.Name == "" || !template.Valid() || input.RetentionDays < 1 || input.RetentionDays > 3650 {
		return growth.ProofTemplate{}, platform.Invalid("invalid_proof_template", "name, supported proof type, workflow version, and retention are required")
	}
	if err := schema.Validate(input.InputSchema); err != nil {
		return growth.ProofTemplate{}, platform.Invalid("invalid_input_schema", err.Error())
	}
	if err := schema.Validate(input.OutputSchema); err != nil {
		return growth.ProofTemplate{}, platform.Invalid("invalid_output_schema", err.Error())
	}
	inputSchema, err := json.Marshal(input.InputSchema)
	if err != nil {
		return growth.ProofTemplate{}, err
	}
	outputSchema, err := json.Marshal(input.OutputSchema)
	if err != nil {
		return growth.ProofTemplate{}, err
	}
	accessPolicy, err := marshalGrowthJSON(input.AccessPolicy)
	if err != nil {
		return growth.ProofTemplate{}, err
	}
	return runCommand(ctx, s, scope, key, "proof_template.create", func(tx pgx.Tx) (growth.ProofTemplate, string, error) {
		var workflowExists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workflow_definitions WHERE tenant_id=$1 AND id=$2)`, scope.TenantID, input.WorkflowVersionID).Scan(&workflowExists); err != nil {
			return growth.ProofTemplate{}, "", err
		}
		if !workflowExists {
			return growth.ProofTemplate{}, "", platform.Invalid("invalid_workflow", "proof workflow version must belong to the tenant")
		}
		item, err := scanProofTemplate(tx.QueryRow(ctx, `INSERT INTO proof_templates (tenant_id,name,proof_type,workflow_version_id,input_schema,output_schema,access_policy,retention_days,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id,tenant_id,name,proof_type,workflow_version_id,input_schema,output_schema,access_policy,retention_days,status,version,created_by,created_at,updated_at`, scope.TenantID, input.Name, input.ProofType, input.WorkflowVersionID, inputSchema, outputSchema, accessPolicy, input.RetentionDays, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"proof_type": item.Type, "workflow_version_id": item.WorkflowVersionID, "retention_days": item.RetentionDays}
		if err = appendAudit(ctx, tx, scope, "proof_template.create", "proof_template", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "proof_template", item.ID, "proof_template.created", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) CreateProofRequest(ctx context.Context, scope tenancy.Scope, leadID string, input application.ProofRequestInput, key string) (growth.ProofRequest, error) {
	if input.TemplateID == "" {
		return growth.ProofRequest{}, platform.Invalid("proof_template_required", "proof template is required")
	}
	inputJSON, err := marshalGrowthJSON(input.Input)
	if err != nil {
		return growth.ProofRequest{}, err
	}
	return runCommand(ctx, s, scope, key, "proof_request.create", func(tx pgx.Tx) (growth.ProofRequest, string, error) {
		lead, err := getGrowthLead(ctx, tx, scope.TenantID, leadID, true)
		if err != nil {
			return growth.ProofRequest{}, "", err
		}
		if lead.Status != "qualified" && lead.Status != "proof_requested" {
			return growth.ProofRequest{}, "", platform.Invalid("lead_not_proof_ready", "lead must be qualified before requesting proof")
		}
		var retentionDays int
		if err = tx.QueryRow(ctx, `SELECT retention_days FROM proof_templates WHERE tenant_id=$1 AND id=$2 AND status='active'`, scope.TenantID, input.TemplateID).Scan(&retentionDays); err != nil {
			return growth.ProofRequest{}, "", err
		}
		if input.DealID != "" {
			var dealMatches bool
			if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM deals WHERE tenant_id=$1 AND id=$2 AND lead_id=$3)`, scope.TenantID, input.DealID, leadID).Scan(&dealMatches); err != nil {
				return growth.ProofRequest{}, "", err
			}
			if !dealMatches {
				return growth.ProofRequest{}, "", platform.Invalid("deal_lead_mismatch", "proof deal must belong to the lead")
			}
		}
		if input.ExpiresAt.IsZero() {
			input.ExpiresAt = time.Now().UTC().Add(time.Duration(retentionDays) * 24 * time.Hour)
		}
		if !input.ExpiresAt.After(time.Now().UTC()) || input.ExpiresAt.After(time.Now().UTC().Add(time.Duration(retentionDays)*24*time.Hour)) {
			return growth.ProofRequest{}, "", platform.Invalid("invalid_proof_expiry", "proof expiry must be in the future and within template retention")
		}
		item, err := scanProofRequest(tx.QueryRow(ctx, `INSERT INTO proof_requests (tenant_id,lead_id,deal_id,template_id,input,requested_by,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id,tenant_id,lead_id,COALESCE(deal_id::text,''),template_id,status,input,requested_by,expires_at,created_at,updated_at`, scope.TenantID, leadID, nullableUUID(input.DealID), input.TemplateID, inputJSON, scope.ActorID, input.ExpiresAt.UTC()))
		if err != nil {
			return item, "", err
		}
		if lead.Status == "qualified" {
			if err = lead.Transition("proof_requested"); err != nil {
				return item, "", err
			}
			if _, err = tx.Exec(ctx, `UPDATE leads SET status=$3,version=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, leadID, lead.Status, lead.Version); err != nil {
				return item, "", err
			}
		}
		metadata := map[string]any{"lead_id": leadID, "template_id": item.TemplateID, "expires_at": item.ExpiresAt}
		if err = appendAudit(ctx, tx, scope, "proof_request.create", "proof_request", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "proof_request", item.ID, "proof.requested", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) GenerateProof(ctx context.Context, scope tenancy.Scope, requestID string, input application.ProofGenerationInput, key string) (growth.ProofInstance, error) {
	resultJSON, err := marshalGrowthJSON(input.Result)
	if err != nil {
		return growth.ProofInstance{}, err
	}
	return runCommand(ctx, s, scope, key, "proof.generate", func(tx pgx.Tx) (growth.ProofInstance, string, error) {
		request, err := getProofRequest(ctx, tx, scope.TenantID, requestID, true)
		if err != nil {
			return growth.ProofInstance{}, "", err
		}
		if request.ExpiresAt.Before(time.Now().UTC()) {
			return growth.ProofInstance{}, "", platform.Invalid("proof_expired", "expired proof request cannot be generated")
		}
		if err = request.Transition("processing"); err != nil {
			return growth.ProofInstance{}, "", err
		}
		if err = request.Transition("review"); err != nil {
			return growth.ProofInstance{}, "", err
		}
		item, err := scanProofInstance(tx.QueryRow(ctx, `INSERT INTO proof_instances (tenant_id,proof_request_id,status,result,artifact_ref,generated_by,expires_at) VALUES($1,$2,'generated',$3,$4,$5,$6) RETURNING id,tenant_id,proof_request_id,status,result,artifact_ref,review_rationale,generated_by,COALESCE(reviewed_by,''),created_at,reviewed_at,expires_at`, scope.TenantID, requestID, resultJSON, strings.TrimSpace(input.ArtifactRef), scope.ActorID, request.ExpiresAt))
		if err != nil {
			return item, "", err
		}
		if _, err = tx.Exec(ctx, `UPDATE proof_requests SET status='review',updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, requestID); err != nil {
			return item, "", err
		}
		metadata := map[string]any{"proof_request_id": requestID, "lead_id": request.LeadID, "artifact_ref": item.ArtifactRef}
		if err = appendAudit(ctx, tx, scope, "proof.generate", "proof_instance", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "proof_request", requestID, "proof.generated", 2, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) ReviewProof(ctx context.Context, scope tenancy.Scope, requestID string, input application.ProofReviewInput, key string) (growth.ProofInstance, error) {
	input.Decision = strings.ToLower(strings.TrimSpace(input.Decision))
	input.Rationale = strings.TrimSpace(input.Rationale)
	if (input.Decision != "approved" && input.Decision != "rejected") || input.Rationale == "" {
		return growth.ProofInstance{}, platform.Invalid("invalid_proof_review", "approved or rejected decision and rationale are required")
	}
	return runCommand(ctx, s, scope, key, "proof.review", func(tx pgx.Tx) (growth.ProofInstance, string, error) {
		request, err := getProofRequest(ctx, tx, scope.TenantID, requestID, true)
		if err != nil {
			return growth.ProofInstance{}, "", err
		}
		if request.Status != "review" {
			return growth.ProofInstance{}, "", platform.Invalid("proof_not_reviewable", "proof request must be awaiting review")
		}
		to := "ready"
		if input.Decision == "rejected" {
			to = "rejected"
		}
		if err = request.Transition(to); err != nil {
			return growth.ProofInstance{}, "", err
		}
		item, err := scanProofInstance(tx.QueryRow(ctx, `UPDATE proof_instances SET status=$3,review_rationale=$4,reviewed_by=$5,reviewed_at=now() WHERE tenant_id=$1 AND proof_request_id=$2 AND status='generated' RETURNING id,tenant_id,proof_request_id,status,result,artifact_ref,review_rationale,generated_by,COALESCE(reviewed_by,''),created_at,reviewed_at,expires_at`, scope.TenantID, requestID, input.Decision, input.Rationale, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		if _, err = tx.Exec(ctx, `UPDATE proof_requests SET status=$3,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, requestID, to); err != nil {
			return item, "", err
		}
		if input.Decision == "approved" {
			lead, leadErr := getGrowthLead(ctx, tx, scope.TenantID, request.LeadID, true)
			if leadErr != nil {
				return item, "", leadErr
			}
			if lead.Status == "proof_requested" {
				if leadErr = lead.Transition("proof_ready"); leadErr != nil {
					return item, "", leadErr
				}
				if _, leadErr = tx.Exec(ctx, `UPDATE leads SET status=$3,version=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, lead.ID, lead.Status, lead.Version); leadErr != nil {
					return item, "", leadErr
				}
			}
		}
		metadata := map[string]any{"proof_request_id": requestID, "decision": input.Decision, "rationale": input.Rationale}
		if err = appendAudit(ctx, tx, scope, "proof.review", "proof_instance", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "proof_request", requestID, "proof.reviewed", 3, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func scanCampaign(row rowScanner) (growth.Campaign, error) {
	var item growth.Campaign
	err := row.Scan(&item.ID, &item.TenantID, &item.MarketSegmentID, &item.Name, &item.Channel, &item.Purpose, &item.Status, &item.Version, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	item.Steps = []growth.CampaignStep{}
	return item, err
}

func scanCampaignStep(row rowScanner) (growth.CampaignStep, error) {
	var item growth.CampaignStep
	var definition []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.CampaignID, &item.Position, &item.Kind, &definition, &item.CreatedAt)
	if err == nil {
		item.Definition, err = unmarshalGrowthJSON(definition)
	}
	return item, err
}

func scanCampaignApproval(row rowScanner) (growth.CampaignApproval, error) {
	var item growth.CampaignApproval
	err := row.Scan(&item.ID, &item.TenantID, &item.CampaignID, &item.CampaignVersion, &item.Decision, &item.Rationale, &item.ReviewedBy, &item.CreatedAt)
	return item, err
}

func getCampaign(ctx context.Context, tx pgx.Tx, tenantID, id string, lock bool) (growth.Campaign, error) {
	query := `SELECT id,tenant_id,COALESCE(market_segment_id::text,''),name,channel,purpose,status,version,created_by,created_at,updated_at FROM campaigns WHERE tenant_id=$1 AND id=$2`
	if lock {
		query += ` FOR UPDATE`
	}
	return scanCampaign(tx.QueryRow(ctx, query, tenantID, id))
}

func (s *Store) CreateCampaign(ctx context.Context, scope tenancy.Scope, input application.CampaignInput, key string) (growth.Campaign, error) {
	input.Name, input.Channel, input.Purpose = strings.TrimSpace(input.Name), strings.ToLower(strings.TrimSpace(input.Channel)), strings.TrimSpace(input.Purpose)
	if input.Name == "" || input.Purpose == "" || !validGrowthChannel(input.Channel) {
		return growth.Campaign{}, platform.Invalid("invalid_campaign", "campaign name, channel, and purpose are required")
	}
	return runCommand(ctx, s, scope, key, "campaign.create", func(tx pgx.Tx) (growth.Campaign, string, error) {
		if input.MarketSegmentID != "" {
			var segmentExists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM market_segments WHERE tenant_id=$1 AND id=$2 AND status='active')`, scope.TenantID, input.MarketSegmentID).Scan(&segmentExists); err != nil {
				return growth.Campaign{}, "", err
			}
			if !segmentExists {
				return growth.Campaign{}, "", platform.Invalid("invalid_segment", "campaign segment must be active and tenant scoped")
			}
		}
		item, err := scanCampaign(tx.QueryRow(ctx, `INSERT INTO campaigns (tenant_id,market_segment_id,name,channel,purpose,created_by) VALUES($1,$2,$3,$4,$5,$6) RETURNING id,tenant_id,COALESCE(market_segment_id::text,''),name,channel,purpose,status,version,created_by,created_at,updated_at`, scope.TenantID, nullableUUID(input.MarketSegmentID), input.Name, input.Channel, input.Purpose, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"name": item.Name, "channel": item.Channel, "market_segment_id": item.MarketSegmentID}
		if err = appendAudit(ctx, tx, scope, "campaign.create", "campaign", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "campaign", item.ID, "campaign.created", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) AddCampaignStep(ctx context.Context, scope tenancy.Scope, campaignID string, input application.CampaignStepInput, key string) (growth.CampaignStep, error) {
	input.Kind = strings.ToLower(strings.TrimSpace(input.Kind))
	validKinds := map[string]bool{"message": true, "wait": true, "condition": true, "proof_request": true, "manual_task": true}
	if input.Position <= 0 || !validKinds[input.Kind] {
		return growth.CampaignStep{}, platform.Invalid("invalid_campaign_step", "positive position and supported step kind are required")
	}
	definition, err := marshalGrowthJSON(input.Definition)
	if err != nil {
		return growth.CampaignStep{}, err
	}
	return runCommand(ctx, s, scope, key, "campaign.step.create", func(tx pgx.Tx) (growth.CampaignStep, string, error) {
		campaign, err := getCampaign(ctx, tx, scope.TenantID, campaignID, true)
		if err != nil {
			return growth.CampaignStep{}, "", err
		}
		if campaign.Status != "draft" {
			return growth.CampaignStep{}, "", platform.Invalid("campaign_not_editable", "campaign steps can only change while draft")
		}
		item, err := scanCampaignStep(tx.QueryRow(ctx, `INSERT INTO campaign_steps (tenant_id,campaign_id,position,kind,definition) VALUES($1,$2,$3,$4,$5) RETURNING id,tenant_id,campaign_id,position,kind,definition,created_at`, scope.TenantID, campaignID, input.Position, input.Kind, definition))
		if err != nil {
			return item, "", err
		}
		campaign.Version++
		if _, err = tx.Exec(ctx, `UPDATE campaigns SET version=$3,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, campaignID, campaign.Version); err != nil {
			return item, "", err
		}
		metadata := map[string]any{"campaign_id": campaignID, "position": item.Position, "kind": item.Kind}
		if err = appendAudit(ctx, tx, scope, "campaign.step.create", "campaign_step", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "campaign", campaignID, "campaign.step_added", campaign.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) TransitionCampaign(ctx context.Context, scope tenancy.Scope, campaignID, to, key string) (growth.Campaign, error) {
	to = strings.ToLower(strings.TrimSpace(to))
	return runCommand(ctx, s, scope, key, "campaign.transition", func(tx pgx.Tx) (growth.Campaign, string, error) {
		item, err := getCampaign(ctx, tx, scope.TenantID, campaignID, true)
		if err != nil {
			return item, "", err
		}
		from := item.Status
		if to == "pending_approval" {
			var steps int
			if err = tx.QueryRow(ctx, `SELECT count(*) FROM campaign_steps WHERE tenant_id=$1 AND campaign_id=$2`, scope.TenantID, campaignID).Scan(&steps); err != nil {
				return item, "", err
			}
			if steps == 0 {
				return item, "", platform.Invalid("campaign_steps_required", "campaign requires at least one step before approval")
			}
		}
		if to == "active" {
			var approved bool
			if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM campaign_approvals WHERE tenant_id=$1 AND campaign_id=$2 AND campaign_version=$3 AND decision='approved')`, scope.TenantID, campaignID, item.Version).Scan(&approved); err != nil {
				return item, "", err
			}
			if !approved {
				return item, "", platform.Invalid("campaign_approval_required", "campaign version requires approval before activation")
			}
		}
		if err = item.Transition(to); err != nil {
			return item, "", err
		}
		if _, err = tx.Exec(ctx, `UPDATE campaigns SET status=$3,version=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, campaignID, item.Status, item.Version); err != nil {
			return item, "", err
		}
		metadata := map[string]any{"from": from, "to": to}
		if err = appendAudit(ctx, tx, scope, "campaign.transition", "campaign", campaignID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "campaign", campaignID, "campaign.transitioned", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) ReviewCampaign(ctx context.Context, scope tenancy.Scope, campaignID string, input application.CampaignApprovalInput, key string) (growth.Campaign, error) {
	input.Decision = strings.ToLower(strings.TrimSpace(input.Decision))
	input.Rationale = strings.TrimSpace(input.Rationale)
	if (input.Decision != "approved" && input.Decision != "rejected") || input.Rationale == "" {
		return growth.Campaign{}, platform.Invalid("invalid_campaign_review", "approved or rejected decision and rationale are required")
	}
	return runCommand(ctx, s, scope, key, "campaign.review", func(tx pgx.Tx) (growth.Campaign, string, error) {
		item, err := getCampaign(ctx, tx, scope.TenantID, campaignID, true)
		if err != nil {
			return item, "", err
		}
		if item.Status != "pending_approval" {
			return item, "", platform.Invalid("campaign_not_reviewable", "campaign must be pending approval")
		}
		if err = item.Transition(input.Decision); err != nil {
			return item, "", err
		}
		if _, err = tx.Exec(ctx, `UPDATE campaigns SET status=$3,version=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, campaignID, item.Status, item.Version); err != nil {
			return item, "", err
		}
		var approval growth.CampaignApproval
		err = tx.QueryRow(ctx, `INSERT INTO campaign_approvals (tenant_id,campaign_id,campaign_version,decision,rationale,reviewed_by) VALUES($1,$2,$3,$4,$5,$6) RETURNING id,tenant_id,campaign_id,campaign_version,decision,rationale,reviewed_by,created_at`, scope.TenantID, campaignID, item.Version, input.Decision, input.Rationale, scope.ActorID).Scan(&approval.ID, &approval.TenantID, &approval.CampaignID, &approval.CampaignVersion, &approval.Decision, &approval.Rationale, &approval.ReviewedBy, &approval.CreatedAt)
		if err != nil {
			return item, "", err
		}
		item.Approval = &approval
		metadata := map[string]any{"decision": input.Decision, "rationale": input.Rationale, "campaign_version": item.Version}
		if err = appendAudit(ctx, tx, scope, "campaign.review", "campaign", campaignID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "campaign", campaignID, "campaign.reviewed", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func scanSuppression(row rowScanner) (growth.SuppressionEntry, error) {
	var item growth.SuppressionEntry
	err := row.Scan(&item.ID, &item.TenantID, &item.SubjectType, &item.SubjectID, &item.Channel, &item.Reason, &item.Status, &item.SourceRef, &item.CreatedBy, &item.CreatedAt, &item.ReleasedAt)
	return item, err
}

func (s *Store) CreateSuppression(ctx context.Context, scope tenancy.Scope, input application.SuppressionInput, key string) (growth.SuppressionEntry, error) {
	input.Channel = strings.ToLower(strings.TrimSpace(input.Channel))
	input.Reason = strings.ToLower(strings.TrimSpace(input.Reason))
	if input.Channel == "" {
		input.Channel = "all"
	}
	validReason := map[string]bool{"do_not_contact": true, "opt_out": true, "bounce": true, "complaint": true, "manual": true, "risk": true}
	if (input.LeadID == "") == (input.ContactID == "") || (input.Channel != "all" && !validGrowthChannel(input.Channel)) || !validReason[input.Reason] {
		return growth.SuppressionEntry{}, platform.Invalid("invalid_suppression", "exactly one lead or contact, channel, and suppression reason are required")
	}
	return runCommand(ctx, s, scope, key, "suppression.create", func(tx pgx.Tx) (growth.SuppressionEntry, string, error) {
		subjectType, subjectID := "lead", input.LeadID
		if input.ContactID != "" {
			subjectType, subjectID = "contact", input.ContactID
			var leadID string
			if err := tx.QueryRow(ctx, `SELECT lead_id FROM contacts WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, input.ContactID).Scan(&leadID); err != nil {
				return growth.SuppressionEntry{}, "", err
			}
			if _, err := tx.Exec(ctx, `UPDATE contacts SET status='suppressed',consent_status='opted_out',updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, input.ContactID); err != nil {
				return growth.SuppressionEntry{}, "", err
			}
		} else {
			lead, err := getGrowthLead(ctx, tx, scope.TenantID, input.LeadID, true)
			if err != nil {
				return growth.SuppressionEntry{}, "", err
			}
			if lead.Status != "won" && lead.Status != "lost" && lead.Status != "suppressed" {
				if err = lead.Transition("suppressed"); err != nil {
					return growth.SuppressionEntry{}, "", err
				}
				if _, err = tx.Exec(ctx, `UPDATE leads SET status=$3,version=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, lead.ID, lead.Status, lead.Version); err != nil {
					return growth.SuppressionEntry{}, "", err
				}
			}
		}
		var item growth.SuppressionEntry
		err := tx.QueryRow(ctx, `INSERT INTO suppression_entries (tenant_id,subject_type,subject_id,subject_key,channel,reason,source_ref,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id,tenant_id,subject_type,subject_id,channel,reason,status,source_ref,created_by,created_at,released_at`, scope.TenantID, subjectType, subjectID, suppressionKey(scope.TenantID, subjectType, subjectID), input.Channel, input.Reason, strings.TrimSpace(input.SourceRef), scope.ActorID).Scan(&item.ID, &item.TenantID, &item.SubjectType, &item.SubjectID, &item.Channel, &item.Reason, &item.Status, &item.SourceRef, &item.CreatedBy, &item.CreatedAt, &item.ReleasedAt)
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"subject_type": subjectType, "subject_id": subjectID, "channel": input.Channel, "reason": input.Reason}
		if err = appendAudit(ctx, tx, scope, "suppression.create", "suppression_entry", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "suppression_entry", item.ID, "suppression.created", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func scanOutreach(row rowScanner) (growth.OutreachMessage, error) {
	var item growth.OutreachMessage
	var content []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.CampaignID, &item.CampaignStepID, &item.LeadID, &item.ContactID, &item.Status, &content, &item.BlockReason, &item.ExternalMessageID, &item.IdempotencyKey, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	if err == nil {
		item.Content, err = unmarshalGrowthJSON(content)
	}
	return item, err
}

func (s *Store) PlanOutreach(ctx context.Context, scope tenancy.Scope, campaignID string, input application.OutreachPlanInput, key string) (growth.OutreachMessage, error) {
	if input.LeadID == "" || input.StepID == "" {
		return growth.OutreachMessage{}, platform.Invalid("outreach_incomplete", "lead and campaign step are required")
	}
	content, err := marshalGrowthJSON(input.Content)
	if err != nil {
		return growth.OutreachMessage{}, err
	}
	return runCommand(ctx, s, scope, key, "outreach.plan", func(tx pgx.Tx) (growth.OutreachMessage, string, error) {
		campaign, err := getCampaign(ctx, tx, scope.TenantID, campaignID, true)
		if err != nil {
			return growth.OutreachMessage{}, "", err
		}
		if campaign.Status != "active" {
			return growth.OutreachMessage{}, "", platform.Invalid("campaign_inactive", "outreach requires an active approved campaign")
		}
		var stepKind string
		if err = tx.QueryRow(ctx, `SELECT kind FROM campaign_steps WHERE tenant_id=$1 AND id=$2 AND campaign_id=$3`, scope.TenantID, input.StepID, campaignID).Scan(&stepKind); err != nil {
			return growth.OutreachMessage{}, "", err
		}
		if stepKind != "message" {
			return growth.OutreachMessage{}, "", platform.Invalid("message_step_required", "outreach can only be planned from a message step")
		}
		lead, err := getGrowthLead(ctx, tx, scope.TenantID, input.LeadID, true)
		if err != nil {
			return growth.OutreachMessage{}, "", err
		}
		if lead.Status != "approved_for_outreach" && lead.Status != "contacted" && lead.Status != "replied" {
			return growth.OutreachMessage{}, "", platform.Invalid("lead_not_outreach_ready", "lead must be approved for outreach")
		}
		contactID := input.ContactID
		contactSuppressed := false
		if campaign.Channel == "email" || campaign.Channel == "phone" {
			if contactID == "" {
				return growth.OutreachMessage{}, "", platform.Invalid("contact_required", "campaign channel requires a contact")
			}
		}
		if contactID != "" {
			contact, contactErr := scanContact(tx.QueryRow(ctx, `SELECT id,tenant_id,lead_id,channel,value,normalized_value,status,consent_status,created_by,created_at,updated_at FROM contacts WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, contactID))
			if contactErr != nil {
				return growth.OutreachMessage{}, "", contactErr
			}
			if contact.LeadID != lead.ID || contact.Channel != campaign.Channel || contact.Status == "invalid" {
				return growth.OutreachMessage{}, "", platform.Invalid("contact_not_usable", "contact must belong to the lead, match the campaign channel, and not be invalid")
			}
			contactSuppressed = contact.Status == "suppressed" || contact.ConsentStatus == "opted_out"
		}

		blockedReason := ""
		if contactSuppressed {
			blockedReason = "suppressed"
		}
		leadKey := suppressionKey(scope.TenantID, "lead", lead.ID)
		contactKey := ""
		if contactID != "" {
			contactKey = suppressionKey(scope.TenantID, "contact", contactID)
		}
		var suppressed bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM suppression_entries WHERE tenant_id=$1 AND status='active' AND channel IN ('all',$2) AND (subject_key=$3 OR ($4<>'' AND subject_key=$4)))`, scope.TenantID, campaign.Channel, leadKey, contactKey).Scan(&suppressed); err != nil {
			return growth.OutreachMessage{}, "", err
		}
		if suppressed {
			blockedReason = "suppressed"
		}

		if blockedReason == "" {
			if _, err = tx.Exec(ctx, `INSERT INTO send_quotas (tenant_id,channel,period_start,period_end,limit_count) VALUES($1,$2,CURRENT_DATE,CURRENT_DATE,1000) ON CONFLICT DO NOTHING`, scope.TenantID, campaign.Channel); err != nil {
				return growth.OutreachMessage{}, "", err
			}
			var limitCount, reservedCount, consumedCount int
			if err = tx.QueryRow(ctx, `SELECT limit_count,reserved_count,consumed_count FROM send_quotas WHERE tenant_id=$1 AND channel=$2 AND period_start=CURRENT_DATE AND period_end=CURRENT_DATE FOR UPDATE`, scope.TenantID, campaign.Channel).Scan(&limitCount, &reservedCount, &consumedCount); err != nil {
				return growth.OutreachMessage{}, "", err
			}
			if reservedCount+consumedCount >= limitCount {
				blockedReason = "quota_exceeded"
			} else if _, err = tx.Exec(ctx, `UPDATE send_quotas SET reserved_count=reserved_count+1 WHERE tenant_id=$1 AND channel=$2 AND period_start=CURRENT_DATE AND period_end=CURRENT_DATE`, scope.TenantID, campaign.Channel); err != nil {
				return growth.OutreachMessage{}, "", err
			}
		}

		status := "planned"
		if blockedReason != "" {
			status = "blocked"
		}
		item, err := scanOutreach(tx.QueryRow(ctx, `INSERT INTO outreach_messages (tenant_id,campaign_id,campaign_step_id,lead_id,contact_id,status,content,block_reason,idempotency_key,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id,tenant_id,campaign_id,campaign_step_id,lead_id,COALESCE(contact_id::text,''),status,content,block_reason,COALESCE(external_message_id,''),idempotency_key,created_by,created_at,updated_at`, scope.TenantID, campaignID, input.StepID, lead.ID, nullableUUID(contactID), status, content, blockedReason, key, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"campaign_id": campaignID, "lead_id": lead.ID, "status": item.Status, "block_reason": item.BlockReason}
		if err = appendAudit(ctx, tx, scope, "outreach.plan", "outreach_message", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		eventType := "outreach.planned"
		if item.Status == "blocked" {
			eventType = "outreach.blocked"
		}
		if err = appendEvent(ctx, tx, scope, "outreach_message", item.ID, eventType, 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) TransitionOutreach(ctx context.Context, scope tenancy.Scope, messageID string, input application.OutreachTransitionInput, key string) (growth.OutreachMessage, error) {
	input.To = strings.ToLower(strings.TrimSpace(input.To))
	return runCommand(ctx, s, scope, key, "outreach.transition", func(tx pgx.Tx) (growth.OutreachMessage, string, error) {
		var channel string
		item, err := scanOutreach(tx.QueryRow(ctx, `SELECT message.id,message.tenant_id,message.campaign_id,message.campaign_step_id,message.lead_id,COALESCE(message.contact_id::text,''),message.status,message.content,message.block_reason,COALESCE(message.external_message_id,''),message.idempotency_key,message.created_by,message.created_at,message.updated_at FROM outreach_messages message WHERE message.tenant_id=$1 AND message.id=$2 FOR UPDATE`, scope.TenantID, messageID))
		if err != nil {
			return item, "", err
		}
		if err = tx.QueryRow(ctx, `SELECT channel FROM campaigns WHERE tenant_id=$1 AND id=$2`, scope.TenantID, item.CampaignID).Scan(&channel); err != nil {
			return item, "", err
		}
		from := item.Status
		if err = item.Transition(input.To); err != nil {
			return item, "", err
		}
		if input.ExternalMessageID != "" {
			item.ExternalMessageID = strings.TrimSpace(input.ExternalMessageID)
		}
		if input.To == "sent" {
			if _, err = tx.Exec(ctx, `UPDATE send_quotas SET reserved_count=GREATEST(reserved_count-1,0),consumed_count=consumed_count+1 WHERE tenant_id=$1 AND channel=$2 AND period_start=CURRENT_DATE AND period_end=CURRENT_DATE`, scope.TenantID, channel); err != nil {
				return item, "", err
			}
		} else if input.To == "cancelled" {
			if _, err = tx.Exec(ctx, `UPDATE send_quotas SET reserved_count=GREATEST(reserved_count-1,0) WHERE tenant_id=$1 AND channel=$2 AND period_start=CURRENT_DATE AND period_end=CURRENT_DATE`, scope.TenantID, channel); err != nil {
				return item, "", err
			}
		}
		if _, err = tx.Exec(ctx, `UPDATE outreach_messages SET status=$3,external_message_id=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, messageID, item.Status, nullableText(item.ExternalMessageID)); err != nil {
			return item, "", err
		}

		lead, err := getGrowthLead(ctx, tx, scope.TenantID, item.LeadID, true)
		if err != nil {
			return item, "", err
		}
		leadChanged := false
		if input.To == "sent" && lead.Status == "approved_for_outreach" {
			err = lead.Transition("contacted")
			leadChanged = err == nil
		} else if input.To == "replied" && lead.Status == "contacted" {
			err = lead.Transition("replied")
			leadChanged = err == nil
		}
		if err != nil {
			return item, "", err
		}
		if input.To == "bounced" || input.To == "complained" {
			subjectType, subjectID := "lead", lead.ID
			if item.ContactID != "" {
				subjectType, subjectID = "contact", item.ContactID
				if _, err = tx.Exec(ctx, `UPDATE contacts SET status='suppressed',consent_status='opted_out',updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, item.ContactID); err != nil {
					return item, "", err
				}
			}
			reason := "bounce"
			if input.To == "complained" {
				reason = "complaint"
			}
			if _, err = tx.Exec(ctx, `INSERT INTO suppression_entries (tenant_id,subject_type,subject_id,subject_key,channel,reason,source_ref,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT DO NOTHING`, scope.TenantID, subjectType, subjectID, suppressionKey(scope.TenantID, subjectType, subjectID), channel, reason, messageID, scope.ActorID); err != nil {
				return item, "", err
			}
			if lead.Status != "won" && lead.Status != "lost" && lead.Status != "suppressed" {
				if err = lead.Transition("suppressed"); err != nil {
					return item, "", err
				}
				leadChanged = true
			}
		}
		if leadChanged {
			if _, err = tx.Exec(ctx, `UPDATE leads SET status=$3,version=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2`, scope.TenantID, lead.ID, lead.Status, lead.Version); err != nil {
				return item, "", err
			}
		}
		metadata := map[string]any{"from": from, "to": item.Status, "lead_id": item.LeadID, "external_message_id": item.ExternalMessageID}
		if err = appendAudit(ctx, tx, scope, "outreach.transition", "outreach_message", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "outreach_message", item.ID, "outreach.transitioned", 2, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

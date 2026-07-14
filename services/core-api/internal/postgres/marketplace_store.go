package postgres

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/application"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/marketplace"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

func scanDeveloper(row rowScanner) (marketplace.Developer, error) {
	var item marketplace.Developer
	err := row.Scan(&item.ID, &item.TenantID, &item.Name, &item.Status, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanPublisher(row rowScanner) (marketplace.Publisher, error) {
	var item marketplace.Publisher
	err := row.Scan(&item.ID, &item.TenantID, &item.DeveloperID, &item.Name, &item.Status, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanListing(row rowScanner) (marketplace.Listing, error) {
	var item marketplace.Listing
	err := row.Scan(&item.ID, &item.TenantID, &item.PublisherID, &item.Name, &item.Type, &item.Status, &item.Version, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	item.Versions = []marketplace.ListingVersion{}
	return item, err
}

func scanListingVersion(row rowScanner) (marketplace.ListingVersion, error) {
	var item marketplace.ListingVersion
	var capabilityManifest, permissionManifest []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.ListingID, &item.Version, &capabilityManifest, &permissionManifest, &item.ContentRef, &item.Checksum, &item.CreatedBy, &item.CreatedAt)
	if err != nil {
		return item, err
	}
	if item.CapabilityManifest, err = unmarshalChannelJSON(capabilityManifest); err != nil {
		return item, err
	}
	item.PermissionManifest, err = unmarshalChannelJSON(permissionManifest)
	return item, err
}

func scanListingReview(row rowScanner) (marketplace.Review, error) {
	var item marketplace.Review
	err := row.Scan(&item.ID, &item.TenantID, &item.ListingID, &item.ListingVersionID, &item.Type, &item.Decision, &item.Rationale, &item.ReviewedBy, &item.CreatedAt)
	return item, err
}

func scanSandboxRun(row rowScanner) (marketplace.SandboxRun, error) {
	var item marketplace.SandboxRun
	var policy, result []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.ListingVersionID, &item.Status, &policy, &result, &item.CreatedBy, &item.StartedAt, &item.CompletedAt)
	if err != nil {
		return item, err
	}
	if item.Policy, err = unmarshalChannelJSON(policy); err != nil {
		return item, err
	}
	item.Result, err = unmarshalChannelJSON(result)
	return item, err
}

func scanQualityScore(row rowScanner) (marketplace.QualityScore, error) {
	var item marketplace.QualityScore
	var dimensions []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.ListingVersionID, &item.ScoreBPS, &dimensions, &item.CreatedBy, &item.CreatedAt)
	if err == nil {
		item.Dimensions, err = unmarshalChannelJSON(dimensions)
	}
	return item, err
}

func scanIncident(row rowScanner) (marketplace.IncidentRecord, error) {
	var item marketplace.IncidentRecord
	err := row.Scan(&item.ID, &item.TenantID, &item.ListingID, &item.Severity, &item.Summary, &item.Status, &item.CreatedBy, &item.CreatedAt, &item.ResolvedAt)
	return item, err
}

func scanRevenueShareRule(row rowScanner) (marketplace.RevenueShareRule, error) {
	var item marketplace.RevenueShareRule
	err := row.Scan(&item.ID, &item.TenantID, &item.ListingID, &item.PublisherID, &item.BasisPoints, &item.Currency, &item.Status, &item.Version, &item.CreatedBy, &item.CreatedAt)
	item.Currency = strings.TrimSpace(item.Currency)
	return item, err
}

func scanPayoutReserve(row rowScanner) (marketplace.PayoutReserve, error) {
	var item marketplace.PayoutReserve
	err := row.Scan(&item.ID, &item.TenantID, &item.ListingID, &item.PublisherID, &item.Currency, &item.AmountMinor, &item.Status, &item.ReferenceType, &item.ReferenceID, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	item.Currency = strings.TrimSpace(item.Currency)
	return item, err
}

func scanMarketplaceDispute(row rowScanner) (marketplace.Dispute, error) {
	var item marketplace.Dispute
	var resolution []byte
	err := row.Scan(&item.ID, &item.TenantID, &item.ListingID, &item.ClaimantType, &item.ClaimantID, &item.Reason, &item.Status, &resolution, &item.CreatedBy, &item.ResolvedBy, &item.CreatedAt, &item.ResolvedAt)
	if err == nil {
		item.Resolution, err = unmarshalChannelJSON(resolution)
	}
	return item, err
}

func scanTakedown(row rowScanner) (marketplace.Takedown, error) {
	var item marketplace.Takedown
	err := row.Scan(&item.ID, &item.TenantID, &item.ListingID, &item.Reason, &item.Status, &item.RequestedBy, &item.ReviewedBy, &item.CreatedAt, &item.ReviewedAt)
	return item, err
}

func listMarketplace(ctx context.Context, tx pgx.Tx, tenantID string) (marketplace.Overview, error) {
	var result marketplace.Overview
	var err error
	result.Developers, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,name,status,created_by,created_at,updated_at FROM developers WHERE tenant_id=$1 ORDER BY updated_at DESC,id`, scanDeveloper, tenantID)
	if err != nil {
		return result, err
	}
	result.Publishers, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,developer_id,name,status,created_by,created_at,updated_at FROM publishers WHERE tenant_id=$1 ORDER BY updated_at DESC,id`, scanPublisher, tenantID)
	if err != nil {
		return result, err
	}
	result.Listings, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,publisher_id,name,listing_type,status,version,created_by,created_at,updated_at FROM listings WHERE tenant_id=$1 ORDER BY updated_at DESC,id`, scanListing, tenantID)
	if err != nil {
		return result, err
	}
	for index := range result.Listings {
		result.Listings[index].Versions, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,listing_id,version,capability_manifest,permission_manifest,content_ref,checksum,created_by,created_at FROM listing_versions WHERE tenant_id=$1 AND listing_id=$2 ORDER BY version DESC`, scanListingVersion, tenantID, result.Listings[index].ID)
		if err != nil {
			return result, err
		}
	}
	result.Reviews, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,listing_id,listing_version_id,review_type,decision,rationale,reviewed_by,created_at FROM listing_reviews WHERE tenant_id=$1 ORDER BY created_at DESC,id`, scanListingReview, tenantID)
	if err != nil {
		return result, err
	}
	result.SandboxRuns, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,listing_version_id,status,policy,result,created_by,started_at,completed_at FROM sandbox_runs WHERE tenant_id=$1 ORDER BY started_at DESC,id`, scanSandboxRun, tenantID)
	if err != nil {
		return result, err
	}
	result.QualityScores, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,listing_version_id,score_bps,dimensions,created_by,created_at FROM quality_scores WHERE tenant_id=$1 ORDER BY created_at DESC,id`, scanQualityScore, tenantID)
	if err != nil {
		return result, err
	}
	result.Incidents, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,listing_id,severity,summary,status,created_by,created_at,resolved_at FROM incident_records WHERE tenant_id=$1 ORDER BY created_at DESC,id`, scanIncident, tenantID)
	if err != nil {
		return result, err
	}
	result.RevenueShareRules, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,listing_id,publisher_id,basis_points,currency,status,version,created_by,created_at FROM revenue_share_rules WHERE tenant_id=$1 ORDER BY created_at DESC,id`, scanRevenueShareRule, tenantID)
	if err != nil {
		return result, err
	}
	result.PayoutReserves, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,listing_id,publisher_id,currency,amount_minor,status,reference_type,reference_id,created_by,created_at,updated_at FROM payout_reserves WHERE tenant_id=$1 ORDER BY created_at DESC,id`, scanPayoutReserve, tenantID)
	if err != nil {
		return result, err
	}
	result.Disputes, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,listing_id,claimant_type,claimant_id,reason,status,resolution,created_by,COALESCE(resolved_by,''),created_at,resolved_at FROM marketplace_disputes WHERE tenant_id=$1 ORDER BY created_at DESC,id`, scanMarketplaceDispute, tenantID)
	if err != nil {
		return result, err
	}
	result.Takedowns, err = collectChannelRows(ctx, tx, `SELECT id,tenant_id,listing_id,reason,status,requested_by,COALESCE(reviewed_by,''),created_at,reviewed_at FROM takedowns WHERE tenant_id=$1 ORDER BY created_at DESC,id`, scanTakedown, tenantID)
	return result, err
}

func (s *Store) CreateDeveloper(ctx context.Context, scope tenancy.Scope, input application.DeveloperInput, key string) (marketplace.Developer, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return marketplace.Developer{}, platform.Invalid("developer_name_required", "developer name is required")
	}
	return runCommand(ctx, s, scope, key, "developer.create", func(tx pgx.Tx) (marketplace.Developer, string, error) {
		item, err := scanDeveloper(tx.QueryRow(ctx, `INSERT INTO developers (tenant_id,name,created_by) VALUES($1,$2,$3) RETURNING id,tenant_id,name,status,created_by,created_at,updated_at`, scope.TenantID, input.Name, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"name": item.Name}
		if err = appendAudit(ctx, tx, scope, "developer.create", "developer", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "developer", item.ID, "developer.created", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) CreatePublisher(ctx context.Context, scope tenancy.Scope, input application.PublisherInput, key string) (marketplace.Publisher, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.DeveloperID == "" || input.Name == "" {
		return marketplace.Publisher{}, platform.Invalid("invalid_publisher", "developer and publisher name are required")
	}
	return runCommand(ctx, s, scope, key, "publisher.create", func(tx pgx.Tx) (marketplace.Publisher, string, error) {
		item, err := scanPublisher(tx.QueryRow(ctx, `INSERT INTO publishers (tenant_id,developer_id,name,created_by) SELECT $1,id,$3,$4 FROM developers WHERE tenant_id=$1 AND id=$2 AND status='active' RETURNING id,tenant_id,developer_id,name,status,created_by,created_at,updated_at`, scope.TenantID, input.DeveloperID, input.Name, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"developer_id": item.DeveloperID, "name": item.Name}
		if err = appendAudit(ctx, tx, scope, "publisher.create", "publisher", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "publisher", item.ID, "publisher.created", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) CreateListing(ctx context.Context, scope tenancy.Scope, input application.ListingInput, key string) (marketplace.Listing, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.ListingType = strings.ToLower(strings.TrimSpace(input.ListingType))
	if input.PublisherID == "" || input.Name == "" || !marketplace.ValidListingType(input.ListingType) {
		return marketplace.Listing{}, platform.Invalid("invalid_listing", "publisher, name, and supported listing type are required")
	}
	return runCommand(ctx, s, scope, key, "listing.create", func(tx pgx.Tx) (marketplace.Listing, string, error) {
		item, err := scanListing(tx.QueryRow(ctx, `INSERT INTO listings (tenant_id,publisher_id,name,listing_type,created_by) SELECT $1,id,$3,$4,$5 FROM publishers WHERE tenant_id=$1 AND id=$2 AND status='active' RETURNING id,tenant_id,publisher_id,name,listing_type,status,version,created_by,created_at,updated_at`, scope.TenantID, input.PublisherID, input.Name, input.ListingType, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"publisher_id": item.PublisherID, "listing_type": item.Type, "name": item.Name}
		if err = appendAudit(ctx, tx, scope, "listing.create", "listing", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "listing", item.ID, "listing.created", item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func getListing(ctx context.Context, tx pgx.Tx, tenantID, id string, lock bool) (marketplace.Listing, error) {
	query := `SELECT id,tenant_id,publisher_id,name,listing_type,status,version,created_by,created_at,updated_at FROM listings WHERE tenant_id=$1 AND id=$2`
	if lock {
		query += ` FOR UPDATE`
	}
	return scanListing(tx.QueryRow(ctx, query, tenantID, id))
}

func (s *Store) CreateListingVersion(ctx context.Context, scope tenancy.Scope, listingID string, input application.ListingVersionInput, key string) (marketplace.ListingVersion, error) {
	input.ContentRef = strings.TrimSpace(input.ContentRef)
	input.Checksum = strings.TrimSpace(input.Checksum)
	if listingID == "" || input.Checksum == "" || len(input.CapabilityManifest) == 0 || len(input.PermissionManifest) == 0 {
		return marketplace.ListingVersion{}, platform.Invalid("invalid_listing_version", "checksum and capability and permission manifests are required")
	}
	capabilityManifest, err := marshalChannelJSON(input.CapabilityManifest)
	if err != nil {
		return marketplace.ListingVersion{}, err
	}
	permissionManifest, err := marshalChannelJSON(input.PermissionManifest)
	if err != nil {
		return marketplace.ListingVersion{}, err
	}
	return runCommand(ctx, s, scope, key, "listing_version.create", func(tx pgx.Tx) (marketplace.ListingVersion, string, error) {
		listing, err := getListing(ctx, tx, scope.TenantID, listingID, true)
		if err != nil {
			return marketplace.ListingVersion{}, "", err
		}
		if listing.Status != "draft" {
			return marketplace.ListingVersion{}, "", platform.Invalid("listing_version_locked", "listing versions can only be added while the listing is draft")
		}
		var version int
		if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(version),0)+1 FROM listing_versions WHERE tenant_id=$1 AND listing_id=$2`, scope.TenantID, listingID).Scan(&version); err != nil {
			return marketplace.ListingVersion{}, "", err
		}
		item, err := scanListingVersion(tx.QueryRow(ctx, `INSERT INTO listing_versions (tenant_id,listing_id,version,capability_manifest,permission_manifest,content_ref,checksum,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id,tenant_id,listing_id,version,capability_manifest,permission_manifest,content_ref,checksum,created_by,created_at`, scope.TenantID, listingID, version, capabilityManifest, permissionManifest, input.ContentRef, input.Checksum, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"listing_id": listingID, "version": item.Version, "checksum": item.Checksum}
		if err = appendAudit(ctx, tx, scope, "listing_version.create", "listing_version", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "listing", listingID, "listing.version_created", listing.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func latestListingVersionID(ctx context.Context, tx pgx.Tx, tenantID, listingID string) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `SELECT id FROM listing_versions WHERE tenant_id=$1 AND listing_id=$2 ORDER BY version DESC LIMIT 1`, tenantID, listingID).Scan(&id)
	return id, err
}

func listingGate(ctx context.Context, tx pgx.Tx, tenantID, listingID, from, to string) error {
	versionID, err := latestListingVersionID(ctx, tx, tenantID, listingID)
	if err != nil {
		return platform.Invalid("listing_version_required", "listing requires an immutable version")
	}
	if from == "automated_review" && to == "manual_review" {
		var approved bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM listing_reviews WHERE tenant_id=$1 AND listing_version_id=$2 AND review_type='automated' AND decision='approved')`, tenantID, versionID).Scan(&approved); err != nil {
			return err
		}
		if !approved {
			return platform.Invalid("automated_review_gate", "approved automated review is required")
		}
	}
	if from == "manual_review" && to == "sandbox_testing" {
		var approvedCount, rejectedCount int
		if err = tx.QueryRow(ctx, `SELECT COUNT(*) FILTER (WHERE review_type IN ('security','license','manual') AND decision='approved'),COUNT(*) FILTER (WHERE decision IN ('rejected','changes_requested')) FROM listing_reviews WHERE tenant_id=$1 AND listing_version_id=$2`, tenantID, versionID).Scan(&approvedCount, &rejectedCount); err != nil {
			return err
		}
		if approvedCount != 3 || rejectedCount != 0 {
			return platform.Invalid("manual_review_gate", "security, license, and manual reviews must approve the latest version")
		}
	}
	if (from == "sandbox_testing" && to == "limited_release") || to == "published" {
		var sandboxPassed, qualityPassed bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sandbox_runs WHERE tenant_id=$1 AND listing_version_id=$2 AND status='succeeded'),EXISTS(SELECT 1 FROM quality_scores WHERE tenant_id=$1 AND listing_version_id=$2 AND score_bps>=8000)`, tenantID, versionID).Scan(&sandboxPassed, &qualityPassed); err != nil {
			return err
		}
		if !sandboxPassed || !qualityPassed {
			return platform.Invalid("listing_release_gate", "successful sandbox and quality score of at least 8000 are required")
		}
	}
	return nil
}

func (s *Store) TransitionListing(ctx context.Context, scope tenancy.Scope, id, to, key string) (marketplace.Listing, error) {
	to = strings.ToLower(strings.TrimSpace(to))
	if to == "removed" {
		return marketplace.Listing{}, platform.Invalid("takedown_required", "listing removal requires an approved takedown")
	}
	return runCommand(ctx, s, scope, key, "listing.transition", func(tx pgx.Tx) (marketplace.Listing, string, error) {
		item, err := getListing(ctx, tx, scope.TenantID, id, true)
		if err != nil {
			return item, "", err
		}
		from := item.Status
		if err = listingGate(ctx, tx, scope.TenantID, id, from, to); err != nil {
			return item, "", err
		}
		if err = item.Transition(to); err != nil {
			return item, "", err
		}
		item, err = scanListing(tx.QueryRow(ctx, `UPDATE listings SET status=$3,version=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2 RETURNING id,tenant_id,publisher_id,name,listing_type,status,version,created_by,created_at,updated_at`, scope.TenantID, id, item.Status, item.Version))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"from": from, "to": item.Status}
		if err = appendAudit(ctx, tx, scope, "listing.transition", "listing", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "listing", item.ID, "listing."+item.Status, item.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) ReviewListing(ctx context.Context, scope tenancy.Scope, listingID string, input application.ListingReviewInput, key string) (marketplace.Review, error) {
	input.ReviewType = strings.ToLower(strings.TrimSpace(input.ReviewType))
	input.Decision = strings.ToLower(strings.TrimSpace(input.Decision))
	input.Rationale = strings.TrimSpace(input.Rationale)
	validType := input.ReviewType == "automated" || input.ReviewType == "security" || input.ReviewType == "license" || input.ReviewType == "manual"
	validDecision := input.Decision == "approved" || input.Decision == "rejected" || input.Decision == "changes_requested"
	if listingID == "" || input.ListingVersionID == "" || !validType || !validDecision || input.Rationale == "" {
		return marketplace.Review{}, platform.Invalid("invalid_listing_review", "listing version, review type, decision, and rationale are required")
	}
	return runCommand(ctx, s, scope, key, "listing.review", func(tx pgx.Tx) (marketplace.Review, string, error) {
		listing, err := getListing(ctx, tx, scope.TenantID, listingID, true)
		if err != nil {
			return marketplace.Review{}, "", err
		}
		if listing.Status != "automated_review" && listing.Status != "manual_review" {
			return marketplace.Review{}, "", platform.Invalid("listing_not_in_review", "listing must be in a review state")
		}
		if input.ReviewType == "automated" && listing.Status != "automated_review" || input.ReviewType != "automated" && listing.Status != "manual_review" {
			return marketplace.Review{}, "", platform.Invalid("review_stage_mismatch", "review type does not match the listing stage")
		}
		var latestID, versionCreator string
		if err = tx.QueryRow(ctx, `SELECT id,created_by FROM listing_versions WHERE tenant_id=$1 AND listing_id=$2 ORDER BY version DESC LIMIT 1`, scope.TenantID, listingID).Scan(&latestID, &versionCreator); err != nil {
			return marketplace.Review{}, "", err
		}
		if latestID != input.ListingVersionID {
			return marketplace.Review{}, "", platform.Invalid("review_version_stale", "review must target the latest listing version")
		}
		if listing.CreatedBy == scope.ActorID || versionCreator == scope.ActorID {
			return marketplace.Review{}, "", platform.ErrPermissionDenied
		}
		item, err := scanListingReview(tx.QueryRow(ctx, `INSERT INTO listing_reviews (tenant_id,listing_id,listing_version_id,review_type,decision,rationale,reviewed_by) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id,tenant_id,listing_id,listing_version_id,review_type,decision,rationale,reviewed_by,created_at`, scope.TenantID, listingID, input.ListingVersionID, input.ReviewType, input.Decision, input.Rationale, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"listing_version_id": item.ListingVersionID, "review_type": item.Type, "decision": item.Decision}
		if err = appendAudit(ctx, tx, scope, "listing.review", "listing_review", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "listing", listingID, "listing.reviewed", listing.Version, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) RunSandbox(ctx context.Context, scope tenancy.Scope, input application.SandboxRunInput, key string) (marketplace.SandboxRun, error) {
	input.Status = strings.ToLower(strings.TrimSpace(input.Status))
	if input.ListingVersionID == "" || (input.Status != "succeeded" && input.Status != "failed") || len(input.Policy) == 0 || len(input.Result) == 0 {
		return marketplace.SandboxRun{}, platform.Invalid("invalid_sandbox_run", "listing version, terminal status, policy, and result are required")
	}
	policy, err := marshalChannelJSON(input.Policy)
	if err != nil {
		return marketplace.SandboxRun{}, err
	}
	result, err := marshalChannelJSON(input.Result)
	if err != nil {
		return marketplace.SandboxRun{}, err
	}
	return runCommand(ctx, s, scope, key, "listing.sandbox", func(tx pgx.Tx) (marketplace.SandboxRun, string, error) {
		var listingID string
		if err := tx.QueryRow(ctx, `SELECT l.id FROM listing_versions lv JOIN listings l ON l.tenant_id=lv.tenant_id AND l.id=lv.listing_id WHERE lv.tenant_id=$1 AND lv.id=$2 AND l.status='sandbox_testing' AND lv.version=(SELECT MAX(version) FROM listing_versions WHERE tenant_id=$1 AND listing_id=l.id)`, scope.TenantID, input.ListingVersionID).Scan(&listingID); err != nil {
			return marketplace.SandboxRun{}, "", err
		}
		item, err := scanSandboxRun(tx.QueryRow(ctx, `INSERT INTO sandbox_runs (tenant_id,listing_version_id,status,policy,result,created_by,completed_at) VALUES($1,$2,$3,$4,$5,$6,now()) RETURNING id,tenant_id,listing_version_id,status,policy,result,created_by,started_at,completed_at`, scope.TenantID, input.ListingVersionID, input.Status, policy, result, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"listing_version_id": item.ListingVersionID, "status": item.Status}
		if err = appendAudit(ctx, tx, scope, "listing.sandbox", "sandbox_run", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "listing", listingID, "listing.sandbox_completed", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) RecordListingQuality(ctx context.Context, scope tenancy.Scope, input application.ListingQualityInput, key string) (marketplace.QualityScore, error) {
	if input.ListingVersionID == "" || input.ScoreBPS < 0 || input.ScoreBPS > 10000 || len(input.Dimensions) == 0 {
		return marketplace.QualityScore{}, platform.Invalid("invalid_listing_quality", "listing version, score, and dimensions are required")
	}
	dimensions, err := marshalChannelJSON(input.Dimensions)
	if err != nil {
		return marketplace.QualityScore{}, err
	}
	return runCommand(ctx, s, scope, key, "listing.quality", func(tx pgx.Tx) (marketplace.QualityScore, string, error) {
		var listingID string
		if err := tx.QueryRow(ctx, `SELECT l.id FROM listing_versions lv JOIN listings l ON l.tenant_id=lv.tenant_id AND l.id=lv.listing_id WHERE lv.tenant_id=$1 AND lv.id=$2 AND l.status='sandbox_testing'`, scope.TenantID, input.ListingVersionID).Scan(&listingID); err != nil {
			return marketplace.QualityScore{}, "", err
		}
		item, err := scanQualityScore(tx.QueryRow(ctx, `INSERT INTO quality_scores (tenant_id,listing_version_id,score_bps,dimensions,created_by) VALUES($1,$2,$3,$4,$5) RETURNING id,tenant_id,listing_version_id,score_bps,dimensions,created_by,created_at`, scope.TenantID, input.ListingVersionID, input.ScoreBPS, dimensions, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"listing_version_id": item.ListingVersionID, "score_bps": item.ScoreBPS}
		if err = appendAudit(ctx, tx, scope, "listing.quality", "quality_score", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "listing", listingID, "listing.quality_recorded", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) CreateMarketplaceDispute(ctx context.Context, scope tenancy.Scope, input application.MarketplaceDisputeInput, key string) (marketplace.Dispute, error) {
	input.ClaimantType = strings.ToLower(strings.TrimSpace(input.ClaimantType))
	input.ClaimantID = strings.TrimSpace(input.ClaimantID)
	input.Reason = strings.TrimSpace(input.Reason)
	validClaimant := input.ClaimantType == "developer" || input.ClaimantType == "publisher" || input.ClaimantType == "customer" || input.ClaimantType == "supplier" || input.ClaimantType == "reseller" || input.ClaimantType == "platform"
	if input.ListingID == "" || !validClaimant || input.ClaimantID == "" || input.Reason == "" {
		return marketplace.Dispute{}, platform.Invalid("invalid_marketplace_dispute", "listing, claimant, and reason are required")
	}
	return runCommand(ctx, s, scope, key, "marketplace_dispute.create", func(tx pgx.Tx) (marketplace.Dispute, string, error) {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM listings WHERE tenant_id=$1 AND id=$2)`, scope.TenantID, input.ListingID).Scan(&exists); err != nil {
			return marketplace.Dispute{}, "", err
		}
		if !exists {
			return marketplace.Dispute{}, "", platform.Invalid("listing_not_found", "listing must belong to the tenant")
		}
		item, err := scanMarketplaceDispute(tx.QueryRow(ctx, `INSERT INTO marketplace_disputes (tenant_id,listing_id,claimant_type,claimant_id,reason,created_by) VALUES($1,$2,$3,$4,$5,$6) RETURNING id,tenant_id,listing_id,claimant_type,claimant_id,reason,status,resolution,created_by,COALESCE(resolved_by,''),created_at,resolved_at`, scope.TenantID, input.ListingID, input.ClaimantType, input.ClaimantID, input.Reason, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"listing_id": item.ListingID, "claimant_type": item.ClaimantType}
		if err = appendAudit(ctx, tx, scope, "marketplace_dispute.create", "marketplace_dispute", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "listing", item.ListingID, "marketplace_dispute.created", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) ResolveMarketplaceDispute(ctx context.Context, scope tenancy.Scope, id string, input application.DisputeResolutionInput, key string) (marketplace.Dispute, error) {
	input.Decision = strings.ToLower(strings.TrimSpace(input.Decision))
	if (input.Decision != "resolved" && input.Decision != "rejected") || len(input.Resolution) == 0 {
		return marketplace.Dispute{}, platform.Invalid("invalid_dispute_resolution", "decision and structured resolution are required")
	}
	resolution, err := marshalChannelJSON(input.Resolution)
	if err != nil {
		return marketplace.Dispute{}, err
	}
	return runCommand(ctx, s, scope, key, "marketplace_dispute.resolve", func(tx pgx.Tx) (marketplace.Dispute, string, error) {
		var creator string
		if err := tx.QueryRow(ctx, `SELECT created_by FROM marketplace_disputes WHERE tenant_id=$1 AND id=$2 AND status IN ('open','in_review') FOR UPDATE`, scope.TenantID, id).Scan(&creator); err != nil {
			return marketplace.Dispute{}, "", err
		}
		if creator == scope.ActorID {
			return marketplace.Dispute{}, "", platform.ErrPermissionDenied
		}
		item, err := scanMarketplaceDispute(tx.QueryRow(ctx, `UPDATE marketplace_disputes SET status=$3,resolution=$4,resolved_by=$5,resolved_at=now() WHERE tenant_id=$1 AND id=$2 RETURNING id,tenant_id,listing_id,claimant_type,claimant_id,reason,status,resolution,created_by,COALESCE(resolved_by,''),created_at,resolved_at`, scope.TenantID, id, input.Decision, resolution, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"listing_id": item.ListingID, "decision": item.Status}
		if err = appendAudit(ctx, tx, scope, "marketplace_dispute.resolve", "marketplace_dispute", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "listing", item.ListingID, "marketplace_dispute."+item.Status, 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) RequestTakedown(ctx context.Context, scope tenancy.Scope, input application.TakedownInput, key string) (marketplace.Takedown, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	if input.ListingID == "" || input.Reason == "" {
		return marketplace.Takedown{}, platform.Invalid("invalid_takedown", "listing and reason are required")
	}
	return runCommand(ctx, s, scope, key, "takedown.request", func(tx pgx.Tx) (marketplace.Takedown, string, error) {
		var status string
		if err := tx.QueryRow(ctx, `SELECT status FROM listings WHERE tenant_id=$1 AND id=$2`, scope.TenantID, input.ListingID).Scan(&status); err != nil {
			return marketplace.Takedown{}, "", err
		}
		if status == "removed" {
			return marketplace.Takedown{}, "", platform.Invalid("listing_already_removed", "listing is already removed")
		}
		item, err := scanTakedown(tx.QueryRow(ctx, `INSERT INTO takedowns (tenant_id,listing_id,reason,requested_by) VALUES($1,$2,$3,$4) RETURNING id,tenant_id,listing_id,reason,status,requested_by,COALESCE(reviewed_by,''),created_at,reviewed_at`, scope.TenantID, input.ListingID, input.Reason, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"listing_id": item.ListingID, "reason": item.Reason}
		if err = appendAudit(ctx, tx, scope, "takedown.request", "takedown", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "listing", item.ListingID, "takedown.requested", 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

func (s *Store) ReviewTakedown(ctx context.Context, scope tenancy.Scope, id string, input application.ReviewInput, key string) (marketplace.Takedown, error) {
	input.Decision = strings.ToLower(strings.TrimSpace(input.Decision))
	if input.Decision != "approved" && input.Decision != "rejected" {
		return marketplace.Takedown{}, platform.Invalid("invalid_takedown_decision", "decision must be approved or rejected")
	}
	return runCommand(ctx, s, scope, key, "takedown.review", func(tx pgx.Tx) (marketplace.Takedown, string, error) {
		item, err := scanTakedown(tx.QueryRow(ctx, `SELECT id,tenant_id,listing_id,reason,status,requested_by,COALESCE(reviewed_by,''),created_at,reviewed_at FROM takedowns WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, scope.TenantID, id))
		if err != nil {
			return item, "", err
		}
		if item.Status != "requested" {
			return item, "", platform.Invalid("takedown_not_pending", "only requested takedowns can be reviewed")
		}
		if item.RequestedBy == scope.ActorID {
			return item, "", platform.ErrPermissionDenied
		}
		status := input.Decision
		if input.Decision == "approved" {
			status = "executed"
			if _, err = tx.Exec(ctx, `UPDATE listings SET status='removed',version=version+1,updated_at=now() WHERE tenant_id=$1 AND id=$2 AND status<>'removed'`, scope.TenantID, item.ListingID); err != nil {
				return item, "", err
			}
		}
		item, err = scanTakedown(tx.QueryRow(ctx, `UPDATE takedowns SET status=$3,reviewed_by=$4,reviewed_at=now() WHERE tenant_id=$1 AND id=$2 RETURNING id,tenant_id,listing_id,reason,status,requested_by,COALESCE(reviewed_by,''),created_at,reviewed_at`, scope.TenantID, id, status, scope.ActorID))
		if err != nil {
			return item, "", err
		}
		metadata := map[string]any{"listing_id": item.ListingID, "decision": input.Decision, "status": item.Status}
		if err = appendAudit(ctx, tx, scope, "takedown.review", "takedown", item.ID, key, metadata); err != nil {
			return item, "", err
		}
		if err = appendEvent(ctx, tx, scope, "listing", item.ListingID, "takedown."+item.Status, 1, metadata); err != nil {
			return item, "", err
		}
		return item, item.ID, nil
	})
}

var _ application.ChannelStore = (*Store)(nil)

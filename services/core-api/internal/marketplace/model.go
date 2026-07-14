package marketplace

import (
	"fmt"
	"time"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/state"
)

var listingTypes = map[string]bool{"adapter": true, "capability": true, "workflow": true, "agent": true, "mcp": true, "business_blueprint": true, "pricing_template": true, "growth_playbook": true}

func ValidListingType(value string) bool { return listingTypes[value] }

type Developer struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Publisher struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	DeveloperID string    `json:"developer_id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Listing struct {
	ID          string           `json:"id"`
	TenantID    string           `json:"tenant_id"`
	PublisherID string           `json:"publisher_id"`
	Name        string           `json:"name"`
	Type        string           `json:"listing_type"`
	Status      string           `json:"status"`
	Version     int              `json:"version"`
	CreatedBy   string           `json:"created_by"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	Versions    []ListingVersion `json:"versions"`
}

type ListingVersion struct {
	ID                 string         `json:"id"`
	TenantID           string         `json:"tenant_id"`
	ListingID          string         `json:"listing_id"`
	Version            int            `json:"version"`
	CapabilityManifest map[string]any `json:"capability_manifest"`
	PermissionManifest map[string]any `json:"permission_manifest"`
	ContentRef         string         `json:"content_ref"`
	Checksum           string         `json:"checksum"`
	CreatedBy          string         `json:"created_by"`
	CreatedAt          time.Time      `json:"created_at"`
}

type Review struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	ListingID        string    `json:"listing_id"`
	ListingVersionID string    `json:"listing_version_id"`
	Type             string    `json:"review_type"`
	Decision         string    `json:"decision"`
	Rationale        string    `json:"rationale"`
	ReviewedBy       string    `json:"reviewed_by"`
	CreatedAt        time.Time `json:"created_at"`
}
type SandboxRun struct {
	ID               string         `json:"id"`
	TenantID         string         `json:"tenant_id"`
	ListingVersionID string         `json:"listing_version_id"`
	Status           string         `json:"status"`
	CreatedBy        string         `json:"created_by"`
	Policy           map[string]any `json:"policy"`
	Result           map[string]any `json:"result"`
	StartedAt        time.Time      `json:"started_at"`
	CompletedAt      *time.Time     `json:"completed_at,omitempty"`
}
type QualityScore struct {
	ID               string         `json:"id"`
	TenantID         string         `json:"tenant_id"`
	ListingVersionID string         `json:"listing_version_id"`
	ScoreBPS         int            `json:"score_bps"`
	Dimensions       map[string]any `json:"dimensions"`
	CreatedBy        string         `json:"created_by"`
	CreatedAt        time.Time      `json:"created_at"`
}
type IncidentRecord struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	ListingID  string     `json:"listing_id"`
	Severity   string     `json:"severity"`
	Summary    string     `json:"summary"`
	Status     string     `json:"status"`
	CreatedBy  string     `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}
type RevenueShareRule struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	ListingID   string    `json:"listing_id"`
	PublisherID string    `json:"publisher_id"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	CreatedBy   string    `json:"created_by"`
	BasisPoints int       `json:"basis_points"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
}
type PayoutReserve struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	ListingID     string    `json:"listing_id"`
	PublisherID   string    `json:"publisher_id"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	ReferenceType string    `json:"reference_type"`
	ReferenceID   string    `json:"reference_id"`
	CreatedBy     string    `json:"created_by"`
	AmountMinor   int64     `json:"amount_minor"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
type Dispute struct {
	ID           string         `json:"id"`
	TenantID     string         `json:"tenant_id"`
	ListingID    string         `json:"listing_id"`
	ClaimantType string         `json:"claimant_type"`
	ClaimantID   string         `json:"claimant_id"`
	Reason       string         `json:"reason"`
	Status       string         `json:"status"`
	CreatedBy    string         `json:"created_by"`
	ResolvedBy   string         `json:"resolved_by,omitempty"`
	Resolution   map[string]any `json:"resolution"`
	CreatedAt    time.Time      `json:"created_at"`
	ResolvedAt   *time.Time     `json:"resolved_at,omitempty"`
}
type Takedown struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	ListingID   string     `json:"listing_id"`
	Reason      string     `json:"reason"`
	Status      string     `json:"status"`
	RequestedBy string     `json:"requested_by"`
	ReviewedBy  string     `json:"reviewed_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
}

type Overview struct {
	Developers        []Developer        `json:"developers"`
	Publishers        []Publisher        `json:"publishers"`
	Listings          []Listing          `json:"listings"`
	Reviews           []Review           `json:"reviews"`
	SandboxRuns       []SandboxRun       `json:"sandbox_runs"`
	QualityScores     []QualityScore     `json:"quality_scores"`
	Incidents         []IncidentRecord   `json:"incidents"`
	RevenueShareRules []RevenueShareRule `json:"revenue_share_rules"`
	PayoutReserves    []PayoutReserve    `json:"payout_reserves"`
	Disputes          []Dispute          `json:"disputes"`
	Takedowns         []Takedown         `json:"takedowns"`
}

func NewListing(tenantID, publisherID, name, listingType string) (Listing, error) {
	if !ValidListingType(listingType) {
		return Listing{}, fmt.Errorf("unsupported listing type")
	}
	now := time.Now().UTC()
	return Listing{ID: platform.NewID("listing"), TenantID: tenantID, PublisherID: publisherID, Name: name, Type: listingType, Status: "draft", Version: 1, CreatedAt: now, UpdatedAt: now, Versions: []ListingVersion{}}, nil
}

func (l *Listing) Transition(to string) error {
	if err := state.Listing.Transition(l.Status, to); err != nil {
		return err
	}
	l.Status, l.Version, l.UpdatedAt = to, l.Version+1, time.Now().UTC()
	return nil
}

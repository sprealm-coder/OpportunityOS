package marketplace

import (
	"fmt"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/state"
)

var listingTypes = map[string]bool{"adapter": true, "capability": true, "workflow": true, "agent": true, "mcp": true, "business_blueprint": true, "pricing_template": true, "growth_playbook": true}

type Listing struct {
	ID, TenantID, PublisherID, Name, Type, Status string
	Version                                       int
	CapabilityManifest, PermissionManifest        map[string]any
}

func NewListing(tenantID, publisherID, name, listingType string) (Listing, error) {
	if !listingTypes[listingType] {
		return Listing{}, fmt.Errorf("unsupported listing type")
	}
	return Listing{ID: platform.NewID("listing"), TenantID: tenantID, PublisherID: publisherID, Name: name, Type: listingType, Status: "draft", Version: 1, CapabilityManifest: map[string]any{}, PermissionManifest: map[string]any{}}, nil
}
func (l *Listing) Transition(to string) error {
	if err := state.Listing.Transition(l.Status, to); err != nil {
		return err
	}
	l.Status = to
	l.Version++
	return nil
}

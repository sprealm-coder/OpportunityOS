package capability

import "github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"

type Capability struct{ ID, TenantID, Name, Description string }
type Requirement struct {
	ID, TenantID, BlueprintID, CapabilityID string
	Required                                bool
}
type Provider struct{ ID, TenantID, Name, Status string }
type ProviderEndpoint struct{ ID, TenantID, ProviderID, CapabilityID, AdapterType, AdapterVersion, Status string }

func New(tenantID, name string) Capability {
	return Capability{ID: platform.NewID("cap"), TenantID: tenantID, Name: name}
}
func NewProvider(tenantID, name string) Provider {
	return Provider{ID: platform.NewID("prv"), TenantID: tenantID, Name: name, Status: "active"}
}
func NewEndpoint(tenantID, providerID, capabilityID, adapterType string) ProviderEndpoint {
	return ProviderEndpoint{ID: platform.NewID("ep"), TenantID: tenantID, ProviderID: providerID, CapabilityID: capabilityID, AdapterType: adapterType, AdapterVersion: "v1", Status: "healthy"}
}

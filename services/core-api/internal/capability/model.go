package capability

import "github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"

type Capability struct {
	ID          string         `json:"id"`
	TenantID    string         `json:"tenant_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Definition  map[string]any `json:"definition"`
}
type Requirement struct {
	ID, TenantID, BlueprintID, CapabilityID string
	Required                                bool
}
type Provider struct {
	ID        string             `json:"id"`
	TenantID  string             `json:"tenant_id"`
	Name      string             `json:"name"`
	Status    string             `json:"status"`
	Endpoints []ProviderEndpoint `json:"endpoints,omitempty"`
}
type ProviderEndpoint struct {
	ID             string `json:"id"`
	TenantID       string `json:"tenant_id"`
	ProviderID     string `json:"provider_id"`
	CapabilityID   string `json:"capability_id"`
	AdapterType    string `json:"adapter_type"`
	AdapterVersion string `json:"adapter_version"`
	Status         string `json:"status"`
}

func New(tenantID, name string) Capability {
	return Capability{ID: platform.NewID("cap"), TenantID: tenantID, Name: name}
}
func NewProvider(tenantID, name string) Provider {
	return Provider{ID: platform.NewID("prv"), TenantID: tenantID, Name: name, Status: "active"}
}
func NewEndpoint(tenantID, providerID, capabilityID, adapterType string) ProviderEndpoint {
	return ProviderEndpoint{ID: platform.NewID("ep"), TenantID: tenantID, ProviderID: providerID, CapabilityID: capabilityID, AdapterType: adapterType, AdapterVersion: "v1", Status: "healthy"}
}

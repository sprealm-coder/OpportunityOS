package featureflag

import "sync"

type Definition struct {
	Key, Description string
	DefaultEnabled   bool
}
type Gate struct {
	mu          sync.RWMutex
	definitions map[string]Definition
	tenant      map[string]bool
}

func New() *Gate                    { return &Gate{definitions: map[string]Definition{}, tenant: map[string]bool{}} }
func (g *Gate) Define(d Definition) { g.mu.Lock(); defer g.mu.Unlock(); g.definitions[d.Key] = d }
func (g *Gate) SetTenant(tenantID, key string, enabled bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.tenant[tenantID+"/"+key] = enabled
}
func (g *Gate) Enabled(tenantID, key string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if enabled, ok := g.tenant[tenantID+"/"+key]; ok {
		return enabled
	}
	return g.definitions[key].DefaultEnabled
}

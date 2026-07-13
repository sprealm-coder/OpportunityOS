package state

import (
	"fmt"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
)

type Machine struct {
	name        string
	transitions map[string]map[string]struct{}
}

func New(name string, transitions map[string][]string) Machine {
	compiled := make(map[string]map[string]struct{}, len(transitions))
	for from, destinations := range transitions {
		compiled[from] = make(map[string]struct{}, len(destinations))
		for _, to := range destinations {
			compiled[from][to] = struct{}{}
		}
	}
	return Machine{name: name, transitions: compiled}
}

func (m Machine) Can(from, to string) bool {
	_, ok := m.transitions[from][to]
	return ok
}

func (m Machine) Transition(from, to string) error {
	if !m.Can(from, to) {
		return &platform.Error{Code: "invalid_transition", Message: fmt.Sprintf("%s cannot transition from %s to %s", m.name, from, to)}
	}
	return nil
}

var Opportunity = New("opportunity", map[string][]string{
	"detected":     {"enriched", "rejected", "archived"},
	"enriched":     {"scored", "rejected", "archived"},
	"scored":       {"under_review", "rejected", "archived"},
	"under_review": {"approved", "rejected"},
	"approved":     {"incubating", "archived"},
	"incubating":   {"archived"},
	"rejected":     {"archived"},
})

var Incubation = New("incubation_project", map[string][]string{
	"draft":       {"researching", "cancelled"},
	"researching": {"validating", "paused", "cancelled"},
	"validating":  {"approved", "researching", "paused", "cancelled"},
	"approved":    {"building", "cancelled"},
	"building":    {"launched", "paused", "cancelled"},
	"launched":    {"completed", "paused"},
	"paused":      {"researching", "validating", "building", "launched", "cancelled"},
})

var Blueprint = New("business_blueprint", map[string][]string{
	"draft":       {"analyzing", "retired"},
	"analyzing":   {"validating", "paused"},
	"validating":  {"approved", "analyzing", "paused"},
	"approved":    {"configuring", "retired"},
	"configuring": {"ready", "paused"},
	"ready":       {"launched", "paused"},
	"launched":    {"paused", "retired"},
	"paused":      {"analyzing", "validating", "configuring", "ready", "launched", "retired"},
})

var Product = New("product", map[string][]string{
	"draft":      {"validating", "retired"},
	"validating": {"ready", "draft"},
	"ready":      {"published", "draft"},
	"published":  {"suspended", "retired"},
	"suspended":  {"published", "retired"},
})

var Order = New("order", map[string][]string{
	"created":          {"awaiting_payment", "cancelled"},
	"awaiting_payment": {"paid", "cancelled"},
	"paid":             {"provisioning", "refund_pending"},
	"provisioning":     {"active", "cancelled", "refund_pending"},
	"active":           {"completed", "cancelled", "refund_pending", "disputed"},
	"completed":        {"refund_pending", "disputed"},
	"refund_pending":   {"refunded", "disputed"},
})

var Execution = New("execution", map[string][]string{
	"created":     {"validating", "cancelled"},
	"validating":  {"reserved", "failed", "cancelled"},
	"reserved":    {"queued", "submitted", "cancelled"},
	"queued":      {"submitted", "cancelled"},
	"submitted":   {"processing", "failed", "cancelled"},
	"processing":  {"succeeded", "failed", "cancelled"},
	"succeeded":   {"reconciling"},
	"failed":      {"reconciling"},
	"cancelled":   {"reconciling"},
	"reconciling": {"settled"},
})

var Lead = New("lead", map[string][]string{
	"discovered":            {"enriched", "suppressed"},
	"enriched":              {"qualified", "suppressed"},
	"qualified":             {"proof_requested", "approved_for_outreach", "lost", "suppressed"},
	"proof_requested":       {"proof_ready", "lost", "suppressed"},
	"proof_ready":           {"approved_for_outreach", "lost", "suppressed"},
	"approved_for_outreach": {"contacted", "suppressed"},
	"contacted":             {"replied", "lost", "suppressed"},
	"replied":               {"meeting", "lost", "suppressed"},
	"meeting":               {"proposal", "lost", "suppressed"},
	"proposal":              {"won", "lost", "suppressed"},
})

var Listing = New("listing", map[string][]string{
	"draft":            {"submitted", "removed"},
	"submitted":        {"automated_review", "removed"},
	"automated_review": {"manual_review", "removed"},
	"manual_review":    {"sandbox_testing", "removed"},
	"sandbox_testing":  {"limited_release", "removed"},
	"limited_release":  {"published", "suspended", "removed"},
	"published":        {"suspended", "removed"},
	"suspended":        {"published", "removed"},
})

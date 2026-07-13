package routing

import (
	"fmt"
	"sort"
)

type Candidate struct {
	EndpointID         string
	Healthy            bool
	Priority           int
	EstimatedCostMinor int64
	Capacity           int
}
type Policy struct {
	ID, TenantID, Strategy string
	Version                int
}

func (p Policy) Select(candidates []Candidate) (Candidate, error) {
	eligible := []Candidate{}
	for _, c := range candidates {
		if c.Healthy && c.Capacity > 0 {
			eligible = append(eligible, c)
		}
	}
	if len(eligible) == 0 {
		return Candidate{}, fmt.Errorf("no healthy provider endpoint with capacity")
	}
	switch p.Strategy {
	case "lowest_cost":
		sort.SliceStable(eligible, func(i, j int) bool { return eligible[i].EstimatedCostMinor < eligible[j].EstimatedCostMinor })
	default:
		sort.SliceStable(eligible, func(i, j int) bool { return eligible[i].Priority < eligible[j].Priority })
	}
	return eligible[0], nil
}

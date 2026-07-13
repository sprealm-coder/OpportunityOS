package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
)

type NodeType string

const (
	Start        NodeType = "start"
	Validate     NodeType = "validate"
	Transform    NodeType = "transform"
	Condition    NodeType = "condition"
	Approval     NodeType = "approval"
	RealtimeCall NodeType = "realtime_call"
	AsyncSubmit  NodeType = "async_submit"
	AsyncWait    NodeType = "async_wait"
	Provision    NodeType = "provision"
	ManualTask   NodeType = "manual_task"
	WebhookWait  NodeType = "webhook_wait"
	Meter        NodeType = "meter"
	Notify       NodeType = "notify"
	Compensate   NodeType = "compensate"
	End          NodeType = "end"
)

var supported = map[NodeType]bool{Start: true, Validate: true, Transform: true, Condition: true, Approval: true, RealtimeCall: true, AsyncSubmit: true, AsyncWait: true, Provision: true, ManualTask: true, WebhookWait: true, Meter: true, Notify: true, Compensate: true, End: true}

type Node struct {
	ID         string
	Type       NodeType
	Config     map[string]any
	RetryLimit int
	Timeout    time.Duration
}
type Edge struct {
	From, To  string
	Condition string
}
type Definition struct {
	ID, TenantID, Name string
	Version            int
	Nodes              []Node
	Edges              []Edge
}
type StepRun struct {
	NodeID                 string
	Status                 string
	Attempt                int
	StartedAt, CompletedAt time.Time
	Error                  string
}
type Run struct {
	ID, TenantID, DefinitionID, IdempotencyKey, Status string
	Variables                                          map[string]any
	Steps                                              []StepRun
	CreatedAt, UpdatedAt                               time.Time
}

func (d Definition) Validate() error {
	if d.TenantID == "" {
		return platform.ErrTenantRequired
	}
	if d.Version < 1 {
		return fmt.Errorf("workflow version must be positive")
	}
	nodes := map[string]Node{}
	starts, ends := 0, 0
	for _, n := range d.Nodes {
		if !supported[n.Type] {
			return fmt.Errorf("unsupported node type %s", n.Type)
		}
		if _, exists := nodes[n.ID]; exists {
			return fmt.Errorf("duplicate node %s", n.ID)
		}
		nodes[n.ID] = n
		if n.Type == Start {
			starts++
		}
		if n.Type == End {
			ends++
		}
	}
	if starts != 1 || ends < 1 {
		return fmt.Errorf("workflow requires exactly one start and at least one end")
	}
	adj := map[string][]string{}
	indegree := map[string]int{}
	for id := range nodes {
		indegree[id] = 0
	}
	for _, edge := range d.Edges {
		if _, ok := nodes[edge.From]; !ok {
			return fmt.Errorf("unknown edge source %s", edge.From)
		}
		if _, ok := nodes[edge.To]; !ok {
			return fmt.Errorf("unknown edge destination %s", edge.To)
		}
		adj[edge.From] = append(adj[edge.From], edge.To)
		indegree[edge.To]++
	}
	queue := []string{}
	for id, n := range indegree {
		if n == 0 {
			queue = append(queue, id)
		}
	}
	visited := 0
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		visited++
		for _, to := range adj[id] {
			indegree[to]--
			if indegree[to] == 0 {
				queue = append(queue, to)
			}
		}
	}
	if visited != len(nodes) {
		return fmt.Errorf("workflow graph contains a cycle")
	}
	return nil
}

type Handler interface {
	Execute(context.Context, Node, map[string]any) (map[string]any, error)
}
type HandlerFunc func(context.Context, Node, map[string]any) (map[string]any, error)

func (f HandlerFunc) Execute(ctx context.Context, n Node, v map[string]any) (map[string]any, error) {
	return f(ctx, n, v)
}

type Engine struct {
	handlers    map[NodeType]Handler
	runs        map[string]*Run
	idempotency map[string]string
}

func NewEngine() *Engine {
	e := &Engine{handlers: map[NodeType]Handler{}, runs: map[string]*Run{}, idempotency: map[string]string{}}
	pass := HandlerFunc(func(_ context.Context, _ Node, v map[string]any) (map[string]any, error) { return v, nil })
	for t := range supported {
		e.handlers[t] = pass
	}
	return e
}
func (e *Engine) Register(nodeType NodeType, handler Handler) { e.handlers[nodeType] = handler }

func (e *Engine) Execute(ctx context.Context, d Definition, idempotencyKey string, input map[string]any) (*Run, error) {
	if idempotencyKey == "" {
		return nil, platform.ErrIdempotencyKeyRequired
	}
	if err := d.Validate(); err != nil {
		return nil, err
	}
	key := d.TenantID + "/" + d.ID + "/" + idempotencyKey
	if runID, ok := e.idempotency[key]; ok {
		return e.runs[runID], nil
	}
	now := time.Now().UTC()
	run := &Run{ID: platform.NewID("run"), TenantID: d.TenantID, DefinitionID: d.ID, IdempotencyKey: idempotencyKey, Status: "processing", Variables: cloneMap(input), CreatedAt: now, UpdatedAt: now}
	e.runs[run.ID] = run
	e.idempotency[key] = run.ID
	ordered, err := topological(d)
	if err != nil {
		return nil, err
	}
	for _, node := range ordered {
		step := StepRun{NodeID: node.ID, Status: "processing", Attempt: 1, StartedAt: time.Now().UTC()}
		handler := e.handlers[node.Type]
		vars, execErr := handler.Execute(ctx, node, cloneMap(run.Variables))
		step.CompletedAt = time.Now().UTC()
		if execErr != nil {
			step.Status = "failed"
			step.Error = execErr.Error()
			run.Steps = append(run.Steps, step)
			run.Status = "failed"
			run.UpdatedAt = step.CompletedAt
			return run, execErr
		}
		step.Status = "succeeded"
		run.Variables = vars
		run.Steps = append(run.Steps, step)
	}
	run.Status = "succeeded"
	run.UpdatedAt = time.Now().UTC()
	return run, nil
}

func cloneMap(source map[string]any) map[string]any {
	target := map[string]any{}
	for k, v := range source {
		target[k] = v
	}
	return target
}
func topological(d Definition) ([]Node, error) {
	nodes := map[string]Node{}
	indegree := map[string]int{}
	adj := map[string][]string{}
	for _, n := range d.Nodes {
		nodes[n.ID] = n
		indegree[n.ID] = 0
	}
	for _, edge := range d.Edges {
		adj[edge.From] = append(adj[edge.From], edge.To)
		indegree[edge.To]++
	}
	queue := []string{}
	for _, n := range d.Nodes {
		if indegree[n.ID] == 0 {
			queue = append(queue, n.ID)
		}
	}
	result := []Node{}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		result = append(result, nodes[id])
		for _, to := range adj[id] {
			indegree[to]--
			if indegree[to] == 0 {
				queue = append(queue, to)
			}
		}
	}
	if len(result) != len(nodes) {
		return nil, fmt.Errorf("workflow graph contains a cycle")
	}
	return result, nil
}

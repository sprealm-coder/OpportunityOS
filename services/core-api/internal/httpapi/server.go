package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/audit"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/opportunity"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/outbox"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

type Server struct {
	opportunities *opportunity.Service
	audit         *audit.Log
	limiter       *rateLimiter
}

func New() http.Handler {
	auditLog := &audit.Log{}
	events := &outbox.Memory{}
	service := opportunity.NewService(opportunity.NewMemoryRepository(), auditLog, events)
	server := &Server{opportunities: service, audit: auditLog, limiter: newRateLimiter(120, time.Minute)}
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(server.requestContext)
	r.Use(server.securityHeaders)
	r.Use(server.limiter.Middleware)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "core-api"})
	})
	r.Route("/v1", func(r chi.Router) {
		r.Use(tenancy.Middleware)
		r.Get("/opportunities", server.listOpportunities)
		r.Post("/opportunities", server.createOpportunity)
		r.Get("/opportunities/{id}", server.getOpportunity)
		r.Post("/opportunities/{id}/evidence", server.addEvidence)
		r.Post("/opportunities/{id}/score", server.scoreOpportunity)
		r.Post("/opportunities/{id}/transitions", server.transitionOpportunity)
		r.Get("/audit", server.listAudit)
	})
	return r
}

func (s *Server) requestContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = platform.NewID("req")
		}
		traceID := strings.TrimSpace(r.Header.Get("X-Trace-ID"))
		if traceID == "" {
			traceID = platform.NewID("trace")
		}
		r.Header.Set("X-Request-ID", requestID)
		r.Header.Set("X-Trace-ID", traceID)
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("X-Trace-ID", traceID)
		next.ServeHTTP(w, r)
	})
}
func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func scope(r *http.Request) (tenancy.Scope, error) { return tenancy.FromContext(r.Context()) }
func idempotency(r *http.Request) (string, error) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		return "", platform.ErrIdempotencyKeyRequired
	}
	return key, nil
}
func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, r, http.StatusBadRequest, platform.Invalid("invalid_json", err.Error()))
		return false
	}
	return true
}

func (s *Server) createOpportunity(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := s.opportunities.Create(tenantScope, input.Name, input.Description, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}
func (s *Server) listOpportunities(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": s.opportunities.List(tenantScope)})
}
func (s *Server) getOpportunity(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	item, err := s.opportunities.Get(tenantScope, chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, platform.Invalid("not_found", "opportunity not found"))
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func (s *Server) addEvidence(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	var input struct {
		Kind       string `json:"kind"`
		Summary    string `json:"summary"`
		Confidence int    `json:"confidence"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := s.opportunities.AddEvidence(tenantScope, chi.URLParam(r, "id"), opportunity.Evidence{Kind: input.Kind, Summary: input.Summary, Confidence: input.Confidence}, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func (s *Server) scoreOpportunity(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	var input struct {
		Score int `json:"score"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := s.opportunities.Score(tenantScope, chi.URLParam(r, "id"), input.Score, requestID)
	if err != nil {
		writeError(w, r, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func (s *Server) transitionOpportunity(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	requestID, err := idempotency(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	var input struct {
		To string `json:"to"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := s.opportunities.Transition(tenantScope, chi.URLParam(r, "id"), input.To, requestID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func (s *Server) listAudit(w http.ResponseWriter, r *http.Request) {
	tenantScope, err := scope(r)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": s.audit.ForTenant(tenantScope.TenantID)})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, r *http.Request, status int, err error) {
	code := "internal_error"
	message := "request failed"
	var domainErr *platform.Error
	if errors.As(err, &domainErr) {
		code = domainErr.Code
		message = domainErr.Message
	} else if status < 500 {
		message = err.Error()
	}
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": message, "request_id": r.Header.Get("X-Request-ID")}})
}

type rateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	clients map[string]*rateBucket
}
type rateBucket struct {
	count int
	reset time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, clients: map[string]*rateBucket{}}
}
func (l *rateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.RemoteAddr
		if index := strings.LastIndex(host, ":"); index > 0 {
			host = host[:index]
		}
		now := time.Now()
		l.mu.Lock()
		bucket := l.clients[host]
		if bucket == nil || now.After(bucket.reset) {
			bucket = &rateBucket{reset: now.Add(l.window)}
			l.clients[host] = bucket
		}
		bucket.count++
		remaining := l.limit - bucket.count
		allowed := bucket.count <= l.limit
		reset := bucket.reset
		l.mu.Unlock()
		if remaining < 0 {
			remaining = 0
		}
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(l.limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))
		if !allowed {
			writeError(w, r, http.StatusTooManyRequests, platform.Invalid("rate_limited", "request rate limit exceeded"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

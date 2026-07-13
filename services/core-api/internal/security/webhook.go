package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
	"time"
)

type ReplayGuard struct {
	mu        sync.Mutex
	seen      map[string]time.Time
	Tolerance time.Duration
}

func NewReplayGuard(tolerance time.Duration) *ReplayGuard {
	return &ReplayGuard{seen: map[string]time.Time{}, Tolerance: tolerance}
}
func Signature(secret []byte, timestamp int64, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(strconv.FormatInt(timestamp, 10) + "."))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
func (g *ReplayGuard) Verify(secret []byte, timestamp int64, body []byte, signature, eventID string, now time.Time) error {
	if eventID == "" {
		return fmt.Errorf("event ID is required")
	}
	eventTime := time.Unix(timestamp, 0)
	if now.Sub(eventTime) > g.Tolerance || eventTime.Sub(now) > g.Tolerance {
		return fmt.Errorf("webhook timestamp outside tolerance")
	}
	expected := Signature(secret, timestamp, body)
	provided, err := hex.DecodeString(signature)
	if err != nil || !hmac.Equal([]byte(expected), []byte(hex.EncodeToString(provided))) {
		return fmt.Errorf("invalid webhook signature")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	for id, expiry := range g.seen {
		if now.After(expiry) {
			delete(g.seen, id)
		}
	}
	if _, exists := g.seen[eventID]; exists {
		return fmt.Errorf("webhook replay detected")
	}
	g.seen[eventID] = now.Add(g.Tolerance)
	return nil
}

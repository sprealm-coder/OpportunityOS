package security

import (
	"net"
	"testing"
	"time"
)

func TestWebhookSignatureAndReplay(t *testing.T) {
	guard := NewReplayGuard(5 * time.Minute)
	now := time.Now().UTC()
	body := []byte(`{"event":"test"}`)
	sig := Signature([]byte("secret"), now.Unix(), body)
	if err := guard.Verify([]byte("secret"), now.Unix(), body, sig, "evt-1", now); err != nil {
		t.Fatal(err)
	}
	if err := guard.Verify([]byte("secret"), now.Unix(), body, sig, "evt-1", now); err == nil {
		t.Fatal("expected replay rejection")
	}
}
func TestSSRFProtection(t *testing.T) {
	private := func(string) ([]net.IP, error) { return []net.IP{net.ParseIP("169.254.169.254")}, nil }
	if err := ValidatePublicURL("http://metadata.example/latest", private); err == nil {
		t.Fatal("metadata address allowed")
	}
	public := func(string) ([]net.IP, error) { return []net.IP{net.ParseIP("93.184.216.34")}, nil }
	if err := ValidatePublicURL("https://example.test/path", public); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePublicURL("file:///etc/passwd", public); err == nil {
		t.Fatal("non-http scheme allowed")
	}
}

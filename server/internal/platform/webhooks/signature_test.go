package webhooks

import (
	"testing"
	"time"
)

func TestSignAndVerify(t *testing.T) {
	now := time.Unix(1720000000, 0).UTC()
	body := []byte(`{"event_type":"payment.captured"}`)
	header := Sign("secret", now, body)

	if !Verify("secret", header, body, 5*time.Minute, now.Add(time.Minute)) {
		t.Fatal("expected signature to verify")
	}
	if Verify("wrong", header, body, 5*time.Minute, now.Add(time.Minute)) {
		t.Fatal("expected wrong secret to fail")
	}
	if Verify("secret", header, []byte(`{}`), 5*time.Minute, now.Add(time.Minute)) {
		t.Fatal("expected tampered body to fail")
	}
	if Verify("secret", header, body, 5*time.Minute, now.Add(10*time.Minute)) {
		t.Fatal("expected stale signature to fail")
	}
}

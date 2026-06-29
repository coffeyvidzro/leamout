package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

const SignatureHeader = "Leamout-Signature"

func Sign(secret string, timestamp time.Time, body []byte) string {
	unix := timestamp.UTC().Unix()
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(strconv.FormatInt(unix, 10)))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(body)
	return fmt.Sprintf("t=%d,v1=%s", unix, hex.EncodeToString(mac.Sum(nil)))
}

func Verify(secret, header string, body []byte, tolerance time.Duration, now time.Time) bool {
	var unix int64
	var sig string
	if _, err := fmt.Sscanf(header, "t=%d,v1=%s", &unix, &sig); err != nil {
		return false
	}
	timestamp := time.Unix(unix, 0).UTC()
	if tolerance > 0 {
		delta := now.UTC().Sub(timestamp)
		if delta < -tolerance || delta > tolerance {
			return false
		}
	}
	expected := Sign(secret, timestamp, body)
	_, expectedSig, ok := splitSignature(expected)
	if !ok {
		return false
	}
	return hmac.Equal([]byte(expectedSig), []byte(sig))
}

func splitSignature(header string) (int64, string, bool) {
	var unix int64
	var sig string
	if _, err := fmt.Sscanf(header, "t=%d,v1=%s", &unix, &sig); err != nil {
		return 0, "", false
	}
	return unix, sig, true
}

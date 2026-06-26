package payment

import (
	"encoding/json"
	"strings"
)

func encodeAnyMap(value map[string]any) ([]byte, error) {
	if value == nil {
		value = map[string]any{}
	}
	return json.Marshal(value)
}

func stringMapToAny(src map[string]string) map[string]any {
	out := make(map[string]any, len(src))
	for key, value := range src {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = value
	}
	return out
}

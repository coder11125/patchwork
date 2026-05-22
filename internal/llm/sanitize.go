package llm

import "strings"

func sanitizeBody(body []byte, apiKey string) string {
	s := string(body)
	if apiKey != "" && len(s) > 0 {
		s = strings.ReplaceAll(s, apiKey, "***")
	}
	return s
}

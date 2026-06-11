package middleware

import (
	"net/http"
	"strings"
)

const MaxBodySize = 1 << 20 // 1MB default

// MaxChatBodySize allows larger payloads on chat endpoints to support
// base64-encoded image attachments for multimodal requests.
const MaxChatBodySize = 30 << 20 // 30MB

// MaxKnowledgeBodySize allows larger payloads on agent endpoints to support
// knowledge-document text uploads (extracted client-side from PDF/Word/TXT).
const MaxKnowledgeBodySize = 5 << 20 // 5MB

// bodyLimitForPath returns the max request body size for a given path.
func bodyLimitForPath(path string) int64 {
	if strings.HasPrefix(path, "/api/chat") {
		return MaxChatBodySize
	}
	if strings.HasPrefix(path, "/api/agents") {
		return MaxKnowledgeBodySize
	}
	return MaxBodySize
}

func BodySizeLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, bodyLimitForPath(r.URL.Path))
		}
		next.ServeHTTP(w, r)
	})
}

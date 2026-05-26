package middleware

import (
	"net/http"
	"strings"

	"gorm.io/gorm"
)

// JWTOrAPIKey routes to JWT middleware when an Authorization: Bearer header is present,
// and falls back to API-key middleware otherwise. Allows the transcribe and job-fetch
// endpoints to serve both dashboard users and developer API clients.
func JWTOrAPIKey(secret string, db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		jwtH := JWT(secret)(next)
		apiKeyH := APIKey(db)(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
				jwtH.ServeHTTP(w, r)
			} else {
				apiKeyH.ServeHTTP(w, r)
			}
		})
	}
}

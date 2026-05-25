package middleware

import (
	"net/http"
	"strings"

	"github.com/srujanpendurthi/transcriptflow/internal/auth"
	"github.com/srujanpendurthi/transcriptflow/pkg/utils"
)

// JWT validates a Bearer token and stamps the user identity onto the context.
// Mount this on any route group that requires a logged-in web session.
func JWT(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				utils.WriteError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}

			claims, err := auth.ParseToken(strings.TrimPrefix(header, "Bearer "), secret)
			if err != nil {
				utils.WriteError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.WithUser(r.Context(), claims.UserID, claims.Plan)))
		})
	}
}

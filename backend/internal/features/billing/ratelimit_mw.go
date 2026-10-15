package billing

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/srujanpendurthi/transcriptflow/internal/features/auth"
	"github.com/srujanpendurthi/transcriptflow/internal/platform/models"
	"github.com/srujanpendurthi/transcriptflow/internal/platform/ratelimit"
	"github.com/srujanpendurthi/transcriptflow/internal/platform/utils"
	"gorm.io/gorm"
)

// RateLimit throttles a caller to their plan's RateLimitPerMin. It buckets by
// API key when one is present (so each key gets its own budget) and otherwise by
// user. Mount it after the auth middleware on the developer route group.
func RateLimit(db *gorm.DB, limiter *ratelimit.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := auth.UserIDFrom(r.Context())
			if !ok {
				next.ServeHTTP(w, r) // unauthenticated: let the handler answer 401
				return
			}

			bucketKey := userID.String()
			if keyID, ok := auth.APIKeyIDFrom(r.Context()); ok {
				bucketKey = "key:" + keyID.String()
			}

			limit := rateLimitFor(r.Context(), db, userID)
			allowed, retry := limiter.Allow(r.Context(), bucketKey, limit, time.Minute)
			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(int(retry.Seconds())+1))
				utils.WriteError(w, http.StatusTooManyRequests, "rate limit exceeded — slow down and retry")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func rateLimitFor(ctx context.Context, db *gorm.DB, userID uuid.UUID) int {
	var user models.User
	if err := db.WithContext(ctx).Select("plan").First(&user, "id = ?", userID).Error; err != nil {
		return EntitlementFor(models.PlanFree).RateLimitPerMin
	}
	return EntitlementFor(user.Plan).RateLimitPerMin
}

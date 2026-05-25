package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/srujanpendurthi/transcriptflow/internal/auth"
	"github.com/srujanpendurthi/transcriptflow/internal/models"
	"github.com/srujanpendurthi/transcriptflow/pkg/utils"
	"gorm.io/gorm"
)

// APIKey validates the X-API-Key header by hashing it with SHA-256 and doing
// a constant-time-equivalent DB lookup against the stored hash.
// On success it stamps the key owner's identity onto the context and bumps last_used_at.
func APIKey(db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawKey := r.Header.Get("X-API-Key")
			if rawKey == "" {
				utils.WriteError(w, http.StatusUnauthorized, "missing X-API-Key header")
				return
			}

			sum := sha256.Sum256([]byte(rawKey))
			keyHash := hex.EncodeToString(sum[:])

			var apiKey models.APIKey
			if err := db.Where("key_hash = ?", keyHash).First(&apiKey).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					utils.WriteError(w, http.StatusUnauthorized, "invalid API key")
				} else {
					utils.WriteError(w, http.StatusInternalServerError, "authentication error")
				}
				return
			}

			now := time.Now()
			db.Model(&apiKey).Update("last_used_at", now) //nolint:errcheck — best-effort update

			next.ServeHTTP(w, r.WithContext(auth.WithAPIKey(r.Context(), apiKey.UserID, apiKey.ID)))
		})
	}
}

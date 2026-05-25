package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/srujanpendurthi/transcriptflow/internal/auth"
	"github.com/srujanpendurthi/transcriptflow/internal/models"
	"github.com/srujanpendurthi/transcriptflow/pkg/utils"
)

type APIKeyHandler struct {
	db *gorm.DB
}

func NewAPIKeyHandler(db *gorm.DB) *APIKeyHandler {
	return &APIKeyHandler{db: db}
}

// GET /api/v1/keys
func (h *APIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFrom(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	var keys []models.APIKey
	if err := h.db.Where("user_id = ?", userID).Order("created_at desc").Find(&keys).Error; err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "failed to fetch keys")
		return
	}

	utils.WriteJSON(w, http.StatusOK, keys)
}

// POST /api/v1/keys
func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFrom(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		utils.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "key generation failed")
		return
	}
	rawKey := fmt.Sprintf("tf_%s", hex.EncodeToString(raw))

	sum := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(sum[:])

	prefix := rawKey
	if len(prefix) > 12 {
		prefix = prefix[:12]
	}

	key := models.APIKey{
		UserID:  userID,
		KeyHash: keyHash,
		Prefix:  prefix,
		Name:    req.Name,
	}
	if err := h.db.Create(&key).Error; err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "failed to create key")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, map[string]any{
		"id":         key.ID,
		"name":       key.Name,
		"prefix":     key.Prefix,
		"created_at": key.CreatedAt,
		"raw_key":    rawKey,
	})
}

// DELETE /api/v1/keys/{keyID}
func (h *APIKeyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFrom(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	keyID, err := uuid.Parse(chi.URLParam(r, "keyID"))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid key ID")
		return
	}

	result := h.db.Where("id = ? AND user_id = ?", keyID, userID).Delete(&models.APIKey{})
	if result.Error != nil {
		utils.WriteError(w, http.StatusInternalServerError, "failed to delete key")
		return
	}
	if result.RowsAffected == 0 {
		utils.WriteError(w, http.StatusNotFound, "key not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

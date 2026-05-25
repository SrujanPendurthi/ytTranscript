package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/srujanpendurthi/transcriptflow/internal/models"
	"github.com/srujanpendurthi/transcriptflow/pkg/utils"
	"gorm.io/gorm"
)

const jwtTTL = 24 * time.Hour

type Handler struct {
	db     *gorm.DB
	secret string
}

func NewHandler(db *gorm.DB, secret string) *Handler {
	return &Handler{db: db, secret: secret}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

// Signup creates a new user account and returns a JWT.
func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		utils.WriteError(w, http.StatusBadRequest, "email and password required")
		return
	}
	if len(req.Password) < 8 {
		utils.WriteError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "failed to process password")
		return
	}

	user := models.User{
		Email:        req.Email,
		PasswordHash: hash,
		Plan:         models.PlanFree,
		IsActive:     true,
	}
	if err := h.db.Create(&user).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			utils.WriteError(w, http.StatusConflict, "email already registered")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	token, err := IssueToken(user.ID, string(user.Plan), h.secret, jwtTTL)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, authResponse{Token: token, User: user})
}

// Login validates credentials and returns a fresh JWT.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		// Return the same message whether the email is wrong or the password is wrong
		// to avoid leaking which accounts exist.
		utils.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if !user.IsActive {
		utils.WriteError(w, http.StatusForbidden, "account is inactive")
		return
	}

	if !CheckPassword(user.PasswordHash, req.Password) {
		utils.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := IssueToken(user.ID, string(user.Plan), h.secret, jwtTTL)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	utils.WriteJSON(w, http.StatusOK, authResponse{Token: token, User: user})
}

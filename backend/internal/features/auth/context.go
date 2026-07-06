package auth

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const (
	ctxKeyUserID   contextKey = "user_id"
	ctxKeyPlan     contextKey = "plan"
	ctxKeyAPIKeyID contextKey = "api_key_id"
)

// WithUser stamps the JWT-derived identity onto the context for web routes.
func WithUser(ctx context.Context, userID uuid.UUID, plan string) context.Context {
	ctx = context.WithValue(ctx, ctxKeyUserID, userID)
	return context.WithValue(ctx, ctxKeyPlan, plan)
}

// WithAPIKey stamps the API-key-derived identity onto the context for dev routes.
func WithAPIKey(ctx context.Context, userID, apiKeyID uuid.UUID) context.Context {
	ctx = context.WithValue(ctx, ctxKeyUserID, userID)
	return context.WithValue(ctx, ctxKeyAPIKeyID, apiKeyID)
}

func UserIDFrom(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(ctxKeyUserID).(uuid.UUID)
	return v, ok
}

func PlanFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyPlan).(string)
	return v, ok
}

func APIKeyIDFrom(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(ctxKeyAPIKeyID).(uuid.UUID)
	return v, ok
}

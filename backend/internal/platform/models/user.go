package models

// Plan represents the user's billing tier.
type Plan string

const (
	PlanFree       Plan = "free"
	PlanPro        Plan = "pro"
	PlanEnterprise Plan = "enterprise"
)

type User struct {
	Base
	Email            string `gorm:"uniqueIndex;not null"               json:"email"`
	PasswordHash     string `gorm:"not null"                           json:"-"`
	StripeCustomerID string `gorm:"index"                              json:"stripe_customer_id,omitempty"`
	Plan             Plan   `gorm:"type:varchar(20);not null;default:'free'" json:"plan"`
	IsActive         bool   `gorm:"not null;default:true"              json:"is_active"`
}

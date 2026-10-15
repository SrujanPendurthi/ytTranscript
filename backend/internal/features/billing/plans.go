package billing

import "github.com/srujanpendurthi/transcriptflow/internal/platform/models"

// Entitlement is the per-plan set of limits that gate the metered API.
// Numbers are launch placeholders — confirm before enabling metered billing.
type Entitlement struct {
	IncludedMinutes int  // media-minutes included each billing period
	AllowOverage    bool // may exceed IncludedMinutes (billed as metered overage)
	RateLimitPerMin int  // API requests allowed per minute
	MaxVideoMinutes int  // reject a single video longer than this (0 = unlimited)
}

var entitlements = map[models.Plan]Entitlement{
	models.PlanFree:       {IncludedMinutes: 30, AllowOverage: false, RateLimitPerMin: 10, MaxVideoMinutes: 60},
	models.PlanPro:        {IncludedMinutes: 500, AllowOverage: true, RateLimitPerMin: 60, MaxVideoMinutes: 240},
	models.PlanEnterprise: {IncludedMinutes: 5000, AllowOverage: true, RateLimitPerMin: 300, MaxVideoMinutes: 0},
}

// EntitlementFor returns the limits for a plan, defaulting to Free for unknown plans.
func EntitlementFor(plan models.Plan) Entitlement {
	if e, ok := entitlements[plan]; ok {
		return e
	}
	return entitlements[models.PlanFree]
}

// MinutesFromSeconds converts a media duration to whole billable minutes,
// rounding up so any partial minute counts as one.
func MinutesFromSeconds(seconds float64) int {
	if seconds <= 0 {
		return 0
	}
	m := int(seconds) / 60
	if int(seconds)%60 != 0 || seconds > float64(m*60) {
		m++
	}
	return m
}

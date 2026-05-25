package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/srujanpendurthi/transcriptflow/internal/auth"
	"github.com/srujanpendurthi/transcriptflow/internal/models"
	"github.com/srujanpendurthi/transcriptflow/pkg/utils"
	stripe "github.com/stripe/stripe-go/v76"
	stripecustomer "github.com/stripe/stripe-go/v76/customer"
	stripeSession "github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/webhook"
	"gorm.io/gorm"
)

const maxWebhookBytes = 65_536 // 64 KB — Stripe payloads are never larger

// BillingHandler owns the two Stripe-facing endpoints.
// stripe.Key must be set globally (in main) before any method is called.
type BillingHandler struct {
	db            *gorm.DB
	priceID       string // Stripe Price ID for the $10/month Pro plan
	webhookSecret string // whsec_… from Stripe dashboard
	appURL        string // base URL for redirect (e.g. https://app.transcriptflow.io)
}

func NewBillingHandler(db *gorm.DB, priceID, webhookSecret, appURL string) *BillingHandler {
	return &BillingHandler{
		db:            db,
		priceID:       priceID,
		webhookSecret: webhookSecret,
		appURL:        appURL,
	}
}

// ---------------------------------------------------------------------------
// POST /api/billing/checkout  (JWT-protected)
// ---------------------------------------------------------------------------

// Checkout creates a Stripe Checkout Session for the Pro subscription.
// If the user has no Stripe customer yet one is created and persisted.
// Returns {"checkout_url":"https://checkout.stripe.com/…"}.
func (h *BillingHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFrom(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "user lookup failed")
		return
	}

	if user.Plan == models.PlanPro {
		utils.WriteError(w, http.StatusConflict, "already subscribed to Pro")
		return
	}

	// Lazily create the Stripe customer and persist the ID so future checkouts
	// and webhooks can reference the same customer.
	customerID, err := h.ensureStripeCustomer(&user)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "stripe customer error: "+err.Error())
		return
	}

	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		Mode:     stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(h.priceID),
				Quantity: stripe.Int64(1),
			},
		},
		// Stripe replaces {CHECKOUT_SESSION_ID} in the URL after payment.
		SuccessURL: stripe.String(h.appURL + "/dashboard?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(h.appURL + "/billing"),
		// Attach user_id so the Stripe dashboard shows a human-readable link.
		Params: stripe.Params{
			Metadata: map[string]string{
				"user_id": user.ID.String(),
			},
		},
	}

	sess, err := stripeSession.New(params)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "checkout session failed: "+err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{"checkout_url": sess.URL})
}

// ensureStripeCustomer creates a Stripe Customer if the user has none yet,
// persists the ID, and returns it.
func (h *BillingHandler) ensureStripeCustomer(user *models.User) (string, error) {
	if user.StripeCustomerID != "" {
		return user.StripeCustomerID, nil
	}

	params := &stripe.CustomerParams{
		Email: stripe.String(user.Email),
		Params: stripe.Params{
			Metadata: map[string]string{"user_id": user.ID.String()},
		},
	}
	cust, err := stripecustomer.New(params)
	if err != nil {
		return "", err
	}

	if err := h.db.Model(user).Update("stripe_customer_id", cust.ID).Error; err != nil {
		return "", err
	}
	user.StripeCustomerID = cust.ID
	return cust.ID, nil
}

// ---------------------------------------------------------------------------
// POST /api/webhook/stripe  (public — Stripe calls this)
// ---------------------------------------------------------------------------

// Webhook validates the Stripe-Signature header and handles:
//   - invoice.paid                    → upgrade user plan to Pro
//   - customer.subscription.deleted   → downgrade user plan to Free
//
// All other event types return 200 immediately (Stripe best-practice: do not
// return errors for unhandled events or Stripe will retry indefinitely).
func (h *BillingHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	// Read the raw body BEFORE any parsing — the HMAC is computed over the
	// exact bytes Stripe sent. A parsed/re-serialised body will not match.
	r.Body = http.MaxBytesReader(w, r.Body, maxWebhookBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "cannot read body")
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(payload, sigHeader, h.webhookSecret)
	if err != nil {
		// Invalid signature — could be a replay or a spoofed request.
		utils.WriteError(w, http.StatusBadRequest, "invalid stripe signature")
		return
	}

	switch event.Type {
	case "invoice.paid":
		h.handleInvoicePaid(w, event)
	case "customer.subscription.deleted":
		h.handleSubscriptionDeleted(w, event)
	default:
		// Acknowledge events we don't handle so Stripe stops retrying.
		w.WriteHeader(http.StatusOK)
	}
}

// handleInvoicePaid upgrades the user to the Pro plan.
// Triggered after every successful subscription payment (initial + renewals).
func (h *BillingHandler) handleInvoicePaid(w http.ResponseWriter, event stripe.Event) {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "malformed invoice payload")
		return
	}

	if invoice.Customer == nil {
		utils.WriteError(w, http.StatusBadRequest, "invoice has no customer")
		return
	}

	if err := h.setPlan(invoice.Customer.ID, models.PlanPro); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "plan upgrade failed")
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleSubscriptionDeleted downgrades the user to the Free plan.
// Triggered when a subscription is cancelled or payment fails after retries.
func (h *BillingHandler) handleSubscriptionDeleted(w http.ResponseWriter, event stripe.Event) {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "malformed subscription payload")
		return
	}

	if sub.Customer == nil {
		utils.WriteError(w, http.StatusBadRequest, "subscription has no customer")
		return
	}

	if err := h.setPlan(sub.Customer.ID, models.PlanFree); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "plan downgrade failed")
		return
	}
	w.WriteHeader(http.StatusOK)
}

// setPlan finds the user by Stripe customer ID and updates their plan.
// Uses errors.Is(gorm.ErrRecordNotFound) to treat unknown customers as no-ops —
// Stripe may fire events for test-mode customers that never existed in our DB.
func (h *BillingHandler) setPlan(stripeCustomerID string, plan models.Plan) error {
	result := h.db.Model(&models.User{}).
		Where("stripe_customer_id = ?", stripeCustomerID).
		Update("plan", plan)

	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}
	return nil
}

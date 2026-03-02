package entity

import (
	"errors"
	"time"

	"github.com/bivex/google-billing-mock/internal/domain/event"
)

// Value types
type PurchaseToken string
type ProductID string
type PackageName string
type SubscriptionID string

// PurchaseState represents the purchase lifecycle.
type PurchaseState int

const (
	PurchaseStatePurchased PurchaseState = 0
	PurchaseStateCanceled  PurchaseState = 1
	PurchaseStatePending   PurchaseState = 2
)

// PaymentState represents payment status.
type PaymentState int

const (
	PaymentStatePending  PaymentState = 0
	PaymentStateReceived PaymentState = 1
	PaymentStateFree     PaymentState = 2
)

// AcknowledgementState represents ack lifecycle.
type AcknowledgementState int

const (
	AcknowledgementStatePending      AcknowledgementState = 0
	AcknowledgementStateAcknowledged AcknowledgementState = 1
)

// CancelReason represents the reason for cancellation.
type CancelReason int

const (
	CancelReasonUserCanceled   CancelReason = 0
	CancelReasonSystemCanceled CancelReason = 1
	CancelReasonReplacedByNew  CancelReason = 2
	CancelReasonDeveloper      CancelReason = 3
)

// SubscriptionPurchase is the aggregate root for subscription state.
type SubscriptionPurchase struct {
	Token                      PurchaseToken
	SubscriptionID             SubscriptionID
	PackageName                PackageName
	Kind                       string
	PurchaseState              PurchaseState
	PaymentState               *PaymentState
	AcknowledgementState       AcknowledgementState
	ExpiryTimeMillis           int64
	PurchaseTimeMillis         int64
	AutoRenewing               bool
	CancelReason               *CancelReason
	UserCancellationTimeMillis *int64
	OrderID                    string
	RegionCode                 string

	events []event.DomainEvent
}

// NewSubscriptionPurchase creates a new active subscription purchase.
func NewSubscriptionPurchase(token PurchaseToken, subID SubscriptionID, pkg PackageName) *SubscriptionPurchase {
	ps := PaymentStateReceived
	return &SubscriptionPurchase{
		Token:                token,
		SubscriptionID:       subID,
		PackageName:          pkg,
		Kind:                 "androidpublisher#subscriptionPurchase",
		PurchaseState:        PurchaseStatePurchased,
		PaymentState:         &ps,
		AcknowledgementState: AcknowledgementStatePending,
		ExpiryTimeMillis:     time.Now().Add(30 * 24 * time.Hour).UnixMilli(),
		PurchaseTimeMillis:   time.Now().UnixMilli(),
		AutoRenewing:         true,
		RegionCode:           "US",
		OrderID:              "GPA.0000-0000-0000-00000",
	}
}

// Acknowledge transitions acknowledgement state from pending → acknowledged.
// Returns error if already acknowledged (invariant).
func (s *SubscriptionPurchase) Acknowledge() error {
	if s.AcknowledgementState == AcknowledgementStateAcknowledged {
		return errors.New("purchase already acknowledged")
	}
	s.AcknowledgementState = AcknowledgementStateAcknowledged
	s.events = append(s.events, event.PurchaseAcknowledged{
		Token:     string(s.Token),
		Timestamp: time.Now(),
	})
	return nil
}

// Cancel sets the subscription to canceled state.
func (s *SubscriptionPurchase) Cancel(reason CancelReason) {
	s.PurchaseState = PurchaseStateCanceled
	s.AutoRenewing = false
	s.CancelReason = &reason
	now := time.Now().UnixMilli()
	s.UserCancellationTimeMillis = &now
	s.events = append(s.events, event.SubscriptionCanceled{
		Token:     string(s.Token),
		Reason:    int(reason),
		Timestamp: time.Now(),
	})
}

// Revoke immediately revokes access (developer-initiated cancel).
func (s *SubscriptionPurchase) Revoke() {
	reason := CancelReasonDeveloper
	s.PurchaseState = PurchaseStateCanceled
	s.AutoRenewing = false
	s.CancelReason = &reason
	s.ExpiryTimeMillis = time.Now().UnixMilli()
	s.events = append(s.events, event.SubscriptionRevoked{
		Token:     string(s.Token),
		Timestamp: time.Now(),
	})
}

// Defer extends the subscription expiry. Returns error if newExpiry is in the past.
func (s *SubscriptionPurchase) Defer(newExpiryMillis int64) error {
	if newExpiryMillis <= time.Now().UnixMilli() {
		return errors.New("desired expiry time must be in the future")
	}
	old := s.ExpiryTimeMillis
	s.ExpiryTimeMillis = newExpiryMillis
	s.events = append(s.events, event.SubscriptionDeferred{
		Token:           string(s.Token),
		OldExpiryMillis: old,
		NewExpiryMillis: newExpiryMillis,
		Timestamp:       time.Now(),
	})
	return nil
}

// IsExpired returns true if the subscription expiry is in the past.
func (s *SubscriptionPurchase) IsExpired() bool {
	return s.ExpiryTimeMillis < time.Now().UnixMilli()
}

// DomainEvents returns pending domain events.
func (s *SubscriptionPurchase) DomainEvents() []event.DomainEvent { return s.events }

// ClearEvents clears published events.
func (s *SubscriptionPurchase) ClearEvents() { s.events = nil }

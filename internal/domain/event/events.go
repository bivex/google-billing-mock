package event

import "time"

// DomainEvent is the base interface for all domain events.
type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
}

// PurchaseAcknowledged fires when a purchase is acknowledged.
type PurchaseAcknowledged struct {
	Token     string
	Timestamp time.Time
}

func (e PurchaseAcknowledged) EventName() string    { return "PurchaseAcknowledged" }
func (e PurchaseAcknowledged) OccurredAt() time.Time { return e.Timestamp }

// SubscriptionCanceled fires when a subscription is canceled.
type SubscriptionCanceled struct {
	Token     string
	Reason    int
	Timestamp time.Time
}

func (e SubscriptionCanceled) EventName() string    { return "SubscriptionCanceled" }
func (e SubscriptionCanceled) OccurredAt() time.Time { return e.Timestamp }

// SubscriptionRevoked fires on immediate revocation.
type SubscriptionRevoked struct {
	Token     string
	Timestamp time.Time
}

func (e SubscriptionRevoked) EventName() string    { return "SubscriptionRevoked" }
func (e SubscriptionRevoked) OccurredAt() time.Time { return e.Timestamp }

// SubscriptionDeferred fires when expiry is extended.
type SubscriptionDeferred struct {
	Token           string
	OldExpiryMillis int64
	NewExpiryMillis int64
	Timestamp       time.Time
}

func (e SubscriptionDeferred) EventName() string    { return "SubscriptionDeferred" }
func (e SubscriptionDeferred) OccurredAt() time.Time { return e.Timestamp }

// ProductRefunded fires when a product purchase is refunded.
type ProductRefunded struct {
	Token     string
	Timestamp time.Time
}

func (e ProductRefunded) EventName() string    { return "ProductRefunded" }
func (e ProductRefunded) OccurredAt() time.Time { return e.Timestamp }

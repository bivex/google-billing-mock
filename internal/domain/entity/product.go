package entity

import (
	"errors"
	"time"

	"github.com/bivex/google-billing-mock/internal/domain/event"
)

// ProductPurchase is the aggregate root for one-time in-app product purchases.
type ProductPurchase struct {
	Token                PurchaseToken
	ProductID            ProductID
	PackageName          PackageName
	Kind                 string
	PurchaseState        PurchaseState
	PaymentState         *PaymentState
	AcknowledgementState AcknowledgementState
	PurchaseTimeMillis   int64
	OrderID              string
	RegionCode           string
	Quantity             int
	ConsumptionState     int // 0=yet to consume, 1=consumed
	DeveloperPayload     string

	events []event.DomainEvent
}

// NewProductPurchase creates a new product purchase.
func NewProductPurchase(token PurchaseToken, productID ProductID, pkg PackageName) *ProductPurchase {
	ps := PaymentStateReceived
	return &ProductPurchase{
		Token:                token,
		ProductID:            productID,
		PackageName:          pkg,
		Kind:                 "androidpublisher#productPurchase",
		PurchaseState:        PurchaseStatePurchased,
		PaymentState:         &ps,
		AcknowledgementState: AcknowledgementStatePending,
		PurchaseTimeMillis:   time.Now().UnixMilli(),
		RegionCode:           "US",
		OrderID:              "GPA.0000-0000-0000-00000",
		Quantity:             1,
	}
}

// Acknowledge transitions state from pending → acknowledged.
func (p *ProductPurchase) Acknowledge() error {
	if p.AcknowledgementState == AcknowledgementStateAcknowledged {
		return errors.New("purchase already acknowledged")
	}
	p.AcknowledgementState = AcknowledgementStateAcknowledged
	p.events = append(p.events, event.PurchaseAcknowledged{
		Token:     string(p.Token),
		Timestamp: time.Now(),
	})
	return nil
}

// Consume marks the product as consumed (consumptionState → 1).
func (p *ProductPurchase) Consume() error {
	if p.ConsumptionState == 1 {
		return errors.New("purchase already consumed")
	}
	p.ConsumptionState = 1
	return nil
}

// Refund marks the product purchase as refunded.
func (p *ProductPurchase) Refund() {
	ps := PaymentStatePending
	p.PurchaseState = PurchaseStateCanceled
	p.PaymentState = &ps
	p.events = append(p.events, event.ProductRefunded{
		Token:     string(p.Token),
		Timestamp: time.Now(),
	})
}

// DomainEvents returns pending domain events.
func (p *ProductPurchase) DomainEvents() []event.DomainEvent { return p.events }

// ClearEvents clears published events.
func (p *ProductPurchase) ClearEvents() { p.events = nil }

package usecase

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/bivex/google-billing-mock/internal/application/dto"
	"github.com/bivex/google-billing-mock/internal/domain/entity"
	"github.com/bivex/google-billing-mock/internal/domain/repository"
	"go.uber.org/zap"
)

// GetSubscription retrieves subscription purchase state.
type GetSubscription struct {
	repo   repository.PurchaseRepository
	logger *zap.Logger
}

func NewGetSubscription(repo repository.PurchaseRepository, logger *zap.Logger) *GetSubscription {
	return &GetSubscription{repo: repo, logger: logger}
}

func (uc *GetSubscription) Execute(ctx context.Context, pkg, subID, token string) (*dto.SubscriptionPurchaseResponse, error) {
	purchase, err := uc.repo.GetSubscription(ctx,
		entity.PackageName(pkg),
		entity.SubscriptionID(subID),
		entity.PurchaseToken(token),
	)
	if err != nil {
		return nil, err
	}
	return toSubscriptionResponse(purchase, subID), nil
}

// GetProduct retrieves product purchase state.
type GetProduct struct {
	repo   repository.PurchaseRepository
	logger *zap.Logger
}

func NewGetProduct(repo repository.PurchaseRepository, logger *zap.Logger) *GetProduct {
	return &GetProduct{repo: repo, logger: logger}
}

func (uc *GetProduct) Execute(ctx context.Context, pkg, productID, token string) (*dto.ProductPurchaseResponse, error) {
	purchase, err := uc.repo.GetProduct(ctx,
		entity.PackageName(pkg),
		entity.ProductID(productID),
		entity.PurchaseToken(token),
	)
	if err != nil {
		return nil, err
	}
	return toProductResponse(purchase), nil
}

// Acknowledge confirms a subscription or product purchase.
type Acknowledge struct {
	repo   repository.PurchaseRepository
	logger *zap.Logger
}

func NewAcknowledge(repo repository.PurchaseRepository, logger *zap.Logger) *Acknowledge {
	return &Acknowledge{repo: repo, logger: logger}
}

func (uc *Acknowledge) ExecuteSubscription(ctx context.Context, pkg, subID, token string) error {
	purchase, err := uc.repo.GetSubscription(ctx,
		entity.PackageName(pkg),
		entity.SubscriptionID(subID),
		entity.PurchaseToken(token),
	)
	if err != nil {
		return err
	}
	if err := purchase.Acknowledge(); err != nil {
		return err
	}
	if err := uc.repo.UpdateSubscription(ctx, purchase); err != nil {
		return err
	}
	for _, ev := range purchase.DomainEvents() {
		uc.logger.Info("domain event", zap.String("event", ev.EventName()), zap.String("token", token))
	}
	purchase.ClearEvents()
	return nil
}

func (uc *Acknowledge) ExecuteProduct(ctx context.Context, pkg, productID, token string) error {
	purchase, err := uc.repo.GetProduct(ctx,
		entity.PackageName(pkg),
		entity.ProductID(productID),
		entity.PurchaseToken(token),
	)
	if err != nil {
		return err
	}
	if err := purchase.Acknowledge(); err != nil {
		return err
	}
	if err := uc.repo.UpdateProduct(ctx, purchase); err != nil {
		return err
	}
	for _, ev := range purchase.DomainEvents() {
		uc.logger.Info("domain event", zap.String("event", ev.EventName()), zap.String("token", token))
	}
	purchase.ClearEvents()
	return nil
}

// Cancel cancels a subscription.
type Cancel struct {
	repo   repository.PurchaseRepository
	logger *zap.Logger
}

func NewCancel(repo repository.PurchaseRepository, logger *zap.Logger) *Cancel {
	return &Cancel{repo: repo, logger: logger}
}

func (uc *Cancel) Execute(ctx context.Context, pkg, subID, token string) error {
	purchase, err := uc.repo.GetSubscription(ctx,
		entity.PackageName(pkg),
		entity.SubscriptionID(subID),
		entity.PurchaseToken(token),
	)
	if err != nil {
		return err
	}
	purchase.Cancel(entity.CancelReasonUserCanceled)
	if err := uc.repo.UpdateSubscription(ctx, purchase); err != nil {
		return err
	}
	for _, ev := range purchase.DomainEvents() {
		uc.logger.Info("domain event", zap.String("event", ev.EventName()), zap.String("token", token))
	}
	purchase.ClearEvents()
	return nil
}

// Refund processes a refund for a subscription.
type Refund struct {
	repo   repository.PurchaseRepository
	logger *zap.Logger
}

func NewRefund(repo repository.PurchaseRepository, logger *zap.Logger) *Refund {
	return &Refund{repo: repo, logger: logger}
}

func (uc *Refund) Execute(ctx context.Context, pkg, subID, token string) error {
	purchase, err := uc.repo.GetSubscription(ctx,
		entity.PackageName(pkg),
		entity.SubscriptionID(subID),
		entity.PurchaseToken(token),
	)
	if err != nil {
		return err
	}
	ps := entity.PaymentStatePending
	purchase.PaymentState = &ps
	purchase.PurchaseState = entity.PurchaseStateCanceled
	return uc.repo.UpdateSubscription(ctx, purchase)
}

// Revoke immediately revokes subscription access.
type Revoke struct {
	repo   repository.PurchaseRepository
	logger *zap.Logger
}

func NewRevoke(repo repository.PurchaseRepository, logger *zap.Logger) *Revoke {
	return &Revoke{repo: repo, logger: logger}
}

func (uc *Revoke) Execute(ctx context.Context, pkg, subID, token string) error {
	purchase, err := uc.repo.GetSubscription(ctx,
		entity.PackageName(pkg),
		entity.SubscriptionID(subID),
		entity.PurchaseToken(token),
	)
	if err != nil {
		return err
	}
	purchase.Revoke()
	if err := uc.repo.UpdateSubscription(ctx, purchase); err != nil {
		return err
	}
	for _, ev := range purchase.DomainEvents() {
		uc.logger.Info("domain event", zap.String("event", ev.EventName()), zap.String("token", token))
	}
	purchase.ClearEvents()
	return nil
}

// DeferSubscription extends subscription expiry.
type DeferSubscription struct {
	repo   repository.PurchaseRepository
	logger *zap.Logger
}

func NewDeferSubscription(repo repository.PurchaseRepository, logger *zap.Logger) *DeferSubscription {
	return &DeferSubscription{repo: repo, logger: logger}
}

func (uc *DeferSubscription) Execute(ctx context.Context, pkg, subID, token string, req dto.DeferSubscriptionRequest) (*dto.DeferSubscriptionResponse, error) {
	newExpiryMillis, err := strconv.ParseInt(req.DeferralInfo.DesiredExpiryTimeMillis, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid desiredExpiryTimeMillis: %w", err)
	}
	purchase, err := uc.repo.GetSubscription(ctx,
		entity.PackageName(pkg),
		entity.SubscriptionID(subID),
		entity.PurchaseToken(token),
	)
	if err != nil {
		return nil, err
	}
	if err := purchase.Defer(newExpiryMillis); err != nil {
		return nil, err
	}
	if err := uc.repo.UpdateSubscription(ctx, purchase); err != nil {
		return nil, err
	}
	for _, ev := range purchase.DomainEvents() {
		uc.logger.Info("domain event", zap.String("event", ev.EventName()), zap.String("token", token))
	}
	purchase.ClearEvents()
	return &dto.DeferSubscriptionResponse{
		NewExpiryTimeMillis: strconv.FormatInt(newExpiryMillis, 10),
	}, nil
}

// --- helpers ---

// toSubscriptionV2Response maps a SubscriptionPurchase to the v2 wire format.
func toSubscriptionV2Response(p *entity.SubscriptionPurchase, subID string) *dto.SubscriptionPurchaseV2Response {
	startTime := time.UnixMilli(p.PurchaseTimeMillis).UTC().Format(time.RFC3339)
	expiryTime := time.UnixMilli(p.ExpiryTimeMillis).UTC().Format(time.RFC3339)

	// Derive subscriptionState from v1 fields
	state := "SUBSCRIPTION_STATE_ACTIVE"
	switch {
	case p.ExpiryTimeMillis < time.Now().UnixMilli():
		state = "SUBSCRIPTION_STATE_EXPIRED"
	case p.PurchaseState == entity.PurchaseStateCanceled:
		state = "SUBSCRIPTION_STATE_CANCELED"
	case p.PaymentState != nil && *p.PaymentState == entity.PaymentStatePending:
		state = "SUBSCRIPTION_STATE_IN_GRACE_PERIOD"
	case p.PurchaseState == entity.PurchaseStatePurchased && !p.AutoRenewing:
		state = "SUBSCRIPTION_STATE_CANCELED"
	}

	ackState := "ACKNOWLEDGEMENT_STATE_PENDING"
	if p.AcknowledgementState == entity.AcknowledgementStateAcknowledged {
		ackState = "ACKNOWLEDGEMENT_STATE_ACKNOWLEDGED"
	}

	autoRenew := &dto.AutoRenewingPlan{AutoRenewEnabled: p.AutoRenewing}

	return &dto.SubscriptionPurchaseV2Response{
		Kind:                 "androidpublisher#subscriptionPurchaseV2",
		StartTime:            startTime,
		RegionCode:           p.RegionCode,
		SubscriptionState:    state,
		LatestOrderId:        p.OrderID,
		AcknowledgementState: ackState,
		LineItems: []dto.SubscriptionPurchaseV2LineItem{
			{
				ProductId:        subID,
				ExpiryTime:       expiryTime,
				AutoRenewingPlan: autoRenew,
			},
		},
	}
}

// toProductV2Response maps a ProductPurchase to the v2 wire format.
func toProductV2Response(p *entity.ProductPurchase) *dto.ProductPurchaseV2Response {
	completionTime := time.UnixMilli(p.PurchaseTimeMillis).UTC().Format(time.RFC3339)

	purchaseState := "PURCHASE_STATE_PURCHASED"
	switch p.PurchaseState {
	case entity.PurchaseStateCanceled:
		purchaseState = "PURCHASE_STATE_CANCELED"
	case entity.PurchaseStatePending:
		purchaseState = "PURCHASE_STATE_PENDING"
	}

	ackState := "ACKNOWLEDGEMENT_STATE_PENDING"
	if p.AcknowledgementState == entity.AcknowledgementStateAcknowledged {
		ackState = "ACKNOWLEDGEMENT_STATE_ACKNOWLEDGED"
	}

	qty := p.Quantity
	if qty == 0 {
		qty = 1
	}

	return &dto.ProductPurchaseV2Response{
		Kind:                   "androidpublisher#productPurchaseV2",
		OrderId:                p.OrderID,
		RegionCode:             p.RegionCode,
		AcknowledgementState:   ackState,
		PurchaseCompletionTime: completionTime,
		ProductLineItem: []dto.ProductLineItemV2{
			{ProductId: string(p.ProductID), Quantity: qty},
		},
		PurchaseStateContext: dto.PurchaseStateContextV2{PurchaseState: purchaseState},
	}
}

// GetSubscriptionV2 retrieves subscription purchase state in the v2 format.
type GetSubscriptionV2 struct {
	repo   repository.PurchaseRepository
	logger *zap.Logger
}

func NewGetSubscriptionV2(repo repository.PurchaseRepository, logger *zap.Logger) *GetSubscriptionV2 {
	return &GetSubscriptionV2{repo: repo, logger: logger}
}

func (uc *GetSubscriptionV2) Execute(ctx context.Context, pkg, subID, token string) (*dto.SubscriptionPurchaseV2Response, error) {
	purchase, err := uc.repo.GetSubscription(ctx,
		entity.PackageName(pkg),
		entity.SubscriptionID(subID),
		entity.PurchaseToken(token),
	)
	if err != nil {
		return nil, err
	}
	return toSubscriptionV2Response(purchase, subID), nil
}

// GetProductV2 retrieves product purchase state in the v2 format.
type GetProductV2 struct {
	repo   repository.PurchaseRepository
	logger *zap.Logger
}

func NewGetProductV2(repo repository.PurchaseRepository, logger *zap.Logger) *GetProductV2 {
	return &GetProductV2{repo: repo, logger: logger}
}

func (uc *GetProductV2) Execute(ctx context.Context, pkg, productID, token string) (*dto.ProductPurchaseV2Response, error) {
	purchase, err := uc.repo.GetProduct(ctx,
		entity.PackageName(pkg),
		entity.ProductID(productID),
		entity.PurchaseToken(token),
	)
	if err != nil {
		return nil, err
	}
	return toProductV2Response(purchase), nil
}

// ConsumeProduct marks a product purchase as consumed (consumptionState → 1).
type ConsumeProduct struct {
	repo   repository.PurchaseRepository
	logger *zap.Logger
}

func NewConsumeProduct(repo repository.PurchaseRepository, logger *zap.Logger) *ConsumeProduct {
	return &ConsumeProduct{repo: repo, logger: logger}
}

func (uc *ConsumeProduct) Execute(ctx context.Context, pkg, productID, token string) error {
	purchase, err := uc.repo.GetProduct(ctx,
		entity.PackageName(pkg),
		entity.ProductID(productID),
		entity.PurchaseToken(token),
	)
	if err != nil {
		return err
	}
	if err := purchase.Consume(); err != nil {
		return err
	}
	return uc.repo.UpdateProduct(ctx, purchase)
}

// DeferSubscriptionV2 extends subscription expiry using ISO 8601 duration offset.
// For mock purposes we simply add 30 days to current expiry when deferDuration is set.
type DeferSubscriptionV2 struct {
	repo   repository.PurchaseRepository
	logger *zap.Logger
}

func NewDeferSubscriptionV2(repo repository.PurchaseRepository, logger *zap.Logger) *DeferSubscriptionV2 {
	return &DeferSubscriptionV2{repo: repo, logger: logger}
}

func (uc *DeferSubscriptionV2) Execute(ctx context.Context, pkg, token string) (*dto.DeferSubscriptionPurchaseV2Response, error) {
	purchase, err := uc.repo.GetSubscription(ctx,
		entity.PackageName(pkg),
		"",
		entity.PurchaseToken(token),
	)
	if err != nil {
		return nil, err
	}
	// Extend by 30 days
	newExpiry := purchase.ExpiryTimeMillis + int64(30*24*time.Hour/time.Millisecond)
	if err := purchase.Defer(newExpiry); err != nil {
		return nil, err
	}
	if err := uc.repo.UpdateSubscription(ctx, purchase); err != nil {
		return nil, err
	}
	newExpiryRFC := time.UnixMilli(newExpiry).UTC().Format(time.RFC3339)
	return &dto.DeferSubscriptionPurchaseV2Response{
		ItemExpiryTimeDetails: []dto.ItemExpiryTimeDetails{
			{ProductId: string(purchase.SubscriptionID), ExpiryTime: newExpiryRFC},
		},
	}, nil
}

// --- helpers ---

func toSubscriptionResponse(p *entity.SubscriptionPurchase, subID string) *dto.SubscriptionPurchaseResponse {
	r := &dto.SubscriptionPurchaseResponse{
		Kind:                 p.Kind,
		StartTimeMillis:      strconv.FormatInt(p.PurchaseTimeMillis, 10),
		ExpiryTimeMillis:     strconv.FormatInt(p.ExpiryTimeMillis, 10),
		AutoRenewing:         p.AutoRenewing,
		PriceCurrencyCode:    "USD",
		PriceAmountMicros:    "0",
		CountryCode:          p.RegionCode,
		DeveloperPayload:     "",
		AcknowledgementState: int(p.AcknowledgementState),
		PurchaseState:        int(p.PurchaseState),
		PurchaseTimeMillis:   strconv.FormatInt(p.PurchaseTimeMillis, 10),
		OrderID:              p.OrderID,
		ProductID:            subID,
		RegionCode:           p.RegionCode,
	}
	if p.PaymentState != nil {
		ps := int(*p.PaymentState)
		r.PaymentState = &ps
	}
	if p.CancelReason != nil {
		cr := int(*p.CancelReason)
		r.CancelReason = &cr
	}
	if p.UserCancellationTimeMillis != nil {
		s := strconv.FormatInt(*p.UserCancellationTimeMillis, 10)
		r.UserCancellationTimeMillis = &s
	}
	return r
}

func toProductResponse(p *entity.ProductPurchase) *dto.ProductPurchaseResponse {
	r := &dto.ProductPurchaseResponse{
		Kind:                 p.Kind,
		PurchaseTimeMillis:   strconv.FormatInt(p.PurchaseTimeMillis, 10),
		PurchaseState:        int(p.PurchaseState),
		ConsumptionState:     p.ConsumptionState,
		DeveloperPayload:     p.DeveloperPayload,
		OrderID:              p.OrderID,
		AcknowledgementState: int(p.AcknowledgementState),
		PurchaseToken:        string(p.Token),
		ProductID:            string(p.ProductID),
		Quantity:             p.Quantity,
		RegionCode:           p.RegionCode,
	}
	if r.Quantity == 0 {
		r.Quantity = 1
	}
	return r
}



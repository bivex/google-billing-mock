package repository

import (
	"context"
	"errors"

	"github.com/bivex/google-billing-mock/internal/domain/entity"
)

// ErrNotFound is returned when a purchase token doesn't exist.
var ErrNotFound = errors.New("purchase not found")

// PurchaseRepository is the port for purchase state persistence.
type PurchaseRepository interface {
	GetSubscription(ctx context.Context, pkg entity.PackageName, subID entity.SubscriptionID, token entity.PurchaseToken) (*entity.SubscriptionPurchase, error)
	UpdateSubscription(ctx context.Context, purchase *entity.SubscriptionPurchase) error
	GetProduct(ctx context.Context, pkg entity.PackageName, productID entity.ProductID, token entity.PurchaseToken) (*entity.ProductPurchase, error)
	UpdateProduct(ctx context.Context, purchase *entity.ProductPurchase) error

	SeedSubscription(token entity.PurchaseToken, purchase *entity.SubscriptionPurchase) error
	SeedProduct(token entity.PurchaseToken, purchase *entity.ProductPurchase) error
	DeleteSubscription(token entity.PurchaseToken) error
	DeleteProduct(token entity.PurchaseToken) error

	ListSubscriptions() []*entity.SubscriptionPurchase
	ListProducts() []*entity.ProductPurchase
}

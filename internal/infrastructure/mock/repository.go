package mock

import (
	"context"
	"sync"

	"github.com/bivex/google-billing-mock/internal/domain/entity"
	"github.com/bivex/google-billing-mock/internal/domain/repository"
)

// InMemoryRepository is a thread-safe in-memory implementation of PurchaseRepository.
// On cache miss it falls back to ScenarioManager to materialise purchases on the fly.
type InMemoryRepository struct {
	mu            sync.RWMutex
	subscriptions map[string]*entity.SubscriptionPurchase
	products      map[string]*entity.ProductPurchase
	scenarioMgr   *ScenarioManager
}

// NewInMemoryRepository creates a new in-memory repository.
func NewInMemoryRepository(sm *ScenarioManager) *InMemoryRepository {
	return &InMemoryRepository{
		subscriptions: make(map[string]*entity.SubscriptionPurchase),
		products:      make(map[string]*entity.ProductPurchase),
		scenarioMgr:   sm,
	}
}

// GetSubscription retrieves a subscription by token.
// Materialises from scenario on miss.
func (r *InMemoryRepository) GetSubscription(_ context.Context, _ entity.PackageName, _ entity.SubscriptionID, token entity.PurchaseToken) (*entity.SubscriptionPurchase, error) {
	key := string(token)

	r.mu.RLock()
	if p, ok := r.subscriptions[key]; ok {
		r.mu.RUnlock()
		return p, nil
	}
	r.mu.RUnlock()

	// Try scenario match.
	purchase, err := r.scenarioMgr.MatchSubscriptionScenario(key)
	if err != nil {
		return nil, err
	}
	if purchase == nil {
		return nil, repository.ErrNotFound
	}

	// Store for subsequent mutations.
	r.mu.Lock()
	r.subscriptions[key] = purchase
	r.mu.Unlock()
	return purchase, nil
}

// UpdateSubscription persists a modified subscription.
func (r *InMemoryRepository) UpdateSubscription(_ context.Context, purchase *entity.SubscriptionPurchase) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.subscriptions[string(purchase.Token)] = purchase
	return nil
}

// GetProduct retrieves a product purchase by token.
func (r *InMemoryRepository) GetProduct(_ context.Context, _ entity.PackageName, _ entity.ProductID, token entity.PurchaseToken) (*entity.ProductPurchase, error) {
	key := string(token)

	r.mu.RLock()
	if p, ok := r.products[key]; ok {
		r.mu.RUnlock()
		return p, nil
	}
	r.mu.RUnlock()

	purchase, err := r.scenarioMgr.MatchProductScenario(key)
	if err != nil {
		return nil, err
	}
	if purchase == nil {
		return nil, repository.ErrNotFound
	}

	r.mu.Lock()
	r.products[key] = purchase
	r.mu.Unlock()
	return purchase, nil
}

// UpdateProduct persists a modified product purchase.
func (r *InMemoryRepository) UpdateProduct(_ context.Context, purchase *entity.ProductPurchase) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.products[string(purchase.Token)] = purchase
	return nil
}

// SeedSubscription adds or replaces a subscription directly.
func (r *InMemoryRepository) SeedSubscription(token entity.PurchaseToken, purchase *entity.SubscriptionPurchase) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.subscriptions[string(token)] = purchase
	return nil
}

// SeedProduct adds or replaces a product purchase directly.
func (r *InMemoryRepository) SeedProduct(token entity.PurchaseToken, purchase *entity.ProductPurchase) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.products[string(token)] = purchase
	return nil
}

// DeleteSubscription removes a subscription.
func (r *InMemoryRepository) DeleteSubscription(token entity.PurchaseToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.subscriptions, string(token))
	return nil
}

// DeleteProduct removes a product purchase.
func (r *InMemoryRepository) DeleteProduct(token entity.PurchaseToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.products, string(token))
	return nil
}

// ListSubscriptions returns all stored subscriptions.
func (r *InMemoryRepository) ListSubscriptions() []*entity.SubscriptionPurchase {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*entity.SubscriptionPurchase, 0, len(r.subscriptions))
	for _, v := range r.subscriptions {
		out = append(out, v)
	}
	return out
}

// ListProducts returns all stored product purchases.
func (r *InMemoryRepository) ListProducts() []*entity.ProductPurchase {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*entity.ProductPurchase, 0, len(r.products))
	for _, v := range r.products {
		out = append(out, v)
	}
	return out
}

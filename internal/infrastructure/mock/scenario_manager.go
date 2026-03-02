package mock

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bivex/google-billing-mock/internal/domain/entity"
)

// ScenarioConfig describes a named test scenario matched by token prefix.
type ScenarioConfig struct {
	Name                 string `json:"name"`
	TokenPrefix          string `json:"token_prefix"`
	Type                 string `json:"type"` // "subscription" or "product"
	PurchaseState        int    `json:"purchase_state"`
	PaymentState         *int   `json:"payment_state"`
	AcknowledgementState int    `json:"acknowledgement_state"`
	AutoRenewing         bool   `json:"auto_renewing"`
	// ExpiryOffsetSeconds is relative to time.Now() when a purchase is materialised.
	ExpiryOffsetSeconds int64  `json:"expiry_offset_seconds"`
	CancelReason        *int   `json:"cancel_reason,omitempty"`
	ErrorCode           *int   `json:"error_code,omitempty"`
	ErrorMessage        string `json:"error_message,omitempty"`
}

// ScenarioError is returned when a scenario defines a forced error response.
type ScenarioError struct {
	Code    int
	Message string
}

func (e *ScenarioError) Error() string { return fmt.Sprintf("scenario error %d: %s", e.Code, e.Message) }

// ScenarioManager loads and matches test scenarios.
type ScenarioManager struct {
	mu        sync.RWMutex
	scenarios []ScenarioConfig
}

// NewScenarioManager returns an empty ScenarioManager.
func NewScenarioManager() *ScenarioManager { return &ScenarioManager{} }

// LoadFromFile reads scenarios from a JSON file.
func (sm *ScenarioManager) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading scenarios file %q: %w", path, err)
	}
	var scenarios []ScenarioConfig
	if err := json.Unmarshal(data, &scenarios); err != nil {
		return fmt.Errorf("parsing scenarios file: %w", err)
	}
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.scenarios = scenarios
	return nil
}

// AddScenario appends or replaces (by name) a scenario.
func (sm *ScenarioManager) AddScenario(s ScenarioConfig) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for i, existing := range sm.scenarios {
		if existing.Name == s.Name {
			sm.scenarios[i] = s
			return
		}
	}
	sm.scenarios = append(sm.scenarios, s)
}

// DeleteScenario removes a scenario by name. Returns true if deleted.
func (sm *ScenarioManager) DeleteScenario(name string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for i, s := range sm.scenarios {
		if s.Name == name {
			sm.scenarios = append(sm.scenarios[:i], sm.scenarios[i+1:]...)
			return true
		}
	}
	return false
}

// ListScenarios returns a snapshot of all scenarios.
func (sm *ScenarioManager) ListScenarios() []ScenarioConfig {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	out := make([]ScenarioConfig, len(sm.scenarios))
	copy(out, sm.scenarios)
	return out
}

// MatchSubscriptionScenario finds a scenario by token prefix and materialises a SubscriptionPurchase.
// Returns ScenarioError if the matched scenario defines a forced error.
func (sm *ScenarioManager) MatchSubscriptionScenario(token string) (*entity.SubscriptionPurchase, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	for _, s := range sm.scenarios {
		if s.Type != "subscription" {
			continue
		}
		if s.TokenPrefix != "" && !strings.HasPrefix(token, s.TokenPrefix) {
			continue
		}
		if s.ErrorCode != nil {
			return nil, &ScenarioError{Code: *s.ErrorCode, Message: s.ErrorMessage}
		}
		return materialiseSubscription(s, entity.PurchaseToken(token)), nil
	}
	return nil, nil
}

// MatchProductScenario finds a scenario by token prefix and materialises a ProductPurchase.
func (sm *ScenarioManager) MatchProductScenario(token string) (*entity.ProductPurchase, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	for _, s := range sm.scenarios {
		if s.Type != "product" {
			continue
		}
		if s.TokenPrefix != "" && !strings.HasPrefix(token, s.TokenPrefix) {
			continue
		}
		if s.ErrorCode != nil {
			return nil, &ScenarioError{Code: *s.ErrorCode, Message: s.ErrorMessage}
		}
		return materialiseProduct(s, entity.PurchaseToken(token)), nil
	}
	return nil, nil
}

func newOrderID() string {
	return fmt.Sprintf("GPA.%04d-%04d-%04d-%05d",
		rand.Intn(10000), rand.Intn(10000), rand.Intn(10000), rand.Intn(100000))
}
func materialiseSubscription(s ScenarioConfig, token entity.PurchaseToken) *entity.SubscriptionPurchase {
	ps := entity.PaymentStateReceived
	if s.PaymentState != nil {
		ps = entity.PaymentState(*s.PaymentState)
	}
	expiry := time.Now().Add(time.Duration(s.ExpiryOffsetSeconds) * time.Second).UnixMilli()

	purchase := &entity.SubscriptionPurchase{
		Token:                token,
		SubscriptionID:       "mock_subscription",
		PackageName:          "com.mock.app",
		Kind:                 "androidpublisher#subscriptionPurchase",
		PurchaseState:        entity.PurchaseState(s.PurchaseState),
		PaymentState:         &ps,
		AcknowledgementState: entity.AcknowledgementState(s.AcknowledgementState),
		ExpiryTimeMillis:     expiry,
		PurchaseTimeMillis:   time.Now().Add(-7 * 24 * time.Hour).UnixMilli(),
		AutoRenewing:         s.AutoRenewing,
		OrderID:              newOrderID(),
		RegionCode:           "US",
	}
	if s.CancelReason != nil {
		cr := entity.CancelReason(*s.CancelReason)
		purchase.CancelReason = &cr
		now := time.Now().UnixMilli()
		purchase.UserCancellationTimeMillis = &now
	}
	return purchase
}

func materialiseProduct(s ScenarioConfig, token entity.PurchaseToken) *entity.ProductPurchase {
	ps := entity.PaymentStateReceived
	if s.PaymentState != nil {
		ps = entity.PaymentState(*s.PaymentState)
	}
	return &entity.ProductPurchase{
		Token:                token,
		ProductID:            "mock_product",
		PackageName:          "com.mock.app",
		Kind:                 "androidpublisher#productPurchase",
		PurchaseState:        entity.PurchaseState(s.PurchaseState),
		PaymentState:         &ps,
		AcknowledgementState: entity.AcknowledgementState(s.AcknowledgementState),
		PurchaseTimeMillis:   time.Now().Add(-time.Hour).UnixMilli(),
		OrderID:              newOrderID(),
		RegionCode:           "US",
		Quantity:             1,
	}
}

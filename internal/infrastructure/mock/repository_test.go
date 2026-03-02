package mock_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/bivex/google-billing-mock/internal/domain/entity"
	"github.com/bivex/google-billing-mock/internal/domain/repository"
	"github.com/bivex/google-billing-mock/internal/infrastructure/mock"
)

func newRepoWithScenarios(t *testing.T) *mock.InMemoryRepository {
	t.Helper()
	sm := mock.NewScenarioManager()
	paymentState := 1
	sm.AddScenario(mock.ScenarioConfig{
		Name:                 "active",
		TokenPrefix:          "active_",
		Type:                 "subscription",
		PurchaseState:        0,
		PaymentState:         &paymentState,
		AcknowledgementState: 0,
		AutoRenewing:         true,
		ExpiryOffsetSeconds:  86400,
	})
	errCode := 410
	sm.AddScenario(mock.ScenarioConfig{
		Name:        "invalid",
		TokenPrefix: "invalid_",
		Type:        "subscription",
		ErrorCode:   &errCode,
		ErrorMessage: "Purchase token no longer valid",
	})
	return mock.NewInMemoryRepository(sm)
}

func TestRepository_GetSubscription_ScenarioMatch(t *testing.T) {
	repo := newRepoWithScenarios(t)
	ctx := context.Background()

	sub, err := repo.GetSubscription(ctx, "com.test", "sub1", "active_token123")
	require.NoError(t, err)
	require.NotNil(t, sub)
	assert.Equal(t, entity.PurchaseState(0), sub.PurchaseState)
}

func TestRepository_GetSubscription_NotFound(t *testing.T) {
	repo := newRepoWithScenarios(t)
	ctx := context.Background()

	_, err := repo.GetSubscription(ctx, "com.test", "sub1", "unknown_token_xyz")
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestRepository_GetSubscription_ScenarioError(t *testing.T) {
	repo := newRepoWithScenarios(t)
	ctx := context.Background()

	_, err := repo.GetSubscription(ctx, "com.test", "sub1", "invalid_token123")
	require.Error(t, err)
	var se *mock.ScenarioError
	assert.ErrorAs(t, err, &se)
	assert.Equal(t, 410, se.Code)
}

func TestRepository_SeedAndGet(t *testing.T) {
	sm := mock.NewScenarioManager()
	repo := mock.NewInMemoryRepository(sm)
	ctx := context.Background()

	purchase := entity.NewSubscriptionPurchase("seeded_tok", "sub1", "com.test")
	err := repo.SeedSubscription("seeded_tok", purchase)
	require.NoError(t, err)

	got, err := repo.GetSubscription(ctx, "com.test", "sub1", "seeded_tok")
	require.NoError(t, err)
	assert.Equal(t, entity.PurchaseToken("seeded_tok"), got.Token)
}

func TestRepository_UpdateSubscription(t *testing.T) {
	repo := newRepoWithScenarios(t)
	ctx := context.Background()

	sub, err := repo.GetSubscription(ctx, "com.test", "sub1", "active_tok")
	require.NoError(t, err)

	_ = sub.Acknowledge()
	require.NoError(t, repo.UpdateSubscription(ctx, sub))

	updated, err := repo.GetSubscription(ctx, "com.test", "sub1", "active_tok")
	require.NoError(t, err)
	assert.Equal(t, entity.AcknowledgementStateAcknowledged, updated.AcknowledgementState)
}

func TestRepository_DeleteSubscription(t *testing.T) {
	repo := newRepoWithScenarios(t)
	ctx := context.Background()

	// Materialise first
	_, err := repo.GetSubscription(ctx, "com.test", "sub1", "active_del")
	require.NoError(t, err)

	require.NoError(t, repo.DeleteSubscription("active_del"))

	// After delete the scenario will re-materialise, so we just confirm no error on re-fetch
	_, err = repo.GetSubscription(ctx, "com.test", "sub1", "active_del")
	require.NoError(t, err) // scenario match creates fresh entry
}

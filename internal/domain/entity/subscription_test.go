package entity_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/bivex/google-billing-mock/internal/domain/entity"
	"github.com/bivex/google-billing-mock/internal/domain/event"
)

func TestSubscriptionPurchase_Acknowledge_Success(t *testing.T) {
	sub := entity.NewSubscriptionPurchase("tok1", "sub1", "com.test")
	assert.Equal(t, entity.AcknowledgementStatePending, sub.AcknowledgementState)

	err := sub.Acknowledge()
	require.NoError(t, err)
	assert.Equal(t, entity.AcknowledgementStateAcknowledged, sub.AcknowledgementState)

	events := sub.DomainEvents()
	require.Len(t, events, 1)
	assert.Equal(t, "PurchaseAcknowledged", events[0].EventName())
}

func TestSubscriptionPurchase_Acknowledge_AlreadyAcknowledged(t *testing.T) {
	sub := entity.NewSubscriptionPurchase("tok2", "sub1", "com.test")
	require.NoError(t, sub.Acknowledge())

	err := sub.Acknowledge()
	assert.Error(t, err, "second acknowledge should fail")
}

func TestSubscriptionPurchase_Cancel(t *testing.T) {
	sub := entity.NewSubscriptionPurchase("tok3", "sub1", "com.test")
	sub.Cancel(entity.CancelReasonUserCanceled)

	assert.Equal(t, entity.PurchaseStateCanceled, sub.PurchaseState)
	assert.False(t, sub.AutoRenewing)
	require.NotNil(t, sub.CancelReason)
	assert.Equal(t, entity.CancelReasonUserCanceled, *sub.CancelReason)

	var found bool
	for _, ev := range sub.DomainEvents() {
		if _, ok := ev.(event.SubscriptionCanceled); ok {
			found = true
		}
	}
	assert.True(t, found, "SubscriptionCanceled event expected")
}

func TestSubscriptionPurchase_Revoke(t *testing.T) {
	sub := entity.NewSubscriptionPurchase("tok4", "sub1", "com.test")
	sub.Revoke()

	assert.Equal(t, entity.PurchaseStateCanceled, sub.PurchaseState)
	assert.False(t, sub.AutoRenewing)
	// ExpiryTimeMillis should be set to now (within a few seconds)
	assert.WithinDuration(t, time.Now(), time.UnixMilli(sub.ExpiryTimeMillis), 2*time.Second)
}

func TestSubscriptionPurchase_Defer_Success(t *testing.T) {
	sub := entity.NewSubscriptionPurchase("tok5", "sub1", "com.test")
	future := time.Now().Add(60 * 24 * time.Hour).UnixMilli()

	err := sub.Defer(future)
	require.NoError(t, err)
	assert.Equal(t, future, sub.ExpiryTimeMillis)
}

func TestSubscriptionPurchase_Defer_PastTime(t *testing.T) {
	sub := entity.NewSubscriptionPurchase("tok6", "sub1", "com.test")
	past := time.Now().Add(-time.Hour).UnixMilli()

	err := sub.Defer(past)
	assert.Error(t, err)
}

func TestSubscriptionPurchase_IsExpired(t *testing.T) {
	sub := entity.NewSubscriptionPurchase("tok7", "sub1", "com.test")
	sub.ExpiryTimeMillis = time.Now().Add(-time.Hour).UnixMilli()
	assert.True(t, sub.IsExpired())

	sub.ExpiryTimeMillis = time.Now().Add(time.Hour).UnixMilli()
	assert.False(t, sub.IsExpired())
}

func TestSubscriptionPurchase_ClearEvents(t *testing.T) {
	sub := entity.NewSubscriptionPurchase("tok8", "sub1", "com.test")
	_ = sub.Acknowledge()
	require.NotEmpty(t, sub.DomainEvents())

	sub.ClearEvents()
	assert.Empty(t, sub.DomainEvents())
}

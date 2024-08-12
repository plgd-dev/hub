package jwt

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const TEST_TIMEOUT = time.Second * 10

func TestTokenRecord_IsExpired(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name       string
		recordTime time.Time
		want       bool
	}{
		{"Not expired", now.Add(time.Hour), false},
		{"Expired", now.Add(-time.Hour), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := newTokenRecord(false, tt.recordTime, nil)
			require.Equal(t, tt.want, record.IsExpired(now))
		})
	}
}

func TestTokenIssuerCacheSetAndGetToken(t *testing.T) {
	cache := newTokenIssuerCache(&HTTPClient{Client: &http.Client{}, tokenEndpoint: "http://example.com"})

	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()
	tokenID := uuid.New()
	// token doesn't exist yet, so we should get a future with a set function
	_, setTokenOrError := cache.getValidTokenRecordOrFuture(tokenID)
	require.NotNil(t, setTokenOrError)

	// we can wait on the future in other goroutine
	// -> setting error on the future should unblock the goroutine
	waiting := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		tf2, setToken2 := cache.getValidTokenRecordOrFuture(tokenID)
		assert.Nil(t, setToken2)
		close(waiting)
		_, err := tf2.Get(ctx)
		assert.Error(t, err)
	}()

	<-waiting
	cache.removeTokenRecordAndSetErrorOnFuture(tokenID, setTokenOrError, errors.New("test"))
	select {
	case <-done:
	case <-ctx.Done():
		require.Fail(t, "timeout")
	}

	// get a new future
	_, setTokenOrError = cache.getValidTokenRecordOrFuture(tokenID)
	require.NotNil(t, setTokenOrError)

	// -> setting an expired token record should result in a future with a set function being returned
	expiredIDs := []uuid.UUID{}
	tr := newTokenRecord(false, time.Now().Add(-time.Hour), func(u uuid.UUID) {
		expiredIDs = append(expiredIDs, u)
	})
	cache.setTokenRecord(tokenID, tr)
	_, setTokenOrError = cache.getValidTokenRecordOrFuture(tokenID)
	require.NotNil(t, setTokenOrError)
	require.Len(t, expiredIDs, 1)
	require.Equal(t, tokenID, expiredIDs[0])

	// -> finally, set valid token record
	tr = newTokenRecord(false, time.Now().Add(time.Hour), nil)
	waiting = make(chan struct{})
	done = make(chan struct{})
	go func() {
		defer close(done)
		tf2, setToken2 := cache.getValidTokenRecordOrFuture(tokenID)
		assert.Nil(t, setToken2)
		close(waiting)
		result, err := tf2.Get(ctx)
		assert.NoError(t, err)
		assert.Equal(t, tr, result)
	}()

	<-waiting
	cache.setTokenRecordAndWaitingFuture(tokenID, tr, setTokenOrError)
	select {
	case <-done:
	case <-ctx.Done():
		require.Fail(t, "timeout")
	}

	// cache should return a token record now, not a future
	tf, _ := cache.getValidTokenRecordOrFuture(tokenID)
	_, ok := tf.tokenOrFuture.(*tokenRecord)
	require.True(t, ok)
}

func TestTokenIssuerCacheCheckExpirations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), TEST_TIMEOUT)
	defer cancel()
	cache := newTokenIssuerCache(&HTTPClient{Client: &http.Client{}, tokenEndpoint: "http://example.com"})

	now := time.Now()
	tokenID1 := uuid.New()
	expiredIDs := []uuid.UUID{}
	onExpire := func(u uuid.UUID) {
		expiredIDs = append(expiredIDs, u)
	}
	tokenRecord1 := newTokenRecord(false, now.Add(-time.Hour), onExpire)
	cache.setTokenRecord(tokenID1, tokenRecord1)

	tokenID2 := uuid.New()
	tokenRecord2 := newTokenRecord(false, now.Add(time.Hour), onExpire)
	cache.setTokenRecord(tokenID2, tokenRecord2)

	cache.checkExpirations(now)

	// tokenRecord1 should have been removed and we should get a future with a set function
	_, setTf1 := cache.getValidTokenRecordOrFuture(tokenID1)
	require.NotNil(t, setTf1)
	require.Len(t, expiredIDs, 1)
	require.Equal(t, tokenID1, expiredIDs[0])
	// tokenRecord2 should still be there
	tf2, setTf2 := cache.getValidTokenRecordOrFuture(tokenID2)
	require.Nil(t, setTf2)
	result, err := tf2.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, tokenRecord2, result)
}

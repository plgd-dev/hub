package jwt

import (
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/go-coap/v3/pkg/cache"
)

type TokenRecord struct {
	Blacklisted bool
}

type TokenCache struct {
	cache      *cache.Cache[uuid.UUID, TokenRecord]
	expiration time.Duration
}

func NewTokenCache(expiration time.Duration) *TokenCache {
	return &TokenCache{
		cache:      cache.NewCache[uuid.UUID, TokenRecord](),
		expiration: expiration,
	}
}

func getCacheKeyID(tokenID, issuer string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(tokenID+"."+issuer))
}

func (t *TokenCache) Load(tokenID, issuer string) (TokenRecord, bool) {
	keyID := getCacheKeyID(tokenID, issuer)
	value := t.cache.Load(keyID)
	if value == nil {
		return TokenRecord{}, false
	}
	return value.Data(), true
}

func (t *TokenCache) AddBlacklisted(tokenID, issuer string, validUntil time.Time) {
	// pkgTime.Unix(expiresAt, 0)
	keyID := getCacheKeyID(tokenID, issuer)
	// use Replace
	//// if not in cache -> add
	//// if in cache ->
	////    - previously whitelisted -> replace
	////    - previously blacklisted -> replace is not necessary, but it doesn't break anything
	t.cache.Replace(keyID, cache.NewElement(TokenRecord{Blacklisted: true}, validUntil, nil))
}

func (t *TokenCache) AddWhitelisted(tokenID, issuer string) bool {
	keyID := getCacheKeyID(tokenID, issuer)
	var validUntil time.Time
	if t.expiration > 0 {
		validUntil = time.Now().Add(t.expiration)
	}
	// use ReplaceWithFunc
	//// if not in cache -> add
	//// if in cache ->
	////    - previously whitelisted -> replace the record to extend the expiration time
	////    - previously blacklisted -> cannot replace, blacklisted tokens stay blacklisted until expiration
	newTr := cache.NewElement(TokenRecord{Blacklisted: false}, validUntil, nil)
	var actual *cache.Element[TokenRecord]
	now := time.Now()
	t.cache.ReplaceWithFunc(keyID, func(oldValue *cache.Element[TokenRecord], oldLoaded bool) (*cache.Element[TokenRecord], bool) {
		if oldLoaded && oldValue.Data().Blacklisted {
			// blacklisted tokens stay blacklisted until expiration
			if !oldValue.IsExpired(now) {
				actual = oldValue
				return oldValue, false
			}
		}
		actual = newTr
		return newTr, false
	})
	return actual == newTr
}

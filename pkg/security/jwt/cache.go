package jwt

import (
	"time"

	"github.com/plgd-dev/go-coap/v3/pkg/cache"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
)

type TokenRecord struct {
	Blacklisted bool
}

type TokenCache struct {
	cache *cache.Cache[string, TokenRecord]
}

func NewTokenCache() *TokenCache {
	return &TokenCache{
		cache: cache.NewCache[string, TokenRecord](),
	}
}

func getCacheKey(tokenID, issuer string) string {
	return tokenID + "." + issuer
}

func (t *TokenCache) Load(tokenID, issuer string) (TokenRecord, bool) {
	key := getCacheKey(tokenID, issuer)
	value := t.cache.Load(key)
	if value == nil {
		return TokenRecord{}, false
	}
	return value.Data(), true
}

func (t *TokenCache) AddBlacklisted(tokenID, issuer string, expiresAt int64) {
	key := getCacheKey(tokenID, issuer)
	// use Replace
	//// if not in cache -> add
	//// if in cache ->
	////    - previously whitelisted -> replace
	////    - previously blacklisted -> replace is not necessary, but it doesn't break anything
	t.cache.Replace(key, cache.NewElement(TokenRecord{Blacklisted: true}, pkgTime.Unix(expiresAt, 0), nil))
}

func (t *TokenCache) AddWhitelisted(tokenID, issuer string) bool {
	key := getCacheKey(tokenID, issuer)
	// use ReplaceWithFunc
	//// if not in cache -> add
	//// if in cache ->
	////    - previously whitelisted -> replace the record to extend the expiration time
	////    - previously blacklisted -> cannot replace, blacklisted tokens stay blacklisted until expiration
	//// TODO: configure expiration interval -> for now use 10s
	newTr := cache.NewElement(TokenRecord{Blacklisted: false}, time.Now().Add(10*time.Second), nil)
	var actual *cache.Element[TokenRecord]
	now := time.Now()
	t.cache.ReplaceWithFunc(key, func(oldValue *cache.Element[TokenRecord], oldLoaded bool) (*cache.Element[TokenRecord], bool) {
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

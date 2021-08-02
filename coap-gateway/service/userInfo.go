package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/pkg/log"
	pkgTime "github.com/plgd-dev/cloud/pkg/time"
	"golang.org/x/oauth2"
)

type userInfo map[string]interface{}

/// Get user info from oauth server
func getUserInfo(ctx context.Context, service *Service, accessToken string) (userInfo, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, service.provider.HTTPClient.HTTP())
	oauthClient := service.provider.OAuth2.Client(ctx, &oauth2.Token{
		AccessToken: accessToken,
		TokenType:   "bearer",
	})
	resp, err := oauthClient.Get(service.provider.UserInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed oauth userinfo request: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Errorf("failed to close userinfo response body: %v", err)
		}
	}()

	var profile map[string]interface{}
	if err = json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to decode userinfo request body: %v", err)
	}
	return profile, nil
}

/// Get expiration time (exp) from user info map.
/// It might not be set, in that case zero time and no error are returned.
func (u userInfo) getExpirationTime() (time.Time, error) {
	const expKey = "exp"
	v, ok := u[expKey]
	if !ok {
		return time.Time{}, nil
	}

	exp, ok := v.(float64) // all integers are float64 in json
	if !ok {
		return time.Time{}, fmt.Errorf("invalid userinfo: invalid %v value type", expKey)
	}
	return pkgTime.Unix(int64(exp), 0), nil
}

/// Validate that ownerClaim is set and that it matches given user ID
func (u userInfo) validateOwnerClaim(ocKey string, userID string) error {
	v, ok := u[ocKey]
	if !ok {
		return fmt.Errorf("invalid userinfo: %v not set", ocKey)
	}
	ownerClaim, ok := v.(string)
	if !ok {
		return fmt.Errorf("invalid userinfo: %v", "invalid ownerClaim value type")
	}
	if ownerClaim != userID {
		return fmt.Errorf("invalid ownerClaim: %v", userID)
	}
	return nil
}

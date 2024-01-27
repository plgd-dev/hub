//go:build test
// +build test

package service

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	coapService "github.com/plgd-dev/hub/v2/pkg/net/coap/service"
	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	oauthUri "github.com/plgd-dev/hub/v2/test/oauth-server/uri"
	"github.com/plgd-dev/hub/v2/test/security"
	"github.com/stretchr/testify/require"
)

func makeConfig() Config {
	var cfg Config
	cfg.APIs.COAP.KeepAlive = new(coapService.KeepAlive)
	cfg.APIs.COAP.KeepAlive.Timeout = time.Second * 20
	cfg.APIs.COAP.Authorization.OwnerClaim = config.OWNER_CLAIM
	cfg.APIs.COAP.Authorization.DeviceIDClaim = oauthUri.DeviceIDClaimKey
	return cfg
}

func TestSignInPostHandlerVerifyToken(t *testing.T) {
	jwks := security.NewTestJwks(t)
	defer jwks.Close()

	cfg := makeConfig()
	service := &Service{
		config:       cfg,
		jwtValidator: oauthTest.GetJWTValidator(jwks.URL()),
	}
	client := newSession(service, nil, "", time.Time{})

	type test struct {
		name    string
		req     CoapSignInReq
		wantErr interface{}
	}

	const userID = "user"
	const deviceID = "device"
	tests := []test{
		{
			name: "invalid token",
			req: CoapSignInReq{
				UserID:      userID,
				AccessToken: "123",
			},
			wantErr: pkgJwt.ErrCannotParseToken,
		},
		{
			name: "missing deviceID claim",
			req: CoapSignInReq{
				UserID: userID,
				AccessToken: security.CreateJwksToken(t, jwt.MapClaims{
					config.OWNER_CLAIM: userID,
				}),
			},
			wantErr: "access token doesn't contain the required device id claim",
		},
		{
			name: "invalid deviceID claim",
			req: CoapSignInReq{
				UserID: userID,
				AccessToken: security.CreateJwksToken(t, jwt.MapClaims{
					config.OWNER_CLAIM:        userID,
					oauthUri.DeviceIDClaimKey: 42,
				}),
			},

			wantErr: jwt.ErrInvalidType,
		},
		{
			name: "invalid expiration time claim",
			req: CoapSignInReq{
				UserID: userID,
				AccessToken: security.CreateJwksToken(t, jwt.MapClaims{
					config.OWNER_CLAIM:         userID,
					oauthUri.DeviceIDClaimKey:  deviceID,
					pkgJwt.ClaimExpirationTime: "invalid",
				}),
			},

			wantErr: jwt.ErrInvalidType,
		},
		{
			name: "non-matching userID",
			req: CoapSignInReq{
				UserID: userID,
				AccessToken: security.CreateJwksToken(t, jwt.MapClaims{
					config.OWNER_CLAIM:        "non-user",
					oauthUri.DeviceIDClaimKey: deviceID,
				}),
			},
			wantErr: pkgJwt.ErrOwnerClaimInvalid,
		},
		{
			name: "valid",
			req: CoapSignInReq{
				UserID: userID,
				AccessToken: security.CreateJwksToken(t, jwt.MapClaims{
					config.OWNER_CLAIM:        userID,
					oauthUri.DeviceIDClaimKey: deviceID,
				}),
			},
		},
	}

	for _, tt := range tests {
		tf := func(t *testing.T) {
			dID, validUntil, err := getSignInDataFromClaims(context.Background(), client, tt.req)
			if tt.wantErr == nil {
				require.NoError(t, err)
				require.Equal(t, deviceID, dID)
				require.Zero(t, validUntil)
				return
			}

			require.Error(t, err)
			if expErr, ok := tt.wantErr.(string); ok {
				require.Contains(t, err.Error(), expErr)
				return
			}
			expErr := tt.wantErr.(error)
			require.ErrorIs(t, err, expErr)
			return
		}
		t.Run(tt.name, tf)
	}
}

func TestSignOutPostHandlerVerifyToken(t *testing.T) {
	jwks := security.NewTestJwks(t)
	defer jwks.Close()

	cfg := makeConfig()
	service := &Service{
		config:       cfg,
		jwtValidator: oauthTest.GetJWTValidator(jwks.URL()),
	}
	client := newSession(service, nil, "", time.Time{})

	type test struct {
		name    string
		req     CoapSignInReq
		wantErr interface{}
	}

	const userID = "user"
	const deviceID = "device"
	tests := []test{
		{
			name: "invalid token",
			req: CoapSignInReq{
				UserID:      userID,
				AccessToken: "123",
			},
			wantErr: pkgJwt.ErrCannotParseToken,
		},
		{
			name: "missing deviceID claim",
			req: CoapSignInReq{
				UserID: userID,
				AccessToken: security.CreateJwksToken(t, jwt.MapClaims{
					config.OWNER_CLAIM: userID,
				}),
			},
			wantErr: "access token doesn't contain the required device id claim",
		},
		{
			name: "invalid deviceID claim",
			req: CoapSignInReq{
				UserID: userID,
				AccessToken: security.CreateJwksToken(t, jwt.MapClaims{
					config.OWNER_CLAIM:        userID,
					oauthUri.DeviceIDClaimKey: 42,
				}),
			},
			wantErr: jwt.ErrInvalidType,
		},
		{
			name: "non-matching userID",
			req: CoapSignInReq{
				UserID: userID,
				AccessToken: security.CreateJwksToken(t, jwt.MapClaims{
					config.OWNER_CLAIM:        "non-user",
					oauthUri.DeviceIDClaimKey: deviceID,
				}),
			},
			wantErr: pkgJwt.ErrOwnerClaimInvalid,
		},
		{
			name: "valid",
			req: CoapSignInReq{
				UserID: userID,
				AccessToken: security.CreateJwksToken(t, jwt.MapClaims{
					config.OWNER_CLAIM:        userID,
					oauthUri.DeviceIDClaimKey: userID,
				}),
			},
		},
	}

	for _, tt := range tests {
		tf := func(t *testing.T) {
			dID, err := getSignOutDataFromClaims(context.Background(), client, tt.req)
			if tt.wantErr == nil {
				require.NoError(t, err)
				require.Equal(t, userID, dID)
				return
			}

			require.Error(t, err)
			if expErr, ok := tt.wantErr.(string); ok {
				require.Contains(t, err.Error(), expErr)
				return
			}
			expErr := tt.wantErr.(error)
			require.ErrorIs(t, err, expErr)
			return
		}
		t.Run(tt.name, tf)
	}
}

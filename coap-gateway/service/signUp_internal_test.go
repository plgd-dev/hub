//go:build test
// +build test

package service

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	oauthUri "github.com/plgd-dev/hub/v2/test/oauth-server/uri"
	"github.com/plgd-dev/hub/v2/test/security"
	"github.com/stretchr/testify/require"
)

func TestSignUpPostHandlerVerifyToken(t *testing.T) {
	jwks := security.NewTestJwks(t)
	defer jwks.Close()

	cfg := makeConfig()
	service := &Service{
		config:       cfg,
		jwtValidator: oauthTest.GetJWTValidator(jwks.URL()),
	}
	client := newSession(service, nil, "", time.Time{})

	type args struct {
		req         CoapSignUpRequest
		accessToken string
	}
	type want struct {
		deviceID string
		owner    string
	}
	type test struct {
		name    string
		args    args
		want    want
		wantErr interface{}
	}
	const userID = "user"
	const deviceID = "device"
	tests := []test{
		{
			name: "invalid token",
			args: args{
				accessToken: "123",
			},
			wantErr: pkgJwt.ErrCannotParseToken,
		},
		{
			name: "missing deviceID claim",
			args: args{
				accessToken: security.CreateJwksToken(t, jwt.MapClaims{
					config.OWNER_CLAIM: userID,
				}),
			},
			wantErr: "access token doesn't contain the required device id claim",
		},
		{
			name: "invalid deviceID claim",
			args: args{
				accessToken: security.CreateJwksToken(t, jwt.MapClaims{
					oauthUri.DeviceIDClaimKey: 42,
					config.OWNER_CLAIM:        userID,
				}),
			},
			wantErr: jwt.ErrInvalidType,
		},
		{
			name: "invalid userID claim",
			args: args{
				accessToken: security.CreateJwksToken(t, jwt.MapClaims{
					oauthUri.DeviceIDClaimKey: deviceID,
					config.OWNER_CLAIM:        42,
				}),
			},
			wantErr: jwt.ErrInvalidType,
		},
		{
			name: "missing owner",
			args: args{
				accessToken: security.CreateJwksToken(t, jwt.MapClaims{
					oauthUri.DeviceIDClaimKey: deviceID,
				}),
			},
			wantErr: "cannot determine owner",
		},
		{
			name: "valid",
			args: args{
				accessToken: security.CreateJwksToken(t, jwt.MapClaims{
					oauthUri.DeviceIDClaimKey: deviceID,
					config.OWNER_CLAIM:        userID,
				}),
				req: CoapSignUpRequest{
					DeviceID: deviceID,
				},
			},
			want: want{
				deviceID: deviceID,
				owner:    userID,
			},
		},
	}

	for _, tt := range tests {
		tf := func(t *testing.T) {
			deviceID, owner, err := getSignUpDataFromClaims(context.Background(), client, tt.args.accessToken, tt.args.req)
			if tt.wantErr == nil {
				require.NoError(t, err)
				require.Equal(t, tt.want.deviceID, deviceID)
				require.Equal(t, tt.want.owner, owner)
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

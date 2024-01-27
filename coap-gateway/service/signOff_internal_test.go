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

func TestSignOffPostHandlerVerifyToken(t *testing.T) {
	jwks := security.NewTestJwks(t)
	defer jwks.Close()

	cfg := makeConfig()
	service := &Service{
		config:       cfg,
		jwtValidator: oauthTest.GetJWTValidator(jwks.URL()),
	}
	client := newSession(service, nil, "", time.Time{})

	type args struct {
		sod signOffData
	}
	type want struct {
		deviceID string
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
				sod: signOffData{
					accessToken: "123",
				},
			},
			wantErr: pkgJwt.ErrCannotParseToken,
		},
		{
			name: "invalid userID claim",
			args: args{
				sod: signOffData{
					userID: userID,
					accessToken: security.CreateJwksToken(t, jwt.MapClaims{
						oauthUri.DeviceIDClaimKey: deviceID,
						config.OWNER_CLAIM:        42,
					}),
				},
			},
			wantErr: pkgJwt.ErrOwnerClaimInvalid,
		},
		{
			name: "missing owner",
			args: args{
				sod: signOffData{
					userID: userID,
					accessToken: security.CreateJwksToken(t, jwt.MapClaims{
						oauthUri.DeviceIDClaimKey: deviceID,
					}),
				},
			},
			wantErr: pkgJwt.ErrOwnerClaimInvalid,
		},
		{
			name: "non-matching owner",
			args: args{
				sod: signOffData{
					userID: userID,
					accessToken: security.CreateJwksToken(t, jwt.MapClaims{
						oauthUri.DeviceIDClaimKey: deviceID,
						config.OWNER_CLAIM:        "other",
					}),
				},
			},
			wantErr: pkgJwt.ErrOwnerClaimInvalid,
		},
		{
			name: "missing deviceID claim",
			args: args{
				sod: signOffData{
					deviceID: deviceID,
					userID:   userID,
					accessToken: security.CreateJwksToken(t, jwt.MapClaims{
						config.OWNER_CLAIM: userID,
					}),
				},
			},
			wantErr: "access token doesn't contain the required device id claim",
		},
		{
			name: "invalid deviceID claim",
			args: args{
				sod: signOffData{
					deviceID: deviceID,
					userID:   userID,
					accessToken: security.CreateJwksToken(t, jwt.MapClaims{
						oauthUri.DeviceIDClaimKey: 42,
						config.OWNER_CLAIM:        userID,
					}),
				},
			},
			wantErr: jwt.ErrInvalidType,
		},
		{
			name: "valid",
			args: args{
				sod: signOffData{
					deviceID: deviceID,
					userID:   userID,
					accessToken: security.CreateJwksToken(t, jwt.MapClaims{
						oauthUri.DeviceIDClaimKey: deviceID,
						config.OWNER_CLAIM:        userID,
					}),
				},
			},
			want: want{
				deviceID: deviceID,
			},
		},
	}

	for _, tt := range tests {
		tf := func(t *testing.T) {
			deviceID, err := getSignOffDataFromClaims(context.Background(), client, tt.args.sod)
			if tt.wantErr == nil {
				require.NoError(t, err)
				require.Equal(t, tt.want.deviceID, deviceID)
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

package grpc

import (
	"testing"
	"time"

	oauthsigner "github.com/plgd-dev/hub/v2/m2m-oauth-server/oauthSigner"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/pb"
	"github.com/stretchr/testify/require"
)

func TestGetExpirationTime(t *testing.T) {
	now := time.Now()
	type args struct {
		clientCfg *oauthsigner.Client
		tokenReq  tokenRequest
	}
	tests := []struct {
		name string
		args args
		want time.Time
	}{
		{
			name: "client access token lifetime = 0h, token expiration = now + 1h",
			args: args{
				clientCfg: &oauthsigner.Client{
					AccessTokenLifetime: 0,
				},
				tokenReq: tokenRequest{
					CreateTokenRequest: &pb.CreateTokenRequest{
						Expiration: now.Unix() + int64(time.Hour.Seconds()),
					},
					issuedAt: now,
				},
			},
			want: now.Add(time.Hour),
		},
		{
			name: "client access token lifetime = 1h, token expiration = 0",
			args: args{
				clientCfg: &oauthsigner.Client{
					AccessTokenLifetime: time.Hour,
				},
				tokenReq: tokenRequest{
					CreateTokenRequest: &pb.CreateTokenRequest{},
					issuedAt:           now,
				},
			},
			want: now.Add(time.Hour),
		},
		{
			name: "client access token lifetime = 1h, token expiration = now + 1h",
			args: args{
				clientCfg: &oauthsigner.Client{
					AccessTokenLifetime: time.Hour,
				},
				tokenReq: tokenRequest{
					CreateTokenRequest: &pb.CreateTokenRequest{
						Expiration: now.Unix() + int64(time.Hour.Seconds()),
					},
					issuedAt: now,
				},
			},
			want: now.Add(time.Hour),
		},
		{
			name: "client access token lifetime = 1h, token expiration = now + 2h",
			args: args{
				clientCfg: &oauthsigner.Client{
					AccessTokenLifetime: time.Hour,
				},
				tokenReq: tokenRequest{
					CreateTokenRequest: &pb.CreateTokenRequest{
						Expiration: now.Unix() + int64(time.Hour.Seconds()*2),
					},
					issuedAt: now,
				},
			},
			want: now.Add(time.Hour),
		},
		{
			name: "client access token lifetime = 0h, token expiration = now - 2h",
			args: args{
				clientCfg: &oauthsigner.Client{},
				tokenReq: tokenRequest{
					CreateTokenRequest: &pb.CreateTokenRequest{
						Expiration: now.Unix() - int64(time.Hour.Seconds()*2),
					},
					issuedAt: now,
				},
			},
			want: time.Time{},
		},
		{
			name: "client access token lifetime = 0h, token expiration = 0",
			args: args{
				clientCfg: &oauthsigner.Client{},
				tokenReq: tokenRequest{
					CreateTokenRequest: &pb.CreateTokenRequest{},
					issuedAt:           now,
				},
			},
			want: time.Time{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getExpirationTime(tt.args.clientCfg, tt.args.tokenReq)
			require.Equal(t, tt.want.Unix(), got.Unix())
		})
	}
}

package service

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestParseTokenMD(t *testing.T) {
	type args struct {
		token      string
		ownerClaim string
	}
	type want struct {
		owner   string
		subject string
	}
	tests := []struct {
		name    string
		args    args
		want    want
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				token: config.CreateJwtToken(t, jwt.MapClaims{
					"sub":   "user",
					"owner": "owner",
				}),
				ownerClaim: "owner",
			},
			want: want{
				owner:   "owner",
				subject: "user",
			},
		},
		{
			name:    "missing token",
			args:    args{},
			wantErr: true,
		},
		{
			name: "invalid token",
			args: args{
				token: "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid owner claim",
			args: args{
				token: config.CreateJwtToken(t, jwt.MapClaims{
					"sub":   "user",
					"owner": 42,
				}),
				ownerClaim: "owner",
			},
			wantErr: true,
		},
		{
			name: "missing owner claim",
			args: args{
				token: config.CreateJwtToken(t, jwt.MapClaims{
					"sub": "user",
				}),
			},
			wantErr: true,
		},
		{
			name: "invalid subject claim",
			args: args{
				token: config.CreateJwtToken(t, jwt.MapClaims{
					"sub":   42,
					"owner": "owner",
				}),
				ownerClaim: "owner",
			},
			wantErr: true,
		},
		{
			name: "missing subject claim",
			args: args{
				token: config.CreateJwtToken(t, jwt.MapClaims{
					"owner": "owner",
				}),
				ownerClaim: "owner",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.args.token != "" {
				ctx = grpc.CtxWithIncomingToken(ctx, tt.args.token)
			}
			owner, subject, err := parseTokenMD(ctx, tt.args.ownerClaim)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want.owner, owner)
			require.Equal(t, tt.want.subject, subject)
		})
	}
}

package grpc_test

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestParseOwnerFromJwtToken(t *testing.T) {
	type args struct {
		ownerClaim string
		token      string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				ownerClaim: "sub",
				token: config.CreateJwtToken(t, jwt.MapClaims{
					"sub": "user",
				}),
			},
			want: "user",
		},
		{
			name: "invalid token",
			args: args{
				ownerClaim: "sub",
				token:      "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid owner claim",
			args: args{
				ownerClaim: "sub",
				token: config.CreateJwtToken(t, jwt.MapClaims{
					"sub": 42,
				}),
			},
			wantErr: true,
		},
		{
			name: "missing owner claim",
			args: args{
				ownerClaim: "sub",
				token:      config.CreateJwtToken(t, jwt.MapClaims{}),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := grpc.ParseOwnerFromJwtToken(tt.args.ownerClaim, tt.args.token)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestOwnerFromTokenMD(t *testing.T) {
	type args struct {
		ownerClaim string
		token      string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				ownerClaim: "sub",
				token: config.CreateJwtToken(t, jwt.MapClaims{
					"sub": "user",
				}),
			},
			want: "user",
		},
		{
			name: "missing token",
			args: args{
				ownerClaim: "sub",
			},
			wantErr: true,
		},
		{
			name: "missing owner claim",
			args: args{
				ownerClaim: "sub",
				token:      config.CreateJwtToken(t, jwt.MapClaims{}),
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
			got, err := grpc.OwnerFromTokenMD(ctx, tt.args.ownerClaim)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSubjectFromTokenMD(t *testing.T) {
	type args struct {
		token string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				token: config.CreateJwtToken(t, jwt.MapClaims{
					"sub": "user",
				}),
			},
			want: "user",
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
			name: "missing subject claim",
			args: args{
				token: config.CreateJwtToken(t, jwt.MapClaims{}),
			},
			wantErr: true,
		},
		{
			name: "invalid subject claim",
			args: args{
				token: config.CreateJwtToken(t, jwt.MapClaims{
					"sub": 42,
				}),
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
			got, err := grpc.SubjectFromTokenMD(ctx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

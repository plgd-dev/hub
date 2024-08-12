package http_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/plgd-dev/go-coap/v3/message"
	grpcPb "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	oauthsigner "github.com/plgd-dev/hub/v2/m2m-oauth-server/oauthSigner"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/pb"
	m2mOauthServerTest "github.com/plgd-dev/hub/v2/m2m-oauth-server/test"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/uri"
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	testHttp "github.com/plgd-dev/hub/v2/test/http"
	testOAuthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	testService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func createTokens(ctx context.Context, t *testing.T, createTokens []*pb.CreateTokenRequest) []string {
	accessTokensIDs := make([]string, 0, len(createTokens))
	for _, createToken := range createTokens {
		data, err := testHttp.GetContentData(&grpcPb.Content{
			ContentType: message.AppOcfCbor.String(),
			Data:        test.EncodeToCbor(t, createToken),
		}, message.AppJSON.String())
		require.NoError(t, err)
		rb := testHttp.NewRequest(http.MethodPost, m2mOauthServerTest.HTTPURI(uri.Tokens), bytes.NewReader(data))
		rb = rb.ContentType(message.AppOcfCbor.String())
		resp := testHttp.Do(t, rb.Build(ctx, t))
		defer func() {
			_ = resp.Body.Close()
		}()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var got pb.CreateTokenResponse
		err = pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &got)
		require.NoError(t, err)
		claims, err := jwt.ParseToken(got.GetAccessToken())
		require.NoError(t, err)
		name, err := claims.GetName()
		require.NoError(t, err)
		require.Equal(t, createToken.GetTokenName(), name)
		id, err := claims.GetID()
		require.NoError(t, err)
		accessTokensIDs = append(accessTokensIDs, id)
	}
	return accessTokensIDs
}

func blacklistTokens(ctx context.Context, t *testing.T, tokenIDs []string, token string) {
	rb := testHttp.NewRequest(http.MethodDelete, m2mOauthServerTest.HTTPURI(uri.Tokens), nil).AuthToken(token)
	rb.AddQuery(uri.IDFilterQuery, tokenIDs...)
	rb = rb.ContentType(message.AppOcfCbor.String())
	resp := testHttp.Do(t, rb.Build(ctx, t))
	defer func() {
		_ = resp.Body.Close()
	}()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetTokens(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()
	tearDown := testService.SetUp(ctx, t)
	defer tearDown()

	token := testOAuthTest.GetDefaultAccessToken(t)
	tokens := []*pb.CreateTokenRequest{
		{
			ClientId:     m2mOauthServerTest.ServiceOAuthClient.ID,
			ClientSecret: m2mOauthServerTest.GetSecret(t, m2mOauthServerTest.ServiceOAuthClient.ID),
			GrantType:    string(oauthsigner.GrantTypeClientCredentials),
			TokenName:    "service token",
		},
		{
			ClientId:            m2mOauthServerTest.JWTPrivateKeyOAuthClient.ID,
			GrantType:           string(oauthsigner.GrantTypeClientCredentials),
			ClientAssertionType: string(uri.ClientAssertionTypeJWT),
			ClientAssertion:     token,
		},
	}
	blacklistedTokens := []*pb.CreateTokenRequest{
		{
			ClientId:     m2mOauthServerTest.ServiceOAuthClient.ID,
			ClientSecret: m2mOauthServerTest.GetSecret(t, m2mOauthServerTest.ServiceOAuthClient.ID),
			GrantType:    string(oauthsigner.GrantTypeClientCredentials),
			TokenName:    "service token blacklisted",
		},
	}
	tokenIDs := createTokens(ctx, t, tokens)
	blacklistTokenIDs := createTokens(ctx, t, blacklistedTokens)

	blacklistTokens(ctx, t, blacklistTokenIDs, token)
	claims, err := jwt.ParseToken(token)
	require.NoError(t, err)

	type args struct {
		req   *pb.GetTokensRequest
		token string
	}
	tests := []struct {
		name         string
		args         args
		wantHTTPCode int
		want         map[string]*pb.Token
		wantErr      bool
	}{
		{
			name: "get all tokens included blacklisted",
			args: args{
				req: &pb.GetTokensRequest{
					IncludeBlacklisted: true,
				},
				token: token,
			},
			wantHTTPCode: http.StatusOK,
			want: map[string]*pb.Token{
				tokenIDs[0]: {
					ClientId: tokens[0].GetClientId(),
					Id:       tokenIDs[0],
					Name:     tokens[0].GetTokenName(),
				},
				tokenIDs[1]: {
					ClientId: tokens[1].GetClientId(),
					Id:       tokenIDs[1],
					Name:     tokens[1].GetTokenName(),
					OriginalTokenClaims: func() *structpb.Value {
						v, err2 := structpb.NewValue(map[string]interface{}(claims))
						require.NoError(t, err2)
						return v
					}(),
				},
				blacklistTokenIDs[0]: {
					ClientId: blacklistedTokens[0].GetClientId(),
					Id:       blacklistTokenIDs[0],
					Name:     blacklistedTokens[0].GetTokenName(),
					Blacklisted: &pb.Token_BlackListed{
						Flag: true,
					},
				},
			},
		},
		{
			name: "get all tokens excluded blacklisted",
			args: args{
				req:   &pb.GetTokensRequest{},
				token: token,
			},
			wantHTTPCode: http.StatusOK,
			want: map[string]*pb.Token{
				tokenIDs[0]: {
					ClientId: tokens[0].GetClientId(),
					Id:       tokenIDs[0],
					Name:     tokens[0].GetTokenName(),
				},
				tokenIDs[1]: {
					ClientId: tokens[1].GetClientId(),
					Id:       tokenIDs[1],
					Name:     tokens[1].GetTokenName(),
					OriginalTokenClaims: func() *structpb.Value {
						v, err2 := structpb.NewValue(map[string]interface{}(claims))
						require.NoError(t, err2)
						return v
					}(),
				},
			},
		},
		{
			name: "get certain tokens with excluded blacklisted",
			args: args{
				req: &pb.GetTokensRequest{
					IdFilter: []string{tokenIDs[1], blacklistTokenIDs[0]},
				},
				token: token,
			},
			wantHTTPCode: http.StatusOK,
			want: map[string]*pb.Token{
				tokenIDs[1]: {
					ClientId: tokens[1].GetClientId(),
					Id:       tokenIDs[1],
					Name:     tokens[1].GetTokenName(),
					OriginalTokenClaims: func() *structpb.Value {
						v, err2 := structpb.NewValue(map[string]interface{}(claims))
						require.NoError(t, err2)
						return v
					}(),
				},
			},
		},
		{
			name: "get certain tokens with included blacklisted",
			args: args{
				req: &pb.GetTokensRequest{
					IdFilter:           []string{tokenIDs[1], blacklistTokenIDs[0]},
					IncludeBlacklisted: true,
				},
				token: token,
			},
			wantHTTPCode: http.StatusOK,
			want: map[string]*pb.Token{
				tokenIDs[1]: {
					ClientId: tokens[1].GetClientId(),
					Id:       tokenIDs[1],
					Name:     tokens[1].GetTokenName(),
					OriginalTokenClaims: func() *structpb.Value {
						v, err2 := structpb.NewValue(map[string]interface{}(claims))
						require.NoError(t, err2)
						return v
					}(),
				},
				blacklistTokenIDs[0]: {
					ClientId: blacklistedTokens[0].GetClientId(),
					Id:       blacklistTokenIDs[0],
					Name:     blacklistedTokens[0].GetTokenName(),
					Blacklisted: &pb.Token_BlackListed{
						Flag: true,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := testHttp.NewRequest(http.MethodGet, m2mOauthServerTest.HTTPURI(uri.Tokens), nil)
			if tt.args.token != "" {
				rb = rb.AuthToken(tt.args.token)
			}
			if tt.args.req.GetIncludeBlacklisted() {
				rb = rb.AddQuery("includeBlacklisted", "true")
			}
			if len(tt.args.req.GetIdFilter()) > 0 {
				rb = rb.AddQuery(uri.IDFilterQuery, tt.args.req.GetIdFilter()...)
			}
			rb = rb.ContentType(message.AppOcfCbor.String())
			resp := testHttp.Do(t, rb.Build(ctx, t))
			defer func() {
				_ = resp.Body.Close()
			}()
			require.Equal(t, tt.wantHTTPCode, resp.StatusCode)

			var got []*pb.Token
			for {
				var gotToken pb.Token
				err := pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &gotToken)
				if errors.Is(err, io.EOF) {
					break
				}
				require.NoError(t, err)
				got = append(got, &gotToken)
			}
			require.Len(t, got, len(tt.want))
			for _, gotToken := range got {
				want, ok := tt.want[gotToken.GetId()]
				require.True(t, ok)
				require.Equal(t, want.GetClientId(), gotToken.GetClientId())
				require.Equal(t, want.GetId(), gotToken.GetId())
				require.Equal(t, want.GetName(), gotToken.GetName())
				if want.GetOriginalTokenClaims() != nil {
					test.CheckProtobufs(t, want.GetOriginalTokenClaims(), gotToken.GetOriginalTokenClaims(), test.RequireToCheckFunc(require.Equal))
				}
				require.Equal(t, want.GetBlacklisted().GetFlag(), gotToken.GetBlacklisted().GetFlag())
			}
		})
	}
}

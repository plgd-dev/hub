package grpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwk"
	oauthsigner "github.com/plgd-dev/hub/v2/m2m-oauth-server/oauthSigner"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/pb"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/store"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// M2MOAuthServiceServer handles incoming requests.
type M2MOAuthServiceServer struct {
	pb.UnimplementedM2MOAuthServiceServer

	signer *oauthsigner.OAuthSigner
	store  store.Store
	logger log.Logger
}

func NewM2MOAuthServerServer(store store.Store, signer *oauthsigner.OAuthSigner, logger log.Logger) *M2MOAuthServiceServer {
	return &M2MOAuthServiceServer{
		store:  store,
		logger: logger,
		signer: signer,
	}
}

func (s *M2MOAuthServiceServer) getOwner(ctx context.Context) (string, error) {
	ownerFromToken, err := pkgGrpc.OwnerFromTokenMD(ctx, s.signer.GetOwnerClaim())
	if err != nil {
		return "", err
	}
	return ownerFromToken, nil
}

func getGRPCErrorCode(err error) codes.Code {
	if errors.Is(err, store.ErrInvalidArgument) {
		return codes.InvalidArgument
	}
	return codes.Internal
}

func errCannotCreateConfiguration(err error) error {
	return fmt.Errorf("cannot get configuration: %w", err)
}

func errCannotCreateToken(err error) error {
	return fmt.Errorf("cannot create token: %w", err)
}

func (s *M2MOAuthServiceServer) CreateToken(ctx context.Context, req *pb.CreateTokenRequest) (*pb.CreateTokenResponse, error) {
	tokenReq := tokenRequest{
		tokenType:          oauthsigner.AccessTokenType_JWT,
		issuedAt:           time.Now(),
		CreateTokenRequest: req,
		issuer:             s.signer.GetAuthority(),
	}
	clientCfg := s.signer.GetClients().Find(tokenReq.GetClientId())
	if clientCfg == nil {
		return nil, status.Errorf(codes.Unauthenticated, "%v", errCannotCreateToken(fmt.Errorf("client(%v) not found", tokenReq.GetClientId())))
	}
	tokenReq.owner = clientCfg.Owner
	if err := s.validateTokenRequest(ctx, clientCfg, &tokenReq); err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "%v", errCannotCreateToken(fmt.Errorf("invalid request: %w", err)))
	}
	var originalTokenClaims *structpb.Value
	if len(tokenReq.originalTokenClaims) > 0 {
		var err error
		originalTokenClaims, err = structpb.NewValue(map[string]interface{}(tokenReq.originalTokenClaims))
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "%v", errCannotCreateToken(fmt.Errorf("cannot convert original token claims: %w", err)))
		}
	}
	tokenReq.scopes = strings.Join(req.GetScope(), " ")
	tokenReq.deviceIDClaim = s.signer.GetDeviceIDClaim()
	tokenReq.ownerClaim = s.signer.GetOwnerClaim()
	tokenReq.id = uuid.NewString()
	tokenReq.expiration = getExpirationTime(clientCfg, tokenReq)
	tokenReq.subject = getSubject(clientCfg, tokenReq)
	accessToken, err := s.generateAccessToken(
		clientCfg,
		tokenReq)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", errCannotCreateToken(err))
	}

	token, err := s.store.CreateToken(ctx, tokenReq.owner, &pb.Token{
		Id:                  tokenReq.id,
		Name:                tokenReq.GetTokenName(),
		Owner:               tokenReq.owner,
		IssuedAt:            tokenReq.issuedAt.Unix(),
		Audience:            tokenReq.GetAudience(),
		Scope:               tokenReq.GetScope(),
		Expiration:          pkgTime.UnixSec(tokenReq.expiration),
		ClientId:            tokenReq.GetClientId(),
		OriginalTokenClaims: originalTokenClaims,
		Subject:             tokenReq.subject,
	})
	if err != nil {
		return nil, status.Errorf(getGRPCErrorCode(err), "%v", errCannotCreateConfiguration(err))
	}
	var expiresIn int64
	if !tokenReq.expiration.IsZero() {
		expiresIn = int64(time.Until(tokenReq.expiration).Seconds())
	}
	return &pb.CreateTokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		Scope:       token.GetScope(),
	}, nil
}

func errCannotGetTokens(err error) error {
	return fmt.Errorf("cannot get tokens: %w", err)
}

func (s *M2MOAuthServiceServer) GetTokens(req *pb.GetTokensRequest, srv pb.M2MOAuthService_GetTokensServer) error {
	owner, err := s.getOwner(srv.Context())
	if err != nil {
		return err
	}
	err = s.store.GetTokens(srv.Context(), owner, req, func(v *pb.Token) error {
		return srv.Send(v)
	})
	if err != nil {
		return status.Errorf(getGRPCErrorCode(err), "%v", errCannotGetTokens(err))
	}
	return nil
}

func errCannotDeleteTokens(err error) error {
	return fmt.Errorf("cannot blacklist tokens: %w", err)
}

func (s *M2MOAuthServiceServer) DeleteTokens(ctx context.Context, req *pb.DeleteTokensRequest) (*pb.DeleteTokensResponse, error) {
	owner, err := s.getOwner(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := s.store.DeleteTokens(ctx, owner, req)
	if err != nil {
		return nil, status.Errorf(getGRPCErrorCode(err), "%v", errCannotDeleteTokens(err))
	}
	return resp, nil
}

func (s *M2MOAuthServiceServer) GetJWK() jwk.Key {
	return s.signer.GetJWK()
}

func (s *M2MOAuthServiceServer) GetDomain() string {
	return s.signer.GetDomain()
}

package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/plgd-dev/go-coap/v3/message"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/hub/v2/coap-gateway/coapconv"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/kit/v2/codec/cbor"
)

type CoapRefreshTokenReq struct {
	DeviceID              string `json:"di"`
	UserID                string `json:"uid"`
	RefreshToken          string `json:"refreshtoken"`
	AuthorizationProvider string `json:"authprovider"`
}

type CoapRefreshTokenResp struct {
	AccessToken  string `json:"accesstoken"`
	RefreshToken string `json:"refreshtoken"`
	ExpiresIn    int64  `json:"expiresin"`
}

// Get data for sign in response
func getRefreshTokenContent(token *oauth2.Token, expiresIn int64, options message.Options) (message.MediaType, []byte, error) {
	coapResp := CoapRefreshTokenResp{
		RefreshToken: token.RefreshToken,
		AccessToken:  token.AccessToken.String(),
		ExpiresIn:    expiresIn,
	}

	accept := coapconv.GetAccept(options)
	encode, err := coapconv.GetEncoder(accept)
	if err != nil {
		return 0, nil, err
	}
	out, err := encode(coapResp)
	if err != nil {
		return 0, nil, err
	}

	return accept, out, nil
}

func validateRefreshToken(req CoapRefreshTokenReq) error {
	if req.DeviceID == "" {
		return errors.New("cannot refresh token: invalid deviceID")
	}
	if req.RefreshToken == "" {
		return errors.New("cannot refresh token: invalid refreshToken")
	}
	if req.UserID == "" {
		return errors.New("cannot refresh token: invalid userId")
	}
	return nil
}

func getProviders(client *session, authorizationProvider string) (map[string]*oauth2.PlgdProvider, error) {
	if authorizationProvider == "" {
		return client.server.providers, nil
	}
	provider, ok := client.server.providers[authorizationProvider]
	if !ok {
		return nil, fmt.Errorf("unknown authorization provider('%v')", authorizationProvider)
	}
	providers := make(map[string]*oauth2.PlgdProvider)
	providers[authorizationProvider] = provider
	return providers, nil
}

func validUntilToExpiresIn(validUntil time.Time) int64 {
	if validUntil.IsZero() {
		return -1
	}
	return int64(time.Until(validUntil).Seconds())
}

func updateClient(client *session, deviceID, owner, accessToken string, validUntil time.Time) {
	if _, err := client.GetAuthorizationContext(); err != nil {
		return
	}
	authCtx := authorizationContext{
		DeviceID:    deviceID,
		UserID:      owner,
		AccessToken: accessToken,
		Expire:      validUntil,
	}
	client.SetAuthorizationContext(&authCtx)

	setExpirationClientCache(client.server.expirationClientCache, deviceID, client, validUntil)
}

func getRefreshTokenDataFromClaims(ctx context.Context, client *session, accessToken string, req CoapRefreshTokenReq) (string, string, error) {
	claim, err := client.ValidateToken(ctx, accessToken)
	if err != nil {
		return "", "", err
	}

	deviceID, err := client.server.VerifyAndResolveDeviceID(client.tlsDeviceID, req.DeviceID, claim)
	if err != nil {
		return "", "", err
	}
	owner, err := claim.GetOwner(client.server.config.APIs.COAP.Authorization.OwnerClaim)
	if err != nil {
		return "", "", err
	}
	if owner == "" {
		owner = req.UserID
	}
	if owner == "" {
		return "", "", errors.New("cannot determine owner")
	}
	return deviceID, owner, nil
}

func refreshTokenPostHandler(req *mux.Message, client *session) (*pool.Message, error) {
	const fmtErr = "cannot handle refresh token for %v: %w"

	var refreshToken CoapRefreshTokenReq
	err := cbor.ReadFrom(req.Body(), &refreshToken)
	if err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, "%w", fmt.Errorf(fmtErr, "unknown", err))
	}

	err = validateRefreshToken(refreshToken)
	if err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, "%w", fmt.Errorf(fmtErr, refreshToken.DeviceID, err))
	}

	// use provider for request
	providers, err := getProviders(client, refreshToken.AuthorizationProvider)
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, "%w", fmt.Errorf(fmtErr, refreshToken.DeviceID, err))
	}

	token, err := client.refreshCache.Execute(req.Context(), providers, client.server.taskQueue, refreshToken.RefreshToken, client.getLogger())
	if err != nil {
		// When OAuth server is not accessible, then return 503 Service Unavailable. If real error occurs them http code is mapped to code.
		return nil, statusErrorf(coapCodes.ServiceUnavailable, "%w", fmt.Errorf(fmtErr, refreshToken.DeviceID, err))
	}

	if token.RefreshToken == "" {
		return nil, statusErrorf(coapCodes.Unauthorized, "%w", fmt.Errorf(fmtErr, refreshToken.DeviceID, errors.New("refresh didn't return a refresh token")))
	}

	deviceID, owner, err := getRefreshTokenDataFromClaims(req.Context(), client, token.AccessToken.String(), refreshToken)
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, "%w", fmt.Errorf(fmtErr, refreshToken.DeviceID, err))
	}

	ctx := kitNetGrpc.CtxWithIncomingToken(kitNetGrpc.CtxWithToken(req.Context(), token.AccessToken.String()), token.AccessToken.String())
	ok, err := client.server.ownerCache.OwnsDevice(ctx, deviceID)
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, "%w", fmt.Errorf(fmtErr, deviceID, fmt.Errorf("cannot check owning: %w", err)))
	}
	if !ok {
		return nil, statusErrorf(coapCodes.Unauthorized, "%w", fmt.Errorf(fmtErr, deviceID, errors.New("device is not registered")))
	}

	expire, ok := ValidUntil(token.Expiry)
	if !ok {
		return nil, statusErrorf(coapCodes.Unauthorized, "%w", fmt.Errorf(fmtErr, deviceID, errors.New("expired access token")))
	}

	validUntil := pkgTime.Unix(0, expire)
	expiresIn := validUntilToExpiresIn(validUntil)

	accept, out, err := getRefreshTokenContent(token, expiresIn, req.Options())
	if err != nil {
		return nil, statusErrorf(coapCodes.ServiceUnavailable, "%w", fmt.Errorf(fmtErr, deviceID, err))
	}

	updateClient(client, deviceID, owner, token.AccessToken.String(), validUntil)

	return client.createResponse(coapCodes.Changed, req.Token(), accept, out), nil
}

// RefreshToken
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.tokenrefresh.swagger.json
func refreshTokenHandler(req *mux.Message, client *session) (*pool.Message, error) {
	switch req.Code() {
	case coapCodes.POST:
		return refreshTokenPostHandler(req, client)
	default:
		return nil, statusErrorf(coapCodes.NotFound, "unsupported method %v", req.Code())
	}
}

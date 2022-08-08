package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	"github.com/plgd-dev/hub/v2/coap-gateway/coapconv"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/kit/v2/codec/cbor"
)

type CoapRefreshTokenReq struct {
	DeviceID     string `json:"di"`
	UserID       string `json:"uid"`
	RefreshToken string `json:"refreshtoken"`
}

type CoapRefreshTokenResp struct {
	ExpiresIn    int64  `json:"expiresin"`
	AccessToken  string `json:"accesstoken"`
	RefreshToken string `json:"refreshtoken"`
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

func validUntilToExpiresIn(validUntil time.Time) int64 {
	if validUntil.IsZero() {
		return -1
	}
	return int64(time.Until(validUntil).Seconds())
}

func updateClient(client *Client, deviceID, owner, accessToken string, validUntil time.Time, jwtClaims jwt.Claims) {
	if _, err := client.GetAuthorizationContext(); err != nil {
		return
	}
	authCtx := authorizationContext{
		DeviceID:    deviceID,
		UserID:      owner,
		AccessToken: accessToken,
		Expire:      validUntil,
		JWTClaims:   jwtClaims,
	}
	client.SetAuthorizationContext(&authCtx)

	setExpirationClientCache(client.server.expirationClientCache, deviceID, client, validUntil)
}

func refreshTokenPostHandler(req *mux.Message, client *Client) (*pool.Message, error) {
	const fmtErr = "cannot handle refresh token for %v: %w"

	var refreshToken CoapRefreshTokenReq
	err := cbor.ReadFrom(req.Body, &refreshToken)
	if err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, "%w", fmt.Errorf(fmtErr, "unknown", err))
	}

	err = validateRefreshToken(refreshToken)
	if err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, "%w", fmt.Errorf(fmtErr, refreshToken.DeviceID, err))
	}

	token, err := client.refreshCache.Execute(req.Context, client.server.providers, client.server.taskQueue, refreshToken.RefreshToken, client.getLogger())
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, "%w", fmt.Errorf(fmtErr, refreshToken.DeviceID, err))
	}

	if token.RefreshToken == "" {
		return nil, statusErrorf(coapCodes.Unauthorized, "%w", fmt.Errorf(fmtErr, refreshToken.DeviceID, fmt.Errorf("refresh didn't return a refresh token")))
	}

	claim, err := client.ValidateToken(req.Context, token.AccessToken.String())
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, "%w", fmt.Errorf(fmtErr, refreshToken.DeviceID, err))
	}

	err = client.server.VerifyDeviceID(client.tlsDeviceID, claim)
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, "%w", fmt.Errorf(fmtErr, refreshToken.DeviceID, err))
	}
	deviceID := client.ResolveDeviceID(claim, refreshToken.DeviceID)
	ctx := kitNetGrpc.CtxWithIncomingToken(kitNetGrpc.CtxWithToken(req.Context, token.AccessToken.String()), token.AccessToken.String())
	ok, err := client.server.ownerCache.OwnsDevice(ctx, deviceID)
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, "%w", fmt.Errorf(fmtErr, deviceID, fmt.Errorf("cannot check owning: %w", err)))
	}
	if !ok {
		return nil, statusErrorf(coapCodes.Unauthorized, "%w", fmt.Errorf(fmtErr, deviceID, fmt.Errorf("device is not registered")))
	}

	owner := claim.Owner(client.server.config.APIs.COAP.Authorization.OwnerClaim)
	if owner == "" {
		owner = refreshToken.UserID
	}
	if owner == "" {
		return nil, statusErrorf(coapCodes.Unauthorized, "%w", fmt.Errorf(fmtErr, deviceID, fmt.Errorf("cannot determine owner")))
	}

	expire, ok := ValidUntil(token.Expiry)
	if !ok {
		return nil, statusErrorf(coapCodes.Unauthorized, "%w", fmt.Errorf(fmtErr, deviceID, fmt.Errorf("expired access token")))
	}

	validUntil := pkgTime.Unix(0, expire)

	expiresIn := validUntilToExpiresIn(validUntil)
	accept, out, err := getRefreshTokenContent(token, expiresIn, req.Options)
	if err != nil {
		return nil, statusErrorf(coapCodes.InternalServerError, "%w", fmt.Errorf(fmtErr, deviceID, err))
	}

	updateClient(client, deviceID, owner, token.AccessToken.String(), validUntil, claim)

	return client.createResponse(coapCodes.Changed, req.Token, accept, out), nil
}

// RefreshToken
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.tokenrefresh.swagger.json
func refreshTokenHandler(req *mux.Message, client *Client) (*pool.Message, error) {
	switch req.Code {
	case coapCodes.POST:
		return refreshTokenPostHandler(req, client)
	default:
		return nil, statusErrorf(coapCodes.NotFound, "unsupported method %v", req.Code)
	}
}

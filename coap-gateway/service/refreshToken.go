package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/hub/coap-gateway/coapconv"
	"github.com/plgd-dev/hub/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/pkg/security/oauth2"
	pkgTime "github.com/plgd-dev/hub/pkg/time"
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

/// Get data for sign in response
func getRefreshTokenContent(token oauth2.Token, expiresIn int64, options message.Options) (message.MediaType, []byte, error) {
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

func updateClient(client *Client, deviceID, owner, accessToken string, validUntil time.Time) {
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

	if validUntil.IsZero() {
		client.server.expirationClientCache.Set(deviceID, nil, time.Millisecond)
	} else {
		expiresIn := validUntilToExpiresIn(validUntil)
		client.server.expirationClientCache.Set(deviceID, client, time.Second*time.Duration(expiresIn))
	}
}

func refreshTokenPostHandler(req *mux.Message, client *Client) {
	const fmtErr = "cannot handle refresh token for %v: %w"
	logErrorAndCloseClient := func(err error, code coapCodes.Code) {
		client.logAndWriteErrorResponse(err, code, req.Token)
		if err := client.Close(); err != nil {
			log.Errorf("refresh token error: %w", err)
		}
	}

	var refreshToken CoapRefreshTokenReq
	err := cbor.ReadFrom(req.Body, &refreshToken)
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf(fmtErr, "unknown", err), coapCodes.BadRequest)
		return
	}

	err = validateRefreshToken(refreshToken)
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf(fmtErr, refreshToken.DeviceID, err), coapCodes.BadRequest)
		return
	}

	token, err := client.refreshCache.Execute(req.Context, client.server.providers, client.server.taskQueue, refreshToken.RefreshToken)
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf(fmtErr, refreshToken.DeviceID, err), coapCodes.Unauthorized)
		return
	}

	claim, err := client.ValidateToken(req.Context, token.AccessToken.String())
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf(fmtErr, refreshToken.DeviceID, err), coapCodes.Unauthorized)
		return
	}

	err = client.server.VerifyDeviceID(client.tlsDeviceID, claim)
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf(fmtErr, refreshToken.DeviceID, err), coapCodes.Unauthorized)
		return
	}
	deviceID := client.ResolveDeviceID(claim, refreshToken.DeviceID)
	ctx := kitNetGrpc.CtxWithIncomingToken(kitNetGrpc.CtxWithToken(req.Context, token.AccessToken.String()), token.AccessToken.String())
	ok, err := client.server.ownerCache.OwnsDevice(ctx, deviceID)
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf(fmtErr, deviceID, fmt.Errorf("cannot check owning: %w", err)), coapCodes.Unauthorized)
		return
	}
	if !ok {
		logErrorAndCloseClient(fmt.Errorf(fmtErr, deviceID, fmt.Errorf("device is not registered")), coapCodes.Unauthorized)
		return
	}

	owner := claim.Owner(client.server.config.APIs.COAP.Authorization.OwnerClaim)
	if owner == "" {
		owner = refreshToken.UserID
	}
	if owner == "" {
		logErrorAndCloseClient(fmt.Errorf(fmtErr, deviceID, fmt.Errorf("cannot determine owner")), coapCodes.Unauthorized)
		return
	}

	expire, ok := ValidUntil(token.Expiry)
	if !ok {
		logErrorAndCloseClient(fmt.Errorf(fmtErr, deviceID, fmt.Errorf("expired access token")), coapCodes.InternalServerError)
		return
	}

	validUntil := pkgTime.Unix(0, expire)

	expiresIn := validUntilToExpiresIn(validUntil)
	accept, out, err := getRefreshTokenContent(token, expiresIn, req.Options)
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf(fmtErr, deviceID, err), coapCodes.InternalServerError)
		return
	}

	updateClient(client, deviceID, owner, token.AccessToken.String(), validUntil)

	client.sendResponse(coapCodes.Changed, req.Token, accept, out)
}

// RefreshToken
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.tokenrefresh.swagger.json
func refreshTokenHandler(req *mux.Message, client *Client) {
	switch req.Code {
	case coapCodes.POST:
		refreshTokenPostHandler(req, client)
	default:
		client.logAndWriteErrorResponse(fmt.Errorf("forbidden request from %v", client.remoteAddrString()), coapCodes.Forbidden, req.Token)
	}
}

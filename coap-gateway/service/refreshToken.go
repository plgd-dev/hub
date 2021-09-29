package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/plgd-dev/cloud/coap-gateway/coapconv"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/security/oauth2"
	pkgTime "github.com/plgd-dev/cloud/pkg/time"
	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/kit/codec/cbor"
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

// the device doesn't provide an authorization provider during refresh token so we need to do try refresh against all providers in parallel
func refreshAccessToken(ctx context.Context, server *Service, refreshToken string) (*oauth2.Token, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	var mutex sync.Mutex
	var token *oauth2.Token
	var err error
	for name, p := range server.providers {
		wg.Add(1)
		provider := p
		errSubmit := server.taskQueue.Submit(func() {
			defer wg.Done()
			tokenTmp, errTmp := provider.Refresh(ctx, refreshToken)
			mutex.Lock()
			defer mutex.Unlock()
			if token == nil {
				token = tokenTmp
			}
			if err == nil {
				err = errTmp
				cancel()
			}
		})
		if errSubmit != nil {
			log.Errorf("cannot refresh token for provider %v: %w", name, errSubmit)
		}
	}
	wg.Wait()
	if token != nil {
		return token, nil
	}
	if err == nil {
		return nil, fmt.Errorf("invalid token")
	}
	return nil, err
}

func refreshTokenPostHandler(req *mux.Message, client *Client) {
	logErrorAndCloseClient := func(err error, code coapCodes.Code) {
		client.logAndWriteErrorResponse(err, code, req.Token)
		if err := client.Close(); err != nil {
			log.Errorf("refresh token error: %w", err)
		}
	}

	var refreshToken CoapRefreshTokenReq
	err := cbor.ReadFrom(req.Body, &refreshToken)
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle refresh token: %w", err), coapCodes.BadRequest)
		return
	}

	err = validateRefreshToken(refreshToken)
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle refresh token: %w", err), coapCodes.BadRequest)
		return
	}

	token, err := refreshAccessToken(req.Context, client.server, refreshToken.RefreshToken)
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle refresh token: %w", err), coapCodes.Unauthorized)
		return
	}

	claim, err := client.ValidateToken(req.Context, token.AccessToken.String())
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle refresh token: %w", err), coapCodes.Unauthorized)
		return
	}

	err = client.server.VerifyDeviceID(client.tlsDeviceID, claim)
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle refresh token: %w", err), coapCodes.Unauthorized)
		return
	}

	owner := token.Owner
	if owner == "" {
		owner = refreshToken.UserID
	}
	if owner == "" {
		logErrorAndCloseClient(fmt.Errorf("cannot refresh token: cannot determine owner"), coapCodes.Unauthorized)
		return
	}

	expire, ok := ValidUntil(token.Expiry)
	if !ok {
		logErrorAndCloseClient(fmt.Errorf("cannot handle refresh token: expired access token"), coapCodes.InternalServerError)
		return
	}

	validUntil := pkgTime.Unix(0, expire)
	deviceID := client.ResolveDeviceID(claim, refreshToken.DeviceID)
	expiresIn := validUntilToExpiresIn(validUntil)
	accept, out, err := getRefreshTokenContent(token, expiresIn, req.Options)
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle refresh token for %v: %w", deviceID, err), coapCodes.InternalServerError)
		return
	}

	if _, err := client.GetAuthorizationContext(); err == nil {
		authCtx := authorizationContext{
			DeviceID:    deviceID,
			UserID:      owner,
			AccessToken: token.AccessToken.String(),
			Expire:      validUntil,
		}
		client.SetAuthorizationContext(&authCtx)

		if validUntil.IsZero() {
			client.server.expirationClientCache.Set(deviceID, nil, time.Millisecond)
		} else {
			client.server.expirationClientCache.Set(deviceID, client, time.Second*time.Duration(expiresIn))
		}
	}

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

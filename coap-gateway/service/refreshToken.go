package service

import (
	"errors"
	"fmt"
	"time"

	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/coap-gateway/coapconv"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/kit/codec/cbor"
	"google.golang.org/grpc/status"
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

func validUntilToExpiresIn(v int64) int64 {
	if v == 0 {
		return -1
	}
	validUntil := time.Unix(0, v)
	return int64(time.Until(validUntil).Seconds())
}

func refreshTokenPostHandler(req *mux.Message, client *Client) {
	var refreshToken CoapRefreshTokenReq
	err := cbor.ReadFrom(req.Body, &refreshToken)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle refresh token: %w", err), coapCodes.BadRequest, req.Token)
		return
	}

	err = validateRefreshToken(refreshToken)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle refresh token: %w", err), coapCodes.BadRequest, req.Token)
		return
	}

	resp, err := client.server.asClient.RefreshToken(req.Context, &pbAS.RefreshTokenRequest{
		DeviceId:     refreshToken.DeviceID,
		UserId:       refreshToken.UserID,
		RefreshToken: refreshToken.RefreshToken,
	})
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle refresh token for %v: %w", refreshToken.DeviceID, err), coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapconv.Update), req.Token)
		return
	}

	coapResp := CoapRefreshTokenResp{
		RefreshToken: resp.RefreshToken,
		AccessToken:  resp.AccessToken,
		ExpiresIn:    validUntilToExpiresIn(resp.GetValidUntil()),
	}

	accept := coapconv.GetAccept(req.Options)
	encode, err := coapconv.GetEncoder(accept)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle refresh token for %v: %w", refreshToken.DeviceID, err), coapCodes.InternalServerError, req.Token)
		return
	}
	out, err := encode(coapResp)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle refresh token for %v: %w", refreshToken.DeviceID, err), coapCodes.InternalServerError, req.Token)
		return
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

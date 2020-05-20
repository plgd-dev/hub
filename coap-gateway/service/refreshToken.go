package service

import (
	"errors"
	"fmt"

	pbAS "github.com/go-ocf/cloud/authorization/pb"
	"github.com/go-ocf/cloud/coap-gateway/coapconv"
	"github.com/go-ocf/go-coap/v2/message"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/mux"
	"github.com/go-ocf/kit/codec/cbor"
	"github.com/go-ocf/kit/net/coap"
	"google.golang.org/grpc/status"
)

type CoapRefreshTokenReq struct {
	DeviceId     string `json:"di"`
	UserId       string `json:"uid"`
	RefreshToken string `json:"refreshtoken"`
}

type CoapRefreshTokenResp struct {
	ExpiresIn    int64  `json:"expiresin"`
	AccessToken  string `json:"accesstoken"`
	RefreshToken string `json:"refreshtoken"`
}

func validateRefreshToken(req CoapRefreshTokenReq) error {
	if req.DeviceId == "" {
		return errors.New("cannot refresh token: invalid deviceID")
	}
	if req.RefreshToken == "" {
		return errors.New("cannot refresh token: invalid refreshToken")
	}
	if req.UserId == "" {
		return errors.New("cannot refresh token: invalid userId")
	}
	return nil
}

func refreshTokenPostHandler(s mux.ResponseWriter, req *message.Message, client *Client) {
	var refreshToken CoapRefreshTokenReq
	err := cbor.Decode(req.Msg.Payload(), &refreshToken)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot handle refresh token: %v", err), s, client, coapCodes.BadRequest)
		return
	}

	err = validateRefreshToken(refreshToken)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot handle refresh token: %v", err), s, client, coapCodes.BadRequest)
		return
	}

	resp, err := client.server.asClient.RefreshToken(req.Ctx, &pbAS.RefreshTokenRequest{
		DeviceId:     refreshToken.DeviceId,
		UserId:       refreshToken.UserId,
		RefreshToken: refreshToken.RefreshToken,
	})
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot handle refresh token: %v", err), s, client, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.POST))
		return
	}

	coapResp := CoapRefreshTokenResp{
		RefreshToken: resp.RefreshToken,
		AccessToken:  resp.AccessToken,
		ExpiresIn:    resp.ExpiresIn,
	}

	accept := coap.GetAccept(req.Msg)
	encode, err := coap.GetEncoder(accept)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: %v", err), s, client, coapCodes.InternalServerError)
		return
	}
	out, err := encode(coapResp)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: %v", err), s, client, coapCodes.InternalServerError)
		return
	}
	sendResponse(s, client, coapCodes.Changed, accept, out)
}

// RefreshToken
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.tokenrefresh.swagger.json
func refreshTokenHandler(s mux.ResponseWriter, req *message.Message, client *Client) {
	switch req.Msg.Code() {
	case coapCodes.POST:
		refreshTokenPostHandler(s, req, client)
	default:
		logAndWriteErrorResponse(fmt.Errorf("Forbidden request from %v", req.Client.RemoteAddr()), s, client, coapCodes.Forbidden)
	}
}

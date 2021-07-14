package service

import (
	"fmt"

	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/coap-gateway/coapconv"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/kit/codec/cbor"
	"google.golang.org/grpc/status"
)

type CoapSignUpRequest struct {
	DeviceID                string `json:"di"`
	AuthorizationCode       string `json:"accesstoken"`
	AuthorizationProvider   string `json:"authprovider"`
	AuthorizationCodeLegacy string `json:"authcode"`
}

type CoapSignUpResponse struct {
	AccessToken  string `json:"accesstoken"`
	UserID       string `json:"uid"`
	RefreshToken string `json:"refreshtoken"`
	ExpiresIn    int64  `json:"expiresin"`
	RedirectURI  string `json:"redirecturi"`
}

// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.account.swagger.json
func signUpPostHandler(r *mux.Message, client *Client) {
	var signUp CoapSignUpRequest
	err := cbor.ReadFrom(r.Body, &signUp)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign up: %w", err), coapCodes.BadRequest, r.Token)
		return
	}

	// set AuthorizationCode from AuthorizationCodeLegacy
	if signUp.AuthorizationCode == "" {
		signUp.AuthorizationCode = signUp.AuthorizationCodeLegacy
	}

	response, err := client.server.asClient.SignUp(r.Context, &pbAS.SignUpRequest{
		DeviceId:              signUp.DeviceID,
		AuthorizationCode:     signUp.AuthorizationCode,
		AuthorizationProvider: signUp.AuthorizationProvider,
	})
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign up: %w", err), coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapconv.Update), r.Token)
		return
	}

	coapResponse := CoapSignUpResponse{
		AccessToken:  response.AccessToken,
		UserID:       response.UserId,
		RefreshToken: response.RefreshToken,
		ExpiresIn:    validUntilToExpiresIn(response.GetValidUntil()),
		RedirectURI:  response.RedirectUri,
	}

	accept := coapconv.GetAccept(r.Options)
	encode, err := coapconv.GetEncoder(accept)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign up: %w", err), coapCodes.InternalServerError, r.Token)
		return
	}
	out, err := encode(coapResponse)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign up: %w", err), coapCodes.InternalServerError, r.Token)
		return
	}

	client.sendResponse(coapCodes.Changed, r.Token, accept, out)
}

// Sign-up
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.account.swagger.json
func signUpHandler(r *mux.Message, client *Client) {
	switch r.Code {
	case coapCodes.POST:
		signUpPostHandler(r, client)
	case coapCodes.DELETE:
		signOffHandler(r, client)
	default:
		client.logAndWriteErrorResponse(fmt.Errorf("forbidden request from %v", client.remoteAddrString()), coapCodes.Forbidden, r.Token)
	}
}

package service

import (
	"context"
	"fmt"
	"net/url"

	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/coap-gateway/coapconv"
	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/kit/codec/cbor"
	"github.com/plgd-dev/kit/net/coap"
	"google.golang.org/grpc/status"
)

var (
	queryAccessToken = "accesstoken"
	queryDeviceID    = "di"
	queryUserID      = "uid" // optional because it is not defined in a current specification => it must be determined from the access token
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

func validateSignUp(req CoapSignUpRequest) error {
	if req.DeviceID == "" {
		return fmt.Errorf("cannot sign up to auth server: invalid deviceID")
	}
	if req.AuthorizationCode == "" {
		return fmt.Errorf("cannot sign up to auth server: invalid authorizationCode")
	}
	return nil
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
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign up: %w", err), coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.POST), r.Token)
		return
	}

	coapResponse := CoapSignUpResponse{
		AccessToken:  response.AccessToken,
		UserID:       response.UserId,
		RefreshToken: response.RefreshToken,
		ExpiresIn:    response.ExpiresIn,
		RedirectURI:  response.RedirectUri,
	}

	accept := coap.GetAccept(r.Options)
	encode, err := coap.GetEncoder(accept)
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

func validateSignOff(deviceID, accessToken string) error {
	if deviceID == "" {
		return fmt.Errorf("invalid '%v'", queryDeviceID)
	}
	if accessToken == "" {
		return fmt.Errorf("invalid '%v'", queryAccessToken)
	}
	return nil
}

// Sign-off
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.account.swagger.json
func signOffHandler(req *mux.Message, client *Client) {
	//from QUERY: di, accesstoken
	var deviceID string
	var accessToken string
	var userID string

	ctx, cancel := context.WithTimeout(client.server.ctx, client.server.RequestTimeout)
	defer cancel()

	queries, _ := req.Options.Queries()
	for _, query := range queries {
		values, err := url.ParseQuery(query)
		if err != nil {
			client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign off: %w", err), coapCodes.BadOption, req.Token)
			return
		}
		if di := values.Get(queryDeviceID); di != "" {
			deviceID = di
		}

		if at := values.Get(queryAccessToken); at != "" {
			accessToken = at
		}

		if uid := values.Get(queryUserID); uid != "" {
			userID = uid
		}
	}
	authCurrentCtx := client.loadAuthorizationContext()
	if userID == "" {
		userID = authCurrentCtx.UserID
	}
	if deviceID == "" {
		deviceID = authCurrentCtx.GetDeviceId()
	}

	err := validateSignOff(deviceID, accessToken)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign off for %v: %w", deviceID, err), coapCodes.BadRequest, req.Token)
		return
	}
	_, err = client.server.asClient.SignOff(ctx, &pbAS.SignOffRequest{
		DeviceId:    deviceID,
		UserId:      userID,
		AccessToken: accessToken,
	})
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign off for %v: %w", deviceID, err), coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.DELETE), req.Token)
		return
	}
	client.replaceAuthorizationContext(nil)
	client.sendResponse(coapCodes.Deleted, req.Token, message.TextPlain, nil)
}

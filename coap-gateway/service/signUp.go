package service

import (
	"fmt"
	"net/url"

	pbAS "github.com/go-ocf/cloud/authorization/pb"
	"github.com/go-ocf/cloud/coap-gateway/coapconv"
	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	"github.com/go-ocf/go-coap/v2/message"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/mux"
	"github.com/go-ocf/kit/codec/cbor"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/net/coap"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
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
	UserId       string `json:"uid"`
	RefreshToken string `json:"refreshtoken"`
	ExpiresIn    int64  `json:"expiresin"`
	RedirectUri  string `json:"redirecturi"`
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
func signUpPostHandler(w mux.ResponseWriter, r *mux.Message, client *Client) {
	var signUp CoapSignUpRequest
	err := cbor.ReadFrom(r.Body, &signUp)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot handle sign up: %v", err), client, coapCodes.BadRequest, r.Token)
		return
	}

	// set AuthorizationCode from AuthorizationCodeLegacy
	if signUp.AuthorizationCode == "" {
		signUp.AuthorizationCode = signUp.AuthorizationCodeLegacy
	}

	err = validateSignUp(signUp)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot handle sign up: %v", err), client, coapCodes.BadRequest, r.Token)
		return
	}

	response, err := client.server.asClient.SignUp(r.Context, &pbAS.SignUpRequest{
		DeviceId:              signUp.DeviceID,
		AuthorizationCode:     signUp.AuthorizationCode,
		AuthorizationProvider: signUp.AuthorizationProvider,
	})
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot handle sign up: %v", err), client, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.POST), r.Token)
		return
	}

	err = client.PublishCloudDeviceStatus(kitNetGrpc.CtxWithToken(r.Context, response.AccessToken), signUp.DeviceID, pbCQRS.AuthorizationContext{
		UserId:   response.UserId,
		DeviceId: signUp.DeviceID,
	})
	if err != nil {
		log.Errorf("cannot publish cloud device status: %v", err)
	}

	coapResponse := CoapSignUpResponse{
		AccessToken:  response.AccessToken,
		UserId:       response.UserId,
		RefreshToken: response.RefreshToken,
		ExpiresIn:    response.ExpiresIn,
		RedirectUri:  response.RedirectUri,
	}

	accept, err := r.Options.Accept()
	if err != nil {
		accept = message.AppOcfCbor
	}
	encode, err := coap.GetEncoder(accept)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot handle sign up: %v", err), client, coapCodes.InternalServerError, r.Token)
		return
	}
	out, err := encode(coapResponse)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot handle sign up: %v", err), client, coapCodes.InternalServerError, r.Token)
		return
	}

	sendResponse(client, coapCodes.Changed, r.Token, accept, out)
}

// Sign-up
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.account.swagger.json
func signUpHandler(w mux.ResponseWriter, r *mux.Message, client *Client) {
	switch r.Code {
	case coapCodes.POST:
		signUpPostHandler(w, r, client)
	case coapCodes.DELETE:
		signOffHandler(w, r, client)
	default:
		logAndWriteErrorResponse(fmt.Errorf("Forbidden request from %v", client.remoteAddrString()), client, coapCodes.Forbidden, r.Token)
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
func signOffHandler(s mux.ResponseWriter, req *mux.Message, client *Client) {
	//from QUERY: di, accesstoken
	var deviceID string
	var accessToken string
	var userID string

	queries, _ := req.Options.Queries()
	for _, query := range queries {
		values, err := url.ParseQuery(query)
		if err != nil {
			logAndWriteErrorResponse(fmt.Errorf("cannot handle sign off: %v", err), client, coapCodes.BadOption, req.Token)
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

	err := validateSignOff(deviceID, accessToken)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot handle sign off: %v", err), client, coapCodes.BadRequest, req.Token)
		return
	}
	_, err = client.server.asClient.SignOff(kitNetGrpc.CtxWithToken(req.Context, accessToken), &pbAS.SignOffRequest{
		DeviceId: deviceID,
		UserId:   userID,
	})
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot handle sign off: %v", err), client, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.DELETE), req.Token)
		return
	}
	client.storeAuthorizationContext(authCtx{})
	sendResponse(client, coapCodes.Deleted, req.Token, message.TextPlain, nil)
}

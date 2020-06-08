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
func signUpPostHandler(w mux.ResponseWriter, r *mux.Message, client *Client) {
	var signUp CoapSignUpRequest
	err := cbor.ReadFrom(r.Body, &signUp)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign up: %v", err), coapCodes.BadRequest, r.Token)
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
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign up: %v", err), coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.POST), r.Token)
		return
	}

	err = client.PublishCloudDeviceStatus(kitNetGrpc.CtxWithUserID(r.Context, response.UserId), signUp.DeviceID, pbCQRS.AuthorizationContext{
		DeviceId: signUp.DeviceID,
	})
	if err != nil {
		log.Errorf("cannot publish cloud device status: %v", err)
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
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign up: %v", err), coapCodes.InternalServerError, r.Token)
		return
	}
	out, err := encode(coapResponse)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign up: %v", err), coapCodes.InternalServerError, r.Token)
		return
	}

	client.sendResponse(coapCodes.Changed, r.Token, accept, out)
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
		client.logAndWriteErrorResponse(fmt.Errorf("Forbidden request from %v", client.remoteAddrString()), coapCodes.Forbidden, r.Token)
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
			client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign off: %v", err), coapCodes.BadOption, req.Token)
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
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign off: %v", err), coapCodes.BadRequest, req.Token)
		return
	}
	_, err = client.server.asClient.SignOff(req.Context, &pbAS.SignOffRequest{
		DeviceId: deviceID,
		UserId:   userID,
	})
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign off: %v", err), coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.DELETE), req.Token)
		return
	}
	client.storeAuthorizationContext(authCtx{})
	client.sendResponse(coapCodes.Deleted, req.Token, message.TextPlain, nil)
}

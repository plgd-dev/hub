package service

import (
	"fmt"
	"net/url"

	pbAS "github.com/go-ocf/cloud/authorization/pb"
	"github.com/go-ocf/cloud/coap-gateway/coapconv"
	gocoap "github.com/go-ocf/go-coap"
	coapCodes "github.com/go-ocf/go-coap/codes"
	"github.com/go-ocf/kit/codec/cbor"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/net/coap"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	"google.golang.org/grpc/status"
)

var (
	queryAccessToken = "accesstoken"
	queryDeviceID    = "di"
	queryUserID      = "uid" // optional because it is not defined in a current specification => it must be determined from the access token
)

type CoapSignUpRequest struct {
	DeviceId                string `json:"di"`
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
	if req.DeviceId == "" {
		return fmt.Errorf("cannot sign up to auth server: invalid deviceId")
	}
	if req.AuthorizationCode == "" {
		return fmt.Errorf("cannot sign up to auth server: invalid authorizationCode")
	}
	return nil
}

// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.account.swagger.json
func signUpPostHandler(s gocoap.ResponseWriter, req *gocoap.Request, client *Client) {
	var signUp CoapSignUpRequest
	err := cbor.Decode(req.Msg.Payload(), &signUp)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot handle sign up: %v", err), s, client, coapCodes.BadRequest)
		return
	}

	// set AuthorizationCode from AuthorizationCodeLegacy
	if signUp.AuthorizationCode == "" {
		signUp.AuthorizationCode = signUp.AuthorizationCodeLegacy
	}

	err = validateSignUp(signUp)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot handle sign up: %v", err), s, client, coapCodes.BadRequest)
		return
	}

	response, err := client.server.asClient.SignUp(req.Ctx, &pbAS.SignUpRequest{
		DeviceId:              signUp.DeviceId,
		AuthorizationCode:     signUp.AuthorizationCode,
		AuthorizationProvider: signUp.AuthorizationProvider,
	})
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot handle sign up: %v", err), s, client, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.POST))
		return
	}

	err = client.PublishCloudDeviceStatus(kitNetGrpc.CtxWithToken(req.Ctx, response.AccessToken), signUp.DeviceId, pbCQRS.AuthorizationContext{
		UserId:   response.UserId,
		DeviceId: signUp.DeviceId,
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

	accept := coap.GetAccept(req.Msg)
	encode, err := coap.GetEncoder(accept)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot handle sign up: %v", err), s, client, coapCodes.InternalServerError)
		return
	}
	out, err := encode(coapResponse)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot handle sign up: %v", err), s, client, coapCodes.InternalServerError)
		return
	}

	sendResponse(s, client, coapCodes.Changed, accept, out)
}

// Sign-up
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.account.swagger.json
func signUpHandler(s gocoap.ResponseWriter, req *gocoap.Request, client *Client) {
	switch req.Msg.Code() {
	case coapCodes.POST:
		signUpPostHandler(s, req, client)
	case coapCodes.DELETE:
		signOffHandler(s, req, client)
	default:
		logAndWriteErrorResponse(fmt.Errorf("Forbidden request from %v", req.Client.RemoteAddr()), s, client, coapCodes.Forbidden)
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
func signOffHandler(s gocoap.ResponseWriter, req *gocoap.Request, client *Client) {
	//from QUERY: di, accesstoken
	var deviceID string
	var accessToken string
	var userID string

	for _, q := range req.Msg.Options(gocoap.URIQuery) {
		var query string
		var ok bool
		if query, ok = q.(string); !ok {
			continue
		}

		values, err := url.ParseQuery(query)
		if err != nil {
			logAndWriteErrorResponse(fmt.Errorf("cannot handle sign off: %v", err), s, client, coapCodes.BadOption)
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
		logAndWriteErrorResponse(fmt.Errorf("cannot handle sign off: %v", err), s, client, coapCodes.BadRequest)
		return
	}
	_, err = client.server.asClient.SignOff(kitNetGrpc.CtxWithToken(req.Ctx, accessToken), &pbAS.SignOffRequest{
		DeviceId: deviceID,
		UserId:   userID,
	})
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot handle sign off: %v", err), s, client, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.DELETE))
		return
	}
	client.storeAuthorizationContext(authCtx{})
	sendResponse(s, client, coapCodes.Deleted, gocoap.TextPlain, nil)
}

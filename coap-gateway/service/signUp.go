package service

import (
	"fmt"

	"github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/coap-gateway/authorization"
	"github.com/plgd-dev/cloud/coap-gateway/coapconv"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	pkgTime "github.com/plgd-dev/cloud/pkg/time"
	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/kit/codec/cbor"
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

/// Check that all required request fields are set
func (request CoapSignUpRequest) checkOAuthRequest() error {
	if request.DeviceID == "" {
		return fmt.Errorf("invalid device id")
	}
	if request.AuthorizationCode == "" {
		return fmt.Errorf("invalid authorization code")
	}
	if request.AuthorizationProvider == "" {
		return fmt.Errorf("invalid authorization provider")
	}
	return nil
}

/// Get data for sign up response
func getSignUpContent(token *authorization.Token, validUntil int64, options message.Options) (message.MediaType, []byte, error) {
	resp := CoapSignUpResponse{
		AccessToken:  token.AccessToken,
		UserID:       token.Owner,
		RefreshToken: token.RefreshToken,
		ExpiresIn:    validUntilToExpiresIn(pkgTime.Unix(0, validUntil)),
		RedirectURI:  "",
	}

	accept := coapconv.GetAccept(options)
	encode, err := coapconv.GetEncoder(accept)
	if err != nil {
		return 0, nil, err
	}
	out, err := encode(resp)
	if err != nil {
		return 0, nil, err
	}
	return accept, out, nil
}

// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.account.swagger.json
func signUpPostHandler(r *mux.Message, client *Client) {
	logErrorAndCloseClient := func(err error, code coapCodes.Code) {
		client.logAndWriteErrorResponse(err, code, r.Token)
		if err := client.Close(); err != nil {
			log.Errorf("sign up error: %w", err)
		}
	}

	var signUp CoapSignUpRequest
	if err := cbor.ReadFrom(r.Body, &signUp); err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign up: %w", err), coapCodes.BadRequest)
		return
	}

	if signUp.AuthorizationCode == "" {
		signUp.AuthorizationCode = signUp.AuthorizationCodeLegacy
	}
	if err := signUp.checkOAuthRequest(); err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign up: %w", err), coapCodes.BadRequest)
		return
	}

	token, err := client.server.provider.Exchange(r.Context, signUp.AuthorizationProvider, signUp.AuthorizationCode)
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign up: %w", err), coapCodes.Unauthorized)
		return
	}

	claim, err := client.ValidateToken(r.Context, token.AccessToken)
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign up: %w", err), coapCodes.Unauthorized)
		return
	}

	err = client.server.VerifyDeviceID(client.tlsDeviceID, claim)
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign up: %w", err), coapCodes.Unauthorized)
		return
	}

	validUntil, ok := ValidUntil(token.Expiry)
	if !ok {
		logErrorAndCloseClient(fmt.Errorf("cannot sign up: expired access token"), coapCodes.Unauthorized)
		return
	}

	deviceID := client.server.GetDeviceID(claim, client.tlsDeviceID, signUp.DeviceID)

	ctx := kitNetGrpc.CtxWithToken(r.Context, token.AccessToken)
	if _, err := client.server.asClient.AddDevice(ctx, &pb.AddDeviceRequest{
		DeviceId: deviceID,
		UserId:   token.Owner,
	}); err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot sign up: %w", err), coapconv.GrpcErr2CoapCode(err, coapconv.Update))
		return
	}

	accept, out, err := getSignUpContent(token, validUntil, r.Options)
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign up: %w", err), coapCodes.InternalServerError)
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

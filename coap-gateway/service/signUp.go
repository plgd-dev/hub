package service

import (
	"fmt"

	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/hub/v2/coap-gateway/coapconv"
	"github.com/plgd-dev/hub/v2/identity-store/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/kit/v2/codec/cbor"
)

type CoapSignUpRequest struct {
	DeviceID                string `json:"di"`
	AuthorizationCode       string `json:"accesstoken"`
	AuthorizationCodeLegacy string `json:"authcode"`
	AuthorizationProvider   string `json:"authprovider"`
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
	return nil
}

/// Get data for sign up response
func getSignUpContent(token *oauth2.Token, owner string, validUntil int64, options message.Options) (message.MediaType, []byte, error) {
	resp := CoapSignUpResponse{
		AccessToken:  token.AccessToken.String(),
		UserID:       owner,
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
		client.logAndWriteErrorResponse(r, fmt.Errorf("cannot handle sign up: %w", err), code, r.Token)
		if err := client.Close(); err != nil {
			log.Errorf("sign up error: %w", err)
		}
	}

	var signUp CoapSignUpRequest
	if err := cbor.ReadFrom(r.Body, &signUp); err != nil {
		logErrorAndCloseClient(err, coapCodes.BadRequest)
		return
	}

	if signUp.AuthorizationCode == "" {
		signUp.AuthorizationCode = signUp.AuthorizationCodeLegacy
	}
	if err := signUp.checkOAuthRequest(); err != nil {
		logErrorAndCloseClient(err, coapCodes.BadRequest)
		return
	}

	provider, ok := client.server.providers[signUp.AuthorizationProvider]
	if !ok {
		logErrorAndCloseClient(fmt.Errorf("unknown authorization provider('%v')", signUp.AuthorizationProvider), coapCodes.Unauthorized)
		return
	}

	token, err := client.exchangeCache.Execute(r.Context, provider, signUp.AuthorizationCode)
	if err != nil {
		logErrorAndCloseClient(err, coapCodes.Unauthorized)
		return
	}
	if token.RefreshToken == "" {
		logErrorAndCloseClient(fmt.Errorf("exchange didn't return a refresh token"), coapCodes.Unauthorized)
		return
	}

	claim, err := client.ValidateToken(r.Context, token.AccessToken.String())
	if err != nil {
		logErrorAndCloseClient(err, coapCodes.Unauthorized)
		return
	}

	err = client.server.VerifyDeviceID(client.tlsDeviceID, claim)
	if err != nil {
		logErrorAndCloseClient(err, coapCodes.Unauthorized)
		return
	}

	validUntil, ok := ValidUntil(token.Expiry)
	if !ok {
		logErrorAndCloseClient(fmt.Errorf("expired access token"), coapCodes.Unauthorized)
		return
	}

	owner := claim.Owner(client.server.config.APIs.COAP.Authorization.OwnerClaim)
	if owner == "" {
		logErrorAndCloseClient(fmt.Errorf("cannot determine owner"), coapCodes.Unauthorized)
		return
	}

	deviceID := client.ResolveDeviceID(claim, signUp.DeviceID)

	ctx := kitNetGrpc.CtxWithToken(r.Context, token.AccessToken.String())
	if _, err := client.server.isClient.AddDevice(ctx, &pb.AddDeviceRequest{
		DeviceId: deviceID,
	}); err != nil {
		logErrorAndCloseClient(err, coapconv.GrpcErr2CoapCode(err, coapconv.Update))
		return
	}

	accept, out, err := getSignUpContent(token, owner, validUntil, r.Options)
	if err != nil {
		logErrorAndCloseClient(err, coapCodes.InternalServerError)
		return
	}

	client.sendResponse(r, coapCodes.Changed, r.Token, accept, out)
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
		client.logAndWriteErrorResponse(r, fmt.Errorf("forbidden request from %v", client.remoteAddrString()), coapCodes.Forbidden, r.Token)
	}
}

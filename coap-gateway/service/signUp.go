package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/plgd-dev/go-coap/v3/message"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/hub/v2/coap-gateway/coapconv"
	"github.com/plgd-dev/hub/v2/identity-store/pb"
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
	RedirectURI  string `json:"redirecturi"`
	ExpiresIn    int64  `json:"expiresin"`
}

// Check that all required request fields are set
func (request CoapSignUpRequest) checkOAuthRequest() error {
	if request.DeviceID == "" {
		return errors.New("invalid device id")
	}
	if request.AuthorizationCode == "" {
		return errors.New("invalid authorization code")
	}
	return nil
}

func encodeResponse(resp interface{}, options message.Options) (message.MediaType, []byte, error) {
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

// Get data for sign up response
func getSignUpContent(token *oauth2.Token, owner string, validUntil int64, options message.Options) (message.MediaType, []byte, error) {
	resp := CoapSignUpResponse{
		AccessToken:  token.AccessToken.String(),
		UserID:       owner,
		RefreshToken: token.RefreshToken,
		ExpiresIn:    validUntilToExpiresIn(pkgTime.Unix(0, validUntil)),
		RedirectURI:  "",
	}
	return encodeResponse(resp, options)
}

const errFmtSignUP = "cannot handle sign up: %w"

func getSignUpToken(ctx context.Context, client *session, signUp CoapSignUpRequest) (*oauth2.Token, error) {
	provider, ok := client.server.providers[signUp.AuthorizationProvider]
	if !ok {
		return nil, statusErrorf(coapCodes.Unauthorized, errFmtSignUP, fmt.Errorf("unknown authorization provider('%v')", signUp.AuthorizationProvider))
	}

	token, err := client.exchangeCache.Execute(ctx, provider, signUp.AuthorizationCode)
	if err != nil {
		// When OAuth server is not accessible, then return 503 Service Unavailable. If real error occurs them http code is mapped to code.
		return nil, statusErrorf(coapCodes.ServiceUnavailable, errFmtSignUP, err)
	}
	if token.RefreshToken == "" {
		return nil, statusErrorf(coapCodes.Unauthorized, errFmtSignUP, errors.New("exchange didn't return a refresh token"))
	}
	return token, nil
}

func getSignUpDataFromClaims(ctx context.Context, client *session, accessToken string, signUp CoapSignUpRequest) (string, string, error) {
	claim, err := client.ValidateToken(ctx, accessToken)
	if err != nil {
		return "", "", err
	}

	owner, err := claim.GetOwner(client.server.config.APIs.COAP.Authorization.OwnerClaim)
	if err != nil {
		return "", "", err
	}
	if owner == "" {
		return "", "", errors.New("cannot determine owner")
	}

	deviceID, err := client.server.VerifyAndResolveDeviceID(client.tlsDeviceID, signUp.DeviceID, claim)
	if err != nil {
		return "", "", err
	}

	return deviceID, owner, nil
}

// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.account.swagger.json
func signUpPostHandler(req *mux.Message, client *session) (*pool.Message, error) {
	var signUp CoapSignUpRequest
	if err := cbor.ReadFrom(req.Body(), &signUp); err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, errFmtSignUP, err)
	}

	if signUp.AuthorizationCode == "" {
		signUp.AuthorizationCode = signUp.AuthorizationCodeLegacy
	}
	if err := signUp.checkOAuthRequest(); err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, errFmtSignUP, err)
	}

	token, err := getSignUpToken(req.Context(), client, signUp)
	if err != nil {
		return nil, err
	}

	validUntil, ok := ValidUntil(token.Expiry)
	if !ok {
		return nil, statusErrorf(coapCodes.Unauthorized, errFmtSignUP, errors.New("expired access token"))
	}

	deviceID, owner, err := getSignUpDataFromClaims(req.Context(), client, token.AccessToken.String(), signUp)
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, errFmtSignUP, err)
	}
	setDeviceIDToTracerSpan(req.Context(), deviceID)

	ctx := kitNetGrpc.CtxWithToken(req.Context(), token.AccessToken.String())
	if _, err = client.server.isClient.AddDevice(ctx, &pb.AddDeviceRequest{
		DeviceId: deviceID,
	}); err != nil {
		return nil, statusErrorf(coapconv.GrpcErr2CoapCode(err, coapconv.Update), errFmtSignUP, err)
	}

	accept, out, err := getSignUpContent(token, owner, validUntil, req.Options())
	if err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, errFmtSignUP, err)
	}

	return client.createResponse(coapCodes.Changed, req.Token(), accept, out), nil
}

// Sign-up
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.account.swagger.json
func signUpHandler(req *mux.Message, client *session) (*pool.Message, error) {
	switch req.Code() {
	case coapCodes.POST:
		return signUpPostHandler(req, client)
	case coapCodes.DELETE:
		return signOffHandler(req, client)
	default:
		return nil, statusErrorf(coapCodes.NotFound, "unsupported method %v", req.Code())
	}
}

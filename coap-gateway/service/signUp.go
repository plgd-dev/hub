package service

import (
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
		return fmt.Errorf("invalid device id")
	}
	if request.AuthorizationCode == "" {
		return fmt.Errorf("invalid authorization code")
	}
	return nil
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

const errFmtSignUP = "cannot handle sign up: %w"

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

	provider, ok := client.server.providers[signUp.AuthorizationProvider]
	if !ok {
		return nil, statusErrorf(coapCodes.Unauthorized, errFmtSignUP, fmt.Errorf("unknown authorization provider('%v')", signUp.AuthorizationProvider))
	}

	token, err := client.exchangeCache.Execute(req.Context(), provider, signUp.AuthorizationCode)
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, errFmtSignUP, err)
	}
	if token.RefreshToken == "" {
		return nil, statusErrorf(coapCodes.Unauthorized, errFmtSignUP, fmt.Errorf("exchange didn't return a refresh token"))
	}

	claim, err := client.ValidateToken(req.Context(), token.AccessToken.String())
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, errFmtSignUP, err)
	}

	err = client.server.VerifyDeviceID(client.tlsDeviceID, claim)
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, errFmtSignUP, err)
	}

	validUntil, ok := ValidUntil(token.Expiry)
	if !ok {
		return nil, statusErrorf(coapCodes.Unauthorized, errFmtSignUP, fmt.Errorf("expired access token"))
	}

	owner := claim.Owner(client.server.config.APIs.COAP.Authorization.OwnerClaim)
	if owner == "" {
		return nil, statusErrorf(coapCodes.Unauthorized, errFmtSignUP, fmt.Errorf("cannot determine owner"))
	}

	deviceID := client.ResolveDeviceID(claim, signUp.DeviceID)
	setDeviceIDToTracerSpan(req.Context(), deviceID)

	ctx := kitNetGrpc.CtxWithToken(req.Context(), token.AccessToken.String())
	if _, err := client.server.isClient.AddDevice(ctx, &pb.AddDeviceRequest{
		DeviceId: deviceID,
	}); err != nil {
		return nil, statusErrorf(coapconv.GrpcErr2CoapCode(err, coapconv.Update), errFmtSignUP, err)
	}

	accept, out, err := getSignUpContent(token, owner, validUntil, req.Options())
	if err != nil {
		return nil, statusErrorf(coapCodes.InternalServerError, errFmtSignUP, err)
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

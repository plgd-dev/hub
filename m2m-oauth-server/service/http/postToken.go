package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/pb"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/uri"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/kit/v2/codec/json"
	"google.golang.org/grpc/codes"
)

type postRequest struct {
	ClientID            string `json:"client_id"`
	Secret              string `json:"client_secret"`
	Audience            string `json:"audience"`
	GrantType           string `json:"grant_type"`
	ClientAssertionType string `json:"client_assertion_type"`
	ClientAssertion     string `json:"client_assertion"`
	TokenName           string `json:"token_name"`
	Scope               string `json:"scope"`
	Expiration          int64  `json:"expiration"`
}

func postFormToCreateTokenRequest(r *http.Request, createTokenRequest *pb.CreateTokenRequest) {
	createTokenRequest.GrantType = r.PostFormValue(uri.GrantTypeKey)
	createTokenRequest.ClientId = r.PostFormValue(uri.ClientIDKey)
	audience := r.PostFormValue(uri.AudienceKey)
	if audience != "" {
		createTokenRequest.Audience = strings.Split(audience, " ")
	}
	scope := r.PostFormValue(uri.ScopeKey)
	if scope != "" {
		createTokenRequest.Scope = strings.Split(scope, " ")
	}
	createTokenRequest.ClientSecret = r.PostFormValue(uri.ClientSecretKey)
	createTokenRequest.ClientAssertionType = r.PostFormValue(uri.ClientAssertionTypeKey)
	createTokenRequest.ClientAssertion = r.PostFormValue(uri.ClientAssertionKey)
	createTokenRequest.TokenName = r.PostFormValue(uri.TokenNameKey)
	expiration := r.PostFormValue(uri.ExpirationKey)
	if expiration == "" {
		return
	}
	if expirationVal, err := strconv.ParseInt(expiration, 10, 64); err == nil {
		createTokenRequest.Expiration = expirationVal
	}
}

func jsonToCreateTokenRequest(req postRequest, createTokenRequest *pb.CreateTokenRequest) {
	createTokenRequest.GrantType = req.GrantType
	createTokenRequest.ClientId = req.ClientID
	audience := req.Audience
	if audience != "" {
		createTokenRequest.Audience = strings.Split(audience, " ")
	}
	scope := req.Scope
	if scope != "" {
		createTokenRequest.Scope = strings.Split(scope, " ")
	}
	createTokenRequest.ClientSecret = req.Secret
	createTokenRequest.ClientAssertionType = req.ClientAssertionType
	createTokenRequest.ClientAssertion = req.ClientAssertion
	createTokenRequest.TokenName = req.TokenName
	createTokenRequest.Expiration = req.Expiration
}

func (requestHandler *RequestHandler) postToken(w http.ResponseWriter, r *http.Request) {
	var createTokenRequest pb.CreateTokenRequest
	const cannotCreateTokenFmt = "cannot create token: %v"
	if strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		err := r.ParseForm()
		if err != nil {
			serverMux.WriteError(w, pkgGrpc.ForwardErrorf(codes.InvalidArgument, cannotCreateTokenFmt, err))
			return
		}
		postFormToCreateTokenRequest(r, &createTokenRequest)
	} else {
		var req postRequest
		err := json.ReadFrom(r.Body, &req)
		if err != nil {
			serverMux.WriteError(w, pkgGrpc.ForwardErrorf(codes.InvalidArgument, cannotCreateTokenFmt, err))
			return
		}
		jsonToCreateTokenRequest(req, &createTokenRequest)
	}
	clientID, secret, ok := r.BasicAuth()
	if ok {
		createTokenRequest.ClientId = clientID
		createTokenRequest.ClientSecret = secret
	}
	grpcResp, err := requestHandler.m2mOAuthServiceServer.CreateToken(r.Context(), &createTokenRequest)
	if err != nil {
		serverMux.WriteError(w, pkgGrpc.ForwardErrorf(codes.InvalidArgument, cannotCreateTokenFmt, err))
		return
	}
	resp := map[string]interface{}{
		uri.AccessTokenKey: grpcResp.GetAccessToken(),
		uri.ScopeKey:       strings.Join(grpcResp.GetScope(), " "),
		uri.TokenTypeKey:   grpcResp.GetTokenType(),
	}
	if grpcResp.GetExpiresIn() > 0 {
		resp[uri.ExpiresInKey] = grpcResp.GetExpiresIn()
	}

	if err = jsonResponseWriter(w, resp); err != nil {
		log.Errorf("failed to write response: %v", err)
	}
}

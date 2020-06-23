package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gofrs/uuid"

	"github.com/go-ocf/cloud/cloud2cloud-connector/store"
	"golang.org/x/oauth2"
)

func (rh *RequestHandler) HandleLinkedAccount(ctx context.Context, linkedCloud store.LinkedCloud, authCode string) (store.Token, error) {
	var oauth oauth2.Config
	oauth = linkedCloud.OAuth.ToOAuth2()
	ctx = linkedCloud.CtxWithHTTPClient(ctx)
	token, err := oauth.Exchange(ctx, authCode)
	if err != nil {
		return store.Token{}, fmt.Errorf("cannot exchange target cloud authorization code for access token: %v", err)
	}
	return store.Token{
		AccessToken:  store.AccessToken(token.AccessToken),
		Expiry:       token.Expiry,
		RefreshToken: token.RefreshToken,
	}, nil
}

func (rh *RequestHandler) oAuthCallback(w http.ResponseWriter, r *http.Request) (int, error) {
	authCode := r.FormValue("code")
	state := r.FormValue("state")

	provisionCacheDataI, ok := rh.provisionCache.Get(string(state))
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("invalid/expired OAuth state")
	}
	rh.provisionCache.Delete(string(state))
	provisionCacheData := provisionCacheDataI.(provisionCacheData)
	linkedAccount := provisionCacheData.linkedAccount
	token, err := rh.HandleLinkedAccount(r.Context(), provisionCacheData.linkedCloud, authCode)
	if err != nil {
		return http.StatusBadRequest, err
	}
	linkedAccount.TargetCloud = token
	id, err := uuid.NewV4()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	linkedAccount.ID = id.String()
	err = rh.store.InsertLinkedAccount(r.Context(), linkedAccount)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot store linked account for url %v: %v", linkedAccount.TargetURL, err)
	}
	if provisionCacheData.linkedCloud.SupportedSubscriptionsEvents.NeedPullDevices() {
		return http.StatusOK, nil
	}
	err = rh.subManager.StartSubscriptions(r.Context(), linkedAccount, provisionCacheData.linkedCloud)
	if err != nil {
		rh.store.RemoveLinkedAccount(r.Context(), linkedAccount.ID)
		return http.StatusBadRequest, fmt.Errorf("cannot start subscriptions %v: %v", linkedAccount.TargetURL, err)
	}
	return http.StatusOK, nil
}

func (rh *RequestHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.oAuthCallback(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot process oauth callback: %v", err), statusCode, w)
	}
}

package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gofrs/uuid"

	"github.com/plgd-dev/cloud/cloud2cloud-connector/store"
)

func (rh *RequestHandler) HandleLinkedAccount(ctx context.Context, linkedCloud store.LinkedCloud, authCode string) (store.Token, error) {
	oauth := linkedCloud.OAuth.ToOAuth2()
	ctx = linkedCloud.CtxWithHTTPClient(ctx)
	token, err := oauth.Exchange(ctx, authCode)
	if err != nil {
		return store.Token{}, fmt.Errorf("cannot exchange target cloud authorization code for access token: %w", err)
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
	_, _, err = rh.store.LoadOrCreateLinkedAccount(r.Context(), linkedAccount)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot store linked account %+v: %w", linkedAccount, err)
	}
	if provisionCacheData.linkedCloud.SupportedSubscriptionsEvents.NeedPullDevices() {
		return http.StatusOK, nil
	}
	rh.triggerTask(Task{
		taskType:      TaskType_SubscribeToDevices,
		linkedAccount: linkedAccount,
		linkedCloud:   provisionCacheData.linkedCloud,
	})

	return http.StatusOK, nil
}

func (rh *RequestHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.oAuthCallback(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot process oauth callback: %w", err), statusCode, w)
	}
}

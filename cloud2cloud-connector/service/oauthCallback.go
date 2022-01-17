package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2"
)

func (rh *RequestHandler) handleLinkedData(ctx context.Context, data provisionCacheData, authCode string) (provisionCacheData, error) {
	if !data.linkedAccount.Data.HasOrigin() {
		token, err := rh.provider.Exchange(ctx, authCode)
		if err != nil {
			return data, fmt.Errorf("cannot exchange origin cloud authorization code for access token: %w", err)
		}
		data.linkedAccount.Data = data.linkedAccount.Data.SetOrigin(*token)
		return data, nil
	}

	if !data.linkedAccount.Data.HasTarget() {
		oauth := data.linkedCloud.OAuth.ToDefaultOAuth2()
		ctx = data.linkedCloud.CtxWithHTTPClient(ctx)
		token, err := oauth.Exchange(ctx, authCode)
		if err != nil {
			return data, fmt.Errorf("cannot exchange target cloud authorization code for access token: %w", err)
		}
		data.linkedAccount.Data = data.linkedAccount.Data.SetTarget(oauth2.Token{
			AccessToken:  oauth2.AccessToken(token.AccessToken),
			Expiry:       token.Expiry,
			RefreshToken: token.RefreshToken,
		})
		return data, nil
	}

	return data, nil
}

func (rh *RequestHandler) oAuthCallback(w http.ResponseWriter, r *http.Request) (int, error) {
	authCode := r.FormValue("code")
	state := r.FormValue("state")

	cacheData := rh.provisionCache.Load(state)
	if cacheData == nil {
		return http.StatusBadRequest, fmt.Errorf("invalid/expired OAuth state")
	}
	rh.provisionCache.Delete(state)

	data, ok := cacheData.Data().(provisionCacheData)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("invalid/expired OAuth state")
	}

	newData, err := rh.handleLinkedData(r.Context(), data, authCode)
	if err != nil {
		return http.StatusBadRequest, err
	}

	if !newData.linkedAccount.Data.HasOrigin() {
		return http.StatusInternalServerError, fmt.Errorf("invalid linked data state(%v)", newData.linkedAccount.Data.State)
	}

	if !newData.linkedAccount.Data.HasTarget() {
		return rh.handleOAuth(w, r, newData.linkedAccount, newData.linkedCloud)
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	newData.linkedAccount.ID = id.String()
	_, _, err = rh.store.LoadOrCreateLinkedAccount(r.Context(), newData.linkedAccount)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot store linked account %+v: %w", newData.linkedAccount, err)
	}
	if newData.linkedCloud.SupportedSubscriptionEvents.NeedPullDevices() {
		return http.StatusOK, nil
	}
	rh.triggerTask(Task{
		taskType:      TaskType_SubscribeToDevices,
		linkedAccount: newData.linkedAccount,
		linkedCloud:   newData.linkedCloud,
	})

	return http.StatusOK, nil
}

func (rh *RequestHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	if statusCode, err := rh.oAuthCallback(w, r); err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot process oauth callback: %w", err), statusCode, w)
	}
}

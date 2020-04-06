package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gofrs/uuid"

	"github.com/go-ocf/ocf-cloud/openapi-connector/store"
	"golang.org/x/oauth2"
)

func (rh *RequestHandler) HandleLinkedAccount(ctx context.Context, data LinkedAccountData, authCode string) (LinkedAccountData, error) {
	var oauth oauth2.Config
	switch data.State {
	case LinkedAccountState_START:
		oauth = rh.originCloud.ToOAuth2Config()
		oauth.RedirectURL = rh.oauthCallback
		ctx = context.WithValue(ctx, oauth2.HTTPClient, http.DefaultClient)
		token, err := oauth.Exchange(ctx, authCode)
		if err != nil {
			return data, fmt.Errorf("cannot exchange origin cloud authorization code for access token: %v", err)
		}
		data.LinkedAccount.OriginCloud.AccessToken = store.AccessToken(token.AccessToken)
		data.LinkedAccount.OriginCloud.Expiry = token.Expiry
		data.LinkedAccount.OriginCloud.RefreshToken = token.RefreshToken
		data.State = LinkedAccountState_PROVISIONED_ORIGIN_CLOUD
		return data, nil
	case LinkedAccountState_PROVISIONED_ORIGIN_CLOUD:
		var h LinkedCloudHandler
		err := rh.store.LoadLinkedClouds(ctx, store.Query{ID: data.LinkedAccount.TargetCloud.LinkedCloudID}, &h)
		if err != nil {
			return data, fmt.Errorf("cannot find linked cloud with ID %v: %v", data.LinkedAccount.TargetCloud.LinkedCloudID, err)
		}
		oauth = h.linkedCloud.ToOAuth2Config()
		oauth.RedirectURL = rh.oauthCallback
		ctx = context.WithValue(ctx, oauth2.HTTPClient, http.DefaultClient)
		token, err := oauth.Exchange(ctx, authCode)
		if err != nil {
			return data, fmt.Errorf("cannot exchange target cloud authorization code for access token: %v", err)
		}
		data.LinkedAccount.TargetCloud.AccessToken = store.AccessToken(token.AccessToken)
		data.LinkedAccount.TargetCloud.Expiry = token.Expiry
		data.LinkedAccount.TargetCloud.RefreshToken = token.RefreshToken
		data.State = LinkedAccountState_PROVISIONED_TARGET_CLOUD
		return data, nil
	case LinkedAccountState_PROVISIONED_TARGET_CLOUD:
		return data, nil
	}
	return data, fmt.Errorf("unknown state %v", data.State)
}

func (rh *RequestHandler) oAuthCallback(w http.ResponseWriter, r *http.Request) (int, error) {
	authCode := r.FormValue("code")
	state := r.FormValue("state")

	linkedAccountData, ok := rh.provisionCache.Get(string(state))
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("invalid/expired OAuth state")
	}
	rh.provisionCache.Delete(string(state))

	data := linkedAccountData.(LinkedAccountData)
	newData, err := rh.HandleLinkedAccount(r.Context(), data, authCode)
	if err != nil {
		return http.StatusBadRequest, err
	}

	switch newData.State {
	case LinkedAccountState_PROVISIONED_ORIGIN_CLOUD:
		return rh.HandleOAuth(w, r, newData)
	case LinkedAccountState_PROVISIONED_TARGET_CLOUD:
		id, err := uuid.NewV4()
		if err != nil {
			return http.StatusInternalServerError, err
		}
		newData.LinkedAccount.ID = id.String()
		err = rh.store.InsertLinkedAccount(r.Context(), newData.LinkedAccount)
		if err != nil {
			return http.StatusBadRequest, fmt.Errorf("cannot store linked account for url %v: %v", data.LinkedAccount.TargetURL, err)
		}
		err = rh.subManager.StartSubscriptions(r.Context(), newData.LinkedAccount)
		if err != nil {
			rh.store.RemoveLinkedAccount(r.Context(), newData.LinkedAccount.ID)
			return http.StatusBadRequest, fmt.Errorf("cannot start subscriptions %v: %v", data.LinkedAccount.TargetURL, err)
		}
		return http.StatusOK, nil
	}
	return http.StatusInternalServerError, fmt.Errorf("invalid linked account state - %v", newData.State)
}

func (rh *RequestHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.oAuthCallback(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot process oauth callback: %v", err), statusCode, w)
	}
}

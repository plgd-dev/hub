package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/go-ocf/cloud/authorization/oauth"
	"github.com/go-ocf/cloud/cloud2cloud-connector/store"
	"github.com/patrickmn/go-cache"
	"golang.org/x/oauth2"
)

type LinkedCloudHandler struct {
	linkedCloud store.LinkedCloud
	set         bool
}

func (h *LinkedCloudHandler) Handle(ctx context.Context, iter store.LinkedCloudIter) (err error) {
	var s store.LinkedCloud
	if iter.Next(ctx, &s) {
		h.set = true
		h.linkedCloud = s
		return iter.Err()
	}
	return fmt.Errorf("not found")
}

func generateRandomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (rh *RequestHandler) GetLinkedCloud(ctx context.Context, data LinkedAccountData) (store.LinkedCloud, error) {
	switch data.State {
	case LinkedAccountState_START:
		return rh.originCloud, nil
	case LinkedAccountState_PROVISIONED_ORIGIN_CLOUD:
		var h LinkedCloudHandler
		err := rh.store.LoadLinkedClouds(ctx, store.Query{ID: data.LinkedAccount.TargetCloud.LinkedCloudID}, &h)
		if err != nil {
			return store.LinkedCloud{}, fmt.Errorf("cannot find linked cloud with ID %v: %v", data.LinkedAccount.TargetCloud.LinkedCloudID, err)
		}
		return h.linkedCloud, nil
	}
	return store.LinkedCloud{}, fmt.Errorf("state %v cannot provide linked cloud", data.State)
}

func (rh *RequestHandler) HandleOAuth(w http.ResponseWriter, r *http.Request, data LinkedAccountData) (int, error) {
	linkedCloud, err := rh.GetLinkedCloud(r.Context(), data)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	oauth := linkedCloud.ToOAuth2Config()
	oauth.RedirectURL = rh.oauthCallback
	t, err := generateRandomString(32)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("cannot generate random token")
	}
	err = rh.provisionCache.Add(t, data, cache.DefaultExpiration)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("cannot store key - collision")
	}
	url := oauth.AuthCodeURL(t, oauth2.AccessTypeOffline)
	if linkedCloud.Audience != "" {
		//"https://portal.shared.pluggedin.cloud"
		url = oauth.AuthCodeURL(t, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("audience", linkedCloud.Audience))
	}
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	return http.StatusOK, nil
}

func (rh *RequestHandler) addLinkedAccount(w http.ResponseWriter, r *http.Request) (int, error) {
	l := store.LinkedAccount{
		TargetURL: r.FormValue("target_url"),
		TargetCloud: oauth.Config{
			LinkedCloudID: r.FormValue("target_linked_cloud_id"),
		},
	}
	if l.TargetURL == "" {
		return http.StatusBadRequest, fmt.Errorf("invalid target_url")
	}
	if l.TargetCloud.LinkedCloudID == "" {
		return http.StatusBadRequest, fmt.Errorf("invalid target_linked_cloud_id")
	}

	data := LinkedAccountData{LinkedAccount: l}
	return rh.HandleOAuth(w, r, data)
}

func (rh *RequestHandler) AddLinkedAccount(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.addLinkedAccount(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot add linked account: %v", err), statusCode, w)
	}
}

package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/patrickmn/go-cache"
	"github.com/plgd-dev/cloud/cloud2cloud-connector/store"
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

func (rh *RequestHandler) HandleOAuth(w http.ResponseWriter, r *http.Request, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) (int, error) {
	linkedCloud, ok := rh.store.LoadCloud(linkedAccount.LinkedCloudID)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("cannot find linked cloud with ID %v: not found", linkedAccount.LinkedCloudID)
	}
	t, err := generateRandomString(32)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("cannot generate random token")
	}
	err = rh.provisionCache.Add(t, provisionCacheData{
		linkedAccount: linkedAccount,
		linkedCloud:   linkedCloud,
	}, cache.DefaultExpiration)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("cannot store key - collision")
	}
	oauthCfg := linkedCloud.OAuth
	if oauthCfg.RedirectURL == "" {
		oauthCfg.RedirectURL = rh.oauthCallback
	}
	url := oauthCfg.AuthCodeURL(t)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	return http.StatusOK, nil
}

func (rh *RequestHandler) addLinkedAccount(w http.ResponseWriter, r *http.Request) (int, error) {
	_, userID, err := ParseAuth(rh.ownerClaim, r.Header.Get("Authorization"))
	if err != nil {
		return http.StatusUnauthorized, fmt.Errorf("cannot get usedID from Authorization header: %w", err)
	}
	vars := mux.Vars(r)
	cloudID := vars[cloudIDKey]
	linkedCloud, ok := rh.store.LoadCloud(cloudID)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("invaid param cloud_id %v: not found", linkedCloud)
	}
	linkedAccount := store.LinkedAccount{
		LinkedCloudID: cloudID,
		UserID:        userID,
	}
	if linkedAccount.LinkedCloudID == "" {
		return http.StatusBadRequest, fmt.Errorf("invalid cloud_id")
	}
	return rh.HandleOAuth(w, r, linkedAccount, linkedCloud)
}

func (rh *RequestHandler) AddLinkedAccount(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.addLinkedAccount(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot add linked account: %w", err), statusCode, w)
	}
}

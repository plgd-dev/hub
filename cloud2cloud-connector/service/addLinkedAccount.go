package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/go-ocf/cloud/cloud2cloud-connector/store"
	"github.com/patrickmn/go-cache"
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

func (rh *RequestHandler) GetLinkedCloud(ctx context.Context, linkedCloudID string) (store.LinkedCloud, error) {
	var h LinkedCloudHandler
	err := rh.store.LoadLinkedClouds(ctx, store.Query{ID: linkedCloudID}, &h)
	if err != nil {
		return store.LinkedCloud{}, fmt.Errorf("cannot find linked cloud with ID %v: %v", linkedCloudID, err)
	}
	return h.linkedCloud, nil
}

func (rh *RequestHandler) HandleOAuth(w http.ResponseWriter, r *http.Request, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) (int, error) {
	linkedCloud, err := rh.GetLinkedCloud(r.Context(), linkedAccount.LinkedCloudID)
	if err != nil {
		return http.StatusInternalServerError, err
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
	url := linkedCloud.OAuth.AuthCodeURL(t)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	return http.StatusOK, nil
}

func (rh *RequestHandler) addLinkedAccount(w http.ResponseWriter, r *http.Request) (int, error) {
	_, userID, err := ParseAuth(r.Header.Get("Authorization"))
	if err != nil {
		return http.StatusUnauthorized, fmt.Errorf("cannot get usedID from Authorization header: %w", err)
	}
	linkedCloud, err := rh.GetLinkedCloud(r.Context(), r.FormValue("target_linked_cloud_id"))
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("invaid param target_linked_cloud_id: %w", err)
	}
	linkedAccount := store.LinkedAccount{
		TargetURL:     r.FormValue("target_url"),
		LinkedCloudID: r.FormValue("target_linked_cloud_id"),
		UserID:        userID,
	}
	if linkedAccount.TargetURL == "" {
		return http.StatusBadRequest, fmt.Errorf("invalid target_url")
	}
	if linkedAccount.LinkedCloudID == "" {
		return http.StatusBadRequest, fmt.Errorf("invalid target_linked_cloud_id")
	}
	return rh.HandleOAuth(w, r, linkedAccount, linkedCloud)
}

func (rh *RequestHandler) AddLinkedAccount(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.addLinkedAccount(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot add linked account: %v", err), statusCode, w)
	}
}

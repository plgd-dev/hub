package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	cache "github.com/plgd-dev/go-coap/v2/pkg/cache"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/store"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
)

type LinkedCloudHandler struct {
	linkedCloud store.LinkedCloud
	set         bool
}

const CacheExpiration = time.Minute * 10

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

func (rh *RequestHandler) handleOAuth(w http.ResponseWriter, r *http.Request, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) (int, error) {
	linkedCloud, ok := rh.store.LoadCloud(linkedAccount.LinkedCloudID)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("cannot find linked cloud with ID %v: not found", linkedAccount.LinkedCloudID)
	}
	t, err := generateRandomString(32)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("cannot generate token")
	}
	_, loaded := rh.provisionCache.LoadOrStore(t, cache.NewElement(provisionCacheData{
		linkedAccount: linkedAccount,
		linkedCloud:   linkedCloud,
	}, time.Now().Add(CacheExpiration), nil))
	if loaded {
		return http.StatusInternalServerError, fmt.Errorf("cannot store key - collision")
	}

	if !linkedAccount.Data.HasOrigin() {
		url := rh.provider.Config.AuthCodeURL(t)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
		return http.StatusOK, nil
	}

	if !linkedAccount.Data.HasTarget() {
		oauthCfg := linkedCloud.OAuth
		if oauthCfg.RedirectURL == "" {
			oauthCfg.RedirectURL = rh.provider.Config.RedirectURL
		}
		url := oauthCfg.AuthCodeURL(t)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
		return http.StatusOK, nil
	}
	return http.StatusOK, nil
}

func (rh *RequestHandler) addLinkedAccount(w http.ResponseWriter, r *http.Request) (int, error) {
	userID, err := grpc.OwnerFromOutgoingTokenMD(r.Context(), rh.ownerClaim)
	if err != nil {
		return http.StatusUnauthorized, fmt.Errorf("cannot get usedID from Authorization header: %w", err)
	}
	vars := mux.Vars(r)
	cloudID := vars[cloudIDKey]
	linkedCloud, ok := rh.store.LoadCloud(cloudID)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("invalid param cloud_id %v: not found", linkedCloud)
	}
	linkedAccount := store.LinkedAccount{
		LinkedCloudID: cloudID,
		UserID:        userID,
	}
	if linkedAccount.LinkedCloudID == "" {
		return http.StatusBadRequest, fmt.Errorf("invalid cloud_id")
	}
	return rh.handleOAuth(w, r, linkedAccount, linkedCloud)
}

func (rh *RequestHandler) AddLinkedAccount(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.addLinkedAccount(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot add linked account: %w", err), statusCode, w)
	}
}

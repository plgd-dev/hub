package service

import (
	"context"
	"errors"
	"time"

	pbCA "github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/security/oauth/clientcredentials"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/atomic"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
)

type LinkedHub struct {
	cfg                  *pb.Hub
	expiration           time.Duration
	certificateAuthority pbCA.CertificateAuthorityClient
	tokenCache           *clientcredentials.Cache
	closer               fn.FuncList
	invalid              atomic.Bool
	tokenExpireAt        atomic.Time
	expireAt             atomic.Time
}

func NewLinkedHub(ctx context.Context, expiration time.Duration, cfg *pb.Hub, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*LinkedHub, error) {
	if cfg.GetId() == "" {
		return nil, errors.New("hub ID is empty")
	}
	var closer fn.FuncList
	certificateAuthority, certificateAuthorityClose, err := newCertificateAuthorityClient(cfg.GetCertificateAuthority().GetGrpc().ToConfig(), fileWatcher, logger, tracerProvider)
	if err != nil {
		closer.Execute()
		return nil, err
	}
	closer.AddFunc(certificateAuthorityClose)

	tokenCache, err := clientcredentials.New(ctx, cfg.GetAuthorization().GetProvider().ToConfig(), fileWatcher, logger, tracerProvider, time.Minute)
	if err != nil {
		closer.Execute()
		return nil, err
	}
	closer.AddFunc(tokenCache.Close)

	return &LinkedHub{
		cfg:                  cfg,
		certificateAuthority: certificateAuthority,
		tokenCache:           tokenCache,
		closer:               closer,
		expiration:           expiration,
	}, nil
}

func (h *LinkedHub) SignIdentityCertificate(ctx context.Context, in *pbCA.SignCertificateRequest, opts ...grpc.CallOption) (*pbCA.SignCertificateResponse, error) {
	v, err := h.certificateAuthority.SignIdentityCertificate(ctx, in, opts...)
	if err == nil {
		h.Refresh(time.Now())
	}
	return v, err
}

func (h *LinkedHub) GetToken(ctx context.Context, key string, urlValues map[string]string, requiredClaims map[string]interface{}) (*oauth2.Token, error) {
	v, err := h.tokenCache.GetToken(ctx, key, urlValues, requiredClaims)
	if err == nil {
		h.Refresh(time.Now())
		h.tokenExpireAt.Store(v.Expiry)
	}
	return v, err
}

func (h *LinkedHub) Invalidate() {
	h.invalid.Store(true)
}

func (h *LinkedHub) GetTokenFromOAuth(ctx context.Context, urlValues map[string]string, requiredClaims map[string]interface{}) (*oauth2.Token, error) {
	t, err := h.tokenCache.GetTokenFromOAuth(ctx, urlValues, requiredClaims)
	if err == nil {
		h.Refresh(time.Now())
	}
	return t, err
}

func (h *LinkedHub) Refresh(now time.Time) {
	h.expireAt.Store(now.Add(h.expiration))
}

func (h *LinkedHub) IsExpired(now time.Time) bool {
	if h.invalid.Load() {
		return true
	}
	if h.expireAt.Load().After(now) {
		return true
	}
	tokenExpireAt := h.tokenExpireAt.Load()
	if tokenExpireAt.IsZero() {
		return false
	}
	return h.tokenExpireAt.Load().After(now)
}

func (h *LinkedHub) Close() {
	h.closer.Execute()
}

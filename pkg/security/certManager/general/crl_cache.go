package general

import (
	"context"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/http/client"
	pkgHttpUri "github.com/plgd-dev/hub/v2/pkg/net/http/uri"
	pkgTls "github.com/plgd-dev/hub/v2/pkg/security/tls"
	"github.com/plgd-dev/hub/v2/pkg/sync/task/future"
	"go.opentelemetry.io/otel/trace"
)

type CRLCache struct {
	revocationLists map[string]*future.Future
	mutex           sync.Mutex
	httpClient      *client.Client
	logger          log.Logger
}

func NewCRLCache(config pkgTls.HTTPConfigurer, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*CRLCache, error) {
	httpClient, err := NewHTTPClient(config, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, err
	}
	c := &CRLCache{
		httpClient:      httpClient,
		revocationLists: make(map[string]*future.Future),
		logger:          logger,
	}
	return c, nil
}

func (c *CRLCache) getFutureRevocationList(key string, old *future.Future) (*future.Future, future.SetFunc) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	f, ok := c.revocationLists[key]
	if !ok || f == old {
		fu, set := future.New()
		c.revocationLists[key] = fu
		return fu, set
	}
	return f, nil
}

func (c *CRLCache) GetRevocationListByHTTP(ctx context.Context, distributionPoint string) (*x509.RevocationList, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, distributionPoint, nil)
	if err != nil {
		return nil, err
	}
	req.Close = true
	resp, err := c.httpClient.HTTP().Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if errC := resp.Body.Close(); errC != nil {
			c.logger.Errorf("failed to close response body stream: %v", errC)
		}
	}()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %v while downloading CRL from %s", resp.StatusCode, distributionPoint)
	}
	return x509.ParseRevocationList(respBody)
}

func (c *CRLCache) GetRevocationList(ctx context.Context, distributionPoint string) (*x509.RevocationList, error) {
	dp := pkgHttpUri.CanonicalURI(distributionPoint)

	var oldF *future.Future
	var setF future.SetFunc
	for {
		f, set := c.getFutureRevocationList(dp, oldF)
		if set != nil {
			setF = set
			break
		}
		v, err := f.Get(ctx)
		if err == nil {
			rl, ok := v.(*x509.RevocationList)
			if !ok {
				return nil, fmt.Errorf("invalid object type(%T) in a future", v)
			}
			if !pkgTls.IsExpired(rl.NextUpdate.UnixNano()) {
				c.logger.Debugf("valid revocation list found in cache")
				return rl, nil
			}
			c.logger.Debugf("expired revocation list found in cache")
		}
		oldF = f
	}
	c.logger.Debugf("downloading revocation list")
	rl, err := c.GetRevocationListByHTTP(ctx, dp)
	setF(rl, err)
	return rl, err
}

// TODO: add expired CRL clean-up

func (c *CRLCache) Close() {
	c.httpClient.Close()
}

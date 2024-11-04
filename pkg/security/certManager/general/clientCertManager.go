package general

import (
	"crypto/tls"
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/http/client"
	pkgTls "github.com/plgd-dev/hub/v2/pkg/security/tls"
	"go.opentelemetry.io/otel/trace"
)

func ClientConfig(caPoolArray []urischeme.URIScheme, keyFile, certFile urischeme.URIScheme, useSystemCAPool bool, crl pkgTls.CRLConfig) Config {
	return Config{
		CAPool:                    caPoolArray,
		KeyFile:                   keyFile,
		CertFile:                  certFile,
		ClientCertificateRequired: false,
		UseSystemCAPool:           useSystemCAPool,
		CRL:                       crl,
	}
}

func ClientLogger(logger log.Logger) log.Logger {
	return logger.With(log.CertManagerKey, "client")
}

// CertManager holds certificates from filesystem watched for changes
type ClientCertManager struct {
	c *CertManager
}

// GetTLSConfig returns tls configuration for clients
func (c *ClientCertManager) GetTLSConfig() *tls.Config {
	return c.c.GetClientTLSConfig()
}

// Close ends watching certificates
func (c *ClientCertManager) Close() {
	c.c.Close()
}

// New creates a new certificate manager which watches for certs in a filesystem
func NewClientCertManager(config pkgTls.ClientConfig, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, opts ...SetOption) (*ClientCertManager, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	caPoolArray, _ := config.CAPoolArray()

	cfg := ClientConfig(caPoolArray, config.KeyFile, config.CertFile, config.UseSystemCAPool, config.CRL)
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	c, err := New(cfg, fileWatcher, ClientLogger(logger), tracerProvider, opts...)
	if err != nil {
		return nil, err
	}
	return &ClientCertManager{
		c: c,
	}, nil
}

func NewHTTPClient(config pkgTls.HTTPConfigurer, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, opts ...SetOption) (*client.Client, error) {
	cm, err := NewClientCertManager(config.GetTLS(), fileWatcher, logger, tracerProvider, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create cert manager %w", err)
	}
	return client.New(config, cm, tracerProvider)
}

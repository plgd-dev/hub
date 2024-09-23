package client

import (
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/http/client"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/general"
	pkgTls "github.com/plgd-dev/hub/v2/pkg/security/tls"
	"go.opentelemetry.io/otel/trace"
)

type Config = pkgTls.ClientConfig

// CertManager holds certificates from filesystem watched for changes
type CertManager = general.ClientCertManager

func New(config Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tp trace.TracerProvider) (*CertManager, error) {
	return general.NewClientCertManager(config, fileWatcher, logger, tp)
}

func NewHTTPClient(config pkgTls.HTTPConfigurer, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*client.Client, error) {
	return general.NewHTTPClient(config, fileWatcher, logger, tracerProvider)
}

package http

import (
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/pkg/net/http/server"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	HTTPConnection listener.Config
	HTTPServer     server.Config

	ServiceName       string
	FileWatcher       *fsnotify.Watcher
	Logger            log.Logger
	WhiteEndpointList []kitNetHttp.RequestMatcher
	AuthRules         map[string][]kitNetHttp.AuthArgs
	TraceProvider     trace.TracerProvider
	Validator         *validator.Validator
}

func (c *Config) Validate() error {
	if err := c.HTTPConnection.Validate(); err != nil {
		return fmt.Errorf("HTTPConnection.%w", err)
	}
	if c.TraceProvider == nil {
		return fmt.Errorf("traceProvider is required")
	}
	if c.AuthRules == nil {
		return fmt.Errorf("authRules is required")
	}
	if c.FileWatcher == nil {
		return fmt.Errorf("fileWatcher is required")
	}
	if c.Logger == nil {
		return fmt.Errorf("logger is required")
	}
	if c.ServiceName == "" {
		return fmt.Errorf("serviceName is required")
	}

	return nil
}

// String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}

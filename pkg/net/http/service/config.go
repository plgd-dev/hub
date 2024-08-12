package http

import (
	"errors"
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgHttpJwt "github.com/plgd-dev/hub/v2/pkg/net/http/jwt"
	"github.com/plgd-dev/hub/v2/pkg/net/http/server"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	HTTPConnection listener.Config
	HTTPServer     server.Config

	ServiceName          string
	FileWatcher          *fsnotify.Watcher
	Logger               log.Logger
	WhiteEndpointList    []pkgHttpJwt.RequestMatcher
	AuthRules            map[string][]pkgHttpJwt.AuthArgs
	TraceProvider        trace.TracerProvider
	Validator            *validator.Validator
	QueryCaseInsensitive map[string]string
}

func (c *Config) Validate() error {
	if err := c.HTTPConnection.Validate(); err != nil {
		return fmt.Errorf("HTTPConnection.%w", err)
	}
	if c.TraceProvider == nil {
		return errors.New("traceProvider is required")
	}
	if c.AuthRules == nil {
		return errors.New("authRules is required")
	}
	if c.FileWatcher == nil {
		return errors.New("fileWatcher is required")
	}
	if c.Logger == nil {
		return errors.New("logger is required")
	}
	if c.ServiceName == "" {
		return errors.New("serviceName is required")
	}

	return nil
}

// String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}

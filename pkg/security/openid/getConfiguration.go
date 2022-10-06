package openid

import (
	"context"
	"fmt"
	"net/http"

	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/kit/v2/codec/json"
)

func GetConfiguration(ctx context.Context, httpClient *http.Client, domain string) (Config, error) {
	href := domain + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, href, nil)
	if err != nil {
		return Config{}, fmt.Errorf("cannot create request for GET %v: %w", href, err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return Config{}, fmt.Errorf("cannot GET %v: %w", href, err)
	}
	if resp.Body == nil {
		return Config{}, fmt.Errorf("invalid response GET %v response: is empty", href)
	}
	defer func() {
		if errC := resp.Body.Close(); errC != nil {
			log.Errorf("failed to close response body stream: %w", errC)
		}
	}()
	var cfg Config
	if err = json.ReadFrom(resp.Body, &cfg); err != nil {
		return Config{}, fmt.Errorf("cannot decode GET %v response: %w", href, err)
	}
	if err = cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid property of GET %v response: %w", href, err)
	}

	return cfg, nil
}

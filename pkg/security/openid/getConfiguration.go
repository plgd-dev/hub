package openid

import (
	"context"
	"fmt"
	"net/http"

	"github.com/plgd-dev/cloud/v2/pkg/log"
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
		if err := resp.Body.Close(); err != nil {
			log.Errorf("failed to close response body stream: %w", err)
		}
	}()
	var cfg Config
	err = json.ReadFrom(resp.Body, &cfg)
	if err != nil {
		return Config{}, fmt.Errorf("cannot decode GET %v response: %w", href, err)
	}
	err = cfg.Validate()
	if err != nil {
		return Config{}, fmt.Errorf("invalid property of GET %v response: %w", href, err)
	}

	return cfg, nil
}

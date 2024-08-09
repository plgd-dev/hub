package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/plgd-dev/hub/v2/m2m-oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/config"
	testService "github.com/plgd-dev/hub/v2/test/service"
)

func TestService(t *testing.T) {
	cfg := test.MakeConfig(t)

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()
	webTearDown := testService.SetUp(ctx, t, testService.WithM2MOAuthConfig(cfg))
	defer webTearDown()

	fmt.Printf("cfg: %v\n", cfg.String())
}

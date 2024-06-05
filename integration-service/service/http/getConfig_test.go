package http_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/test/service"
)

func TestRequestHandlerGetDevices(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	tearDown := service.SetUp(ctx, t)
	defer tearDown()

}

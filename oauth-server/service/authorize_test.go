package service_test

import (
	"testing"
	"time"

	"github.com/plgd-dev/cloud/oauth-server/test"
)

func TestRequestHandler_authorize(t *testing.T) {
	webTearDown := test.SetUp(t)
	defer webTearDown()

	time.Sleep(time.Second * 3600)
}

package service_test

import (
	"testing"

	"github.com/go-ocf/cloud/http-gateway/service"
	"github.com/go-ocf/cloud/http-gateway/test"
	"github.com/stretchr/testify/require"
)

func TestBackendConfig(t *testing.T) {
	s := test.NewTestBackendConfig().String()
	_, err := service.ParseConfig(s)
	require.NoError(t, err)
}

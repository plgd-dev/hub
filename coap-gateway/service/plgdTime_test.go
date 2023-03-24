//go:build test
// +build test

package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema/plgdtime"
	"github.com/plgd-dev/go-coap/v3/message"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/stretchr/testify/require"
)

func TestPlgdTime(t *testing.T) {
	shutdown := setUp(t)
	defer shutdown()
	co := testCoapDial(t, "", true, true, time.Now().Add(time.Minute))
	if co == nil {
		return
	}
	defer func() {
		_ = co.Close()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	resp, err := co.Get(ctx, plgdtime.ResourceURI)
	require.NoError(t, err)
	require.Equal(t, coapCodes.Content, resp.Code())
	contentType, err := resp.ContentFormat()
	require.NoError(t, err)
	require.Equal(t, message.AppOcfCbor, contentType)
	var plgdTime plgdtime.PlgdTime
	err = cbor.ReadFrom(resp.Body(), &plgdTime)
	require.NoError(t, err)
	require.NotEmpty(t, plgdTime.Time)
}

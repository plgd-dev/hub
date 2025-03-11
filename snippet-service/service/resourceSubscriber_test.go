package service_test

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/snippet-service/service"
	"github.com/plgd-dev/hub/v2/snippet-service/test"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	hubTestService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type testHandler struct {
	ch chan *events.ResourceChanged
}

func (h *testHandler) Handle(ctx context.Context, iter eventbus.Iter) (err error) {
	for {
		ev, ok := iter.Next(ctx)
		if !ok {
			return iter.Err()
		}
		var s events.ResourceChanged
		if ev.EventType() != s.EventType() {
			continue
		}
		if err := ev.Unmarshal(&s); err != nil {
			return err
		}
		h.ch <- &s
	}
}

func TestResourceSubscriber(t *testing.T) {
	deviceID := hubTest.MustFindDeviceByName(hubTest.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := hubTestService.SetUp(ctx, t)
	defer tearDown()

	logger := log.NewLogger(log.MakeDefaultConfig())
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		err = fileWatcher.Close()
		require.NoError(t, err)
	}()
	h := testHandler{
		ch: make(chan *events.ResourceChanged, 8),
	}
	cfg := test.MakeConfig(t)
	rs, err := service.NewResourceSubscriber(ctx, cfg.Clients.EventBus.NATS, cfg.Clients.EventBus.SubscriptionID, fileWatcher, logger, noop.NewTracerProvider(), &h)
	require.NoError(t, err)
	defer rs.Close()

	ctx = pkgGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	conn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: hubTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := pb.NewGrpcGatewayClient(conn)

	resources := hubTest.GetAllBackendResourceLinks()
	_, shutdownDevSim := hubTest.OnboardDevSim(ctx, t, c, deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, resources)
	defer shutdownDevSim()

	events := make(map[string]*events.ResourceChanged)
	stop := false
	for !stop {
		select {
		case ev := <-h.ch:
			id := ev.GetResourceId().GetDeviceId() + ":" + ev.GetResourceId().GetHref()
			events[id] = ev
		case <-time.After(time.Second * 3):
			stop = true
			break
		case <-ctx.Done():
			require.Fail(t, "timeout")
		}
	}

	require.Len(t, events, len(resources))
}

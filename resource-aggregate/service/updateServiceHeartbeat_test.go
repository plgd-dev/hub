package service_test

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/publisher"
	natsTest "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/test"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/service"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewServiceHeartbeat(t *testing.T) {
	ctx := context.Background()
	config := raTest.MakeConfig(t)
	config.Clients.Eventstore.ConcurrencyExceptionMaxRetry = 16
	logger := log.NewLogger(config.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, fileWatcher, logger, noop.NewTracerProvider(), publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	serviceHeartbeat := service.NewServiceHeartbeat(config, eventstore, publisher, logger)
	defer serviceHeartbeat.Close()

	const num = 100
	var wg sync.WaitGroup
	wg.Add(num)
	chans := make([]chan service.UpdateServiceMetadataResponseChanData, num)
	for i := range num {
		time.Sleep(time.Millisecond)
		chans[i] = make(chan service.UpdateServiceMetadataResponseChanData, 1)
		go func(j int) {
			defer wg.Done()
			err := serviceHeartbeat.ProcessRequest(service.UpdateServiceMetadataReqResp{
				Request: &commands.UpdateServiceMetadataRequest{
					Update: &commands.UpdateServiceMetadataRequest_Heartbeat{
						Heartbeat: &commands.ServiceHeartbeat{
							ServiceId: fmt.Sprintf("instanceId-%v", j),
							Register:  true,
						},
					},
				},
				ResponseChan: chans[j],
			})
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()
	cases := make([]reflect.SelectCase, len(chans)+1)
	cases[0] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ctx.Done())}
	for i, ch := range chans {
		cases[i+1] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
	}
	for range chans {
		chosen, value, ok := reflect.Select(cases)
		if ok {
			if chosen == 0 {
				require.Fail(t, "context canceled")
			}
			if chosen != 0 {
				data := value.Interface().(service.UpdateServiceMetadataResponseChanData)
				require.NoError(t, data.Err)
			}
		}
		if !ok {
			require.Fail(t, "channel closed")
			break
		}
	}
}

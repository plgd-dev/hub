package service_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/publisher"
	natsTest "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/test"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/resource-aggregate/service"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestNewServiceStatus(t *testing.T) {
	ctx := context.Background()
	config := raTest.MakeConfig(t)
	logger := log.NewLogger(config.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, trace.NewNoopTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, trace.NewNoopTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		assert.NoError(t, errC)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, fileWatcher, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	serviceStatus, err := service.NewServiceStatus(config, eventstore, publisher, logger)
	require.NoError(t, err)
	defer serviceStatus.Close()

	const num = 100
	var wg sync.WaitGroup
	wg.Add(100)
	for i := 0; i < num; i++ {
		time.Sleep(time.Millisecond)
		go func(j int) {
			defer wg.Done()
			serviceStatus.SetOfflineServices([]*events.ServicesStatus_Status{{
				Id: fmt.Sprintf("instanceId-%v", j),
			}})
		}(i)
	}
	wg.Wait()
	for {
		if serviceStatus.NumOfflineServices() == 0 {
			break
		}
		time.Sleep(time.Millisecond * 500)
	}
}

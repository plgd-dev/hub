package service

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/pkg/net/grpc/client"
	"github.com/plgd-dev/cloud/pkg/net/grpc/server"
	raClient "github.com/plgd-dev/cloud/resource-aggregate/client"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"go.uber.org/zap"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type closeFunc []func()

func (s closeFunc) Close() {
	if len(s) == 0 {
		return
	}
	for _, f := range s {
		f()
	}
}

// RequestHandler handles incoming requests.
type RequestHandler struct {
	pb.UnimplementedGrpcGatewayServer
	resourceDirectoryClient pb.GrpcGatewayClient

	resourceAggregateClient *raClient.Client
	closeFunc               closeFunc
}

func AddHandler(ctx context.Context, svr *server.Server, config ClientsConfig, logger *zap.Logger, goroutinePoolGo func(func()) error) error {
	handler, err := NewRequestHandlerFromConfig(ctx, config, logger, goroutinePoolGo)
	if err != nil {
		return err
	}
	svr.AddCloseFunc(handler.Close)
	pb.RegisterGrpcGatewayServer(svr.Server, handler)
	return nil
}

// Register registers the handler instance with a gRPC server.
func Register(server *grpc.Server, handler *RequestHandler) {
	pb.RegisterGrpcGatewayServer(server, handler)
}

func NewRequestHandlerFromConfig(ctx context.Context, config ClientsConfig, logger *zap.Logger, goroutinePoolGo func(func()) error) (*RequestHandler, error) {
	var closeFunc closeFunc

	resourceSubscriber, err := subscriber.New(config.Eventbus.NATS, logger, subscriber.WithGoPool(goroutinePoolGo))
	if err != nil {
		closeFunc.Close()
		return nil, fmt.Errorf("cannot create eventbus subscriber: %w", err)
	}
	closeFunc = append(closeFunc, resourceSubscriber.Close)

	rdConn, err := client.New(config.ResourceDirectory.Connection, logger)
	if err != nil {
		closeFunc.Close()
		return nil, fmt.Errorf("cannot connect to resource-directory: %w", err)
	}
	closeFunc = append(closeFunc, func() {
		err := rdConn.Close()
		if err != nil {
			logger.Sugar().Errorf("error occurs during close connection to resource-directory: %w", err)
		}
	})
	resourceDirectoryClient := pb.NewGrpcGatewayClient(rdConn.GRPC())
	raConn, err := client.New(config.ResourceAggregate.Connection, logger)
	if err != nil {
		closeFunc.Close()
		return nil, fmt.Errorf("cannot connect to resource-aggregate: %w", err)
	}
	closeFunc = append(closeFunc, func() {
		err := raConn.Close()
		if err != nil {
			logger.Sugar().Errorf("error occurs during close connection to resource-aggregate: %w", err)
		}
	})
	resourceAggregateClient := raClient.New(raConn.GRPC(), resourceSubscriber)

	return NewRequestHandler(
		resourceDirectoryClient,
		resourceAggregateClient,
		closeFunc,
	), nil
}

// NewRequestHandler factory for new RequestHandler.
func NewRequestHandler(
	resourceDirectoryClient pb.GrpcGatewayClient,
	resourceAggregateClient *raClient.Client,
	closeFunc closeFunc,
) *RequestHandler {
	return &RequestHandler{
		resourceDirectoryClient: resourceDirectoryClient,
		resourceAggregateClient: resourceAggregateClient,
		closeFunc:               closeFunc,
	}
}

func (r *RequestHandler) SubscribeForEvents(srv pb.GrpcGateway_SubscribeForEventsServer) (errRet error) {
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()
	rd, err := r.resourceDirectoryClient.SubscribeForEvents(ctx)
	if err != nil {
		return log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot subscribe for events: %v", err))
	}
	clientErr := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	defer func() {
		wg.Wait()
		select {
		case err := <-clientErr:
			log.LogAndReturnError(err)
			if errRet != nil {
				errRet = err
			}
		default:
		}
	}()
	go func() {
		defer wg.Done()
		for {
			req, err := srv.Recv()
			if err == io.EOF {
				cancel()
				clientErr <- err
				return
			}
			if err != nil {
				cancel()
				clientErr <- kitNetGrpc.ForwardErrorf(codes.Internal, "cannot receive commands: %v", err)
				return
			}
			err = rd.Send(req)
			if err != nil {
				cancel()
				clientErr <- kitNetGrpc.ForwardErrorf(codes.Internal, "cannot send commands: %v", err)
				return
			}
		}
	}()

	for {
		resp, err := rd.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot receive events: %v", err))
		}
		err = srv.Send(resp)
		if err != nil {
			return log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot send events: %v", err))
		}
	}
	return nil
}

func (r *RequestHandler) Close() {
	r.closeFunc.Close()
}

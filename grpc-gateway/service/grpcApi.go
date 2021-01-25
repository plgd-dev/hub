package service

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/kit/log"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

// RequestHandler handles incoming requests.
type RequestHandler struct {
	pb.UnimplementedGrpcGatewayServer
	resourceDirectoryClient pb.GrpcGatewayClient
	closeFunc               func()
}

/*type HandlerConfig struct {
	Service RDConfig
}
*/
func AddHandler(svr *kitNetGrpc.Server, config RDConfig, clientTLS *tls.Config) error {
	handler, err := NewRequestHandlerFromConfig(config, clientTLS)
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

func NewRequestHandlerFromConfig(config RDConfig, clientTLS *tls.Config) (*RequestHandler, error) {
	rdConn, err := grpc.Dial(
		config.ResourceDirectoryAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(clientTLS)),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to resource aggregate: %w", err)
	}
	resourceDirectoryClient := pb.NewGrpcGatewayClient(rdConn)

	closeFunc := func() {
		rdConn.Close()
	}

	h := NewRequestHandler(
		resourceDirectoryClient,
		closeFunc,
	)
	return h, nil
}

// NewRequestHandler factory for new RequestHandler.
func NewRequestHandler(
	resourceDirectoryClient pb.GrpcGatewayClient,
	closeFunc func(),
) *RequestHandler {
	return &RequestHandler{
		resourceDirectoryClient: resourceDirectoryClient,
		closeFunc:               closeFunc,
	}
}

func logAndReturnError(err error) error {
	if errors.Is(err, io.EOF) {
		return err
	}
	log.Errorf("%v", err)
	return err
}

func (r *RequestHandler) SubscribeForEvents(srv pb.GrpcGateway_SubscribeForEventsServer) (errRet error) {
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()
	rd, err := r.resourceDirectoryClient.SubscribeForEvents(ctx)
	if err != nil {
		return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot subscribe for events: %v", err)
	}
	clientErr := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	defer func() {
		wg.Wait()
		select {
		case err := <-clientErr:
			logAndReturnError(err)
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
			return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot receive events: %v", err)
		}
		err = srv.Send(resp)
		if err != nil {
			return kitNetGrpc.ForwardErrorf(codes.Internal, "cannot send events: %v", err)
		}
	}
	return nil
}

func (r *RequestHandler) Close() {
	r.closeFunc()
}

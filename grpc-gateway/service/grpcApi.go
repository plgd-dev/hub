package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"sync"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"

	"github.com/go-ocf/kit/security/oauth/manager"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

// RequestHandler handles incoming requests.
type RequestHandler struct {
	resourceDirectoryClient pb.GrpcGatewayClient
	closeFunc               func()
}

type HandlerConfig struct {
	Service Config
}

func AddHandler(svr *kitNetGrpc.Server, config HandlerConfig, clientTLS *tls.Config) error {
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

func NewRequestHandlerFromConfig(config HandlerConfig, clientTLS *tls.Config) (*RequestHandler, error) {
	svc := config.Service
	oauthMgr, err := manager.NewManagerFromConfiguration(svc.OAuth, clientTLS)
	if err != nil {
		return nil, fmt.Errorf("cannot create oauth manager: %w", err)
	}
	rdConn, err := grpc.Dial(
		svc.ResourceDirectoryAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(clientTLS)),
		grpc.WithPerRPCCredentials(kitNetGrpc.NewOAuthAccess(oauthMgr.GetToken)),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to resource aggregate: %w", err)
	}
	resourceDirectoryClient := pb.NewGrpcGatewayClient(rdConn)

	closeFunc := func() {
		rdConn.Close()
		oauthMgr.Close()
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
	log.Errorf("%v", err)
	return err
}

func makeCtx(ctx context.Context) context.Context {
	token, err := kitNetGrpc.TokenFromMD(ctx)
	if err != nil {
		ctx = kitNetGrpc.CtxWithToken(ctx, token)
	}
	return ctx
}

func (r *RequestHandler) SubscribeForEvents(srv pb.GrpcGateway_SubscribeForEventsServer) (errRet error) {
	ctx, cancel := context.WithCancel(makeCtx(srv.Context()))
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

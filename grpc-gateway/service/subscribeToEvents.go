package service

import (
	"context"
	"io"
	"sync"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"

	"google.golang.org/grpc/codes"
)

func (r *RequestHandler) SubscribeToEvents(srv pb.GrpcGateway_SubscribeToEventsServer) (errRet error) {
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()
	rd, err := r.resourceDirectoryClient.SubscribeToEvents(ctx)
	if err != nil {
		return log.LogAndReturnError(kitNetGrpc.ForwardErrorf(codes.Internal, "cannot subscribe to events: %v", err))
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

package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pbRD "github.com/plgd-dev/hub/v2/resource-directory/pb"
	"github.com/plgd-dev/kit/v2/strings"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (r *RequestHandler) GetLatestDeviceETags(ctx context.Context, req *pbRD.GetLatestDeviceETagsRequest) (*pbRD.GetLatestDeviceETagsResponse, error) {
	_, err := kitNetGrpc.OwnerFromTokenMD(ctx, r.ownerCache.OwnerClaim())
	if err != nil {
		return nil, kitNetGrpc.ForwardFromError(codes.InvalidArgument, err)
	}
	deviceIDs, err := r.getOwnerDevices(ctx)
	if err != nil {
		return nil, log.LogAndReturnError(status.Errorf(status.Convert(err).Code(), "cannot get latest device etag: %v", err))
	}
	deviceIds := strings.MakeSet(deviceIDs...)
	if len(filterDevices(deviceIds, []string{req.GetDeviceId()})) == 0 {
		return nil, status.Errorf(codes.NotFound, "cannot get latest device etag: device %v not found", req.GetDeviceId())
	}
	etag, err := r.eventStore.GetLatestDeviceETags(ctx, req.GetDeviceId(), req.GetLimit())
	if err != nil {
		return nil, kitNetGrpc.ForwardFromError(codes.InvalidArgument, fmt.Errorf("cannot get latest device etag: %w", err))
	}
	return &pbRD.GetLatestDeviceETagsResponse{
		Etags: etag,
	}, nil
}

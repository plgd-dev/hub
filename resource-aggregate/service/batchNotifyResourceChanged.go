package service

import (
	"context"
	"fmt"

	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
)

func (r RequestHandler) BatchNotifyResourceChanged(ctx context.Context, request *commands.BatchNotifyResourceChangedRequest) (*commands.BatchNotifyResourceChangedResponse, error) {
	owner, err := kitNetGrpc.OwnerFromTokenMD(ctx, r.config.APIs.GRPC.Authorization.OwnerClaim)
	if err != nil {
		return nil, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot process batch  notify resource changed: invalid owner: %v", err)
	}
	var errors []error
	for _, notify := range request.GetBatch() {
		err := r.validateAccessToDeviceWithOwner(ctx, notify.GetResourceId().GetDeviceId(), owner)
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot notify resource changed %v: %w", notify.GetResourceId().ToString(), err))
			continue
		}
		err = r.notifyResourceChanged(ctx, notify, owner)
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot notify resource changed %v: %w", notify.GetResourceId().ToString(), err))
		}
	}
	if len(errors) > 0 {
		return nil, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot process batch  notify resource changed: %v", err)
	}
	auditContext := commands.NewAuditContext(owner, "")
	return &commands.BatchNotifyResourceChangedResponse{
		AuditContext: auditContext,
	}, nil
}

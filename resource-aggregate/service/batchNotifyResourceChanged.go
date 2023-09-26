package service

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
)

func (r RequestHandler) BatchNotifyResourceChanged(ctx context.Context, request *commands.BatchNotifyResourceChangedRequest) (*commands.BatchNotifyResourceChangedResponse, error) {
	owner, err := grpc.OwnerFromTokenMD(ctx, r.config.APIs.GRPC.Authorization.OwnerClaim)
	if err != nil {
		return nil, grpc.ForwardErrorf(codes.InvalidArgument, "cannot process batch  notify resource changed: invalid owner: %v", err)
	}
	userID, err := grpc.SubjectFromTokenMD(ctx)
	if err != nil {
		return nil, grpc.ForwardErrorf(codes.InvalidArgument, "cannot process batch  notify resource changed: invalid userID: %v", err)
	}

	var errors *multierror.Error
	for _, notify := range request.GetBatch() {
		err := r.validateAccessToDeviceWithOwner(ctx, notify.GetResourceId().GetDeviceId(), owner)
		if err != nil {
			errors = multierror.Append(errors, fmt.Errorf("cannot notify resource changed %v: %w", notify.GetResourceId().ToString(), err))
			continue
		}
		err = r.notifyResourceChanged(ctx, notify, userID, owner)
		if err != nil {
			errors = multierror.Append(errors, fmt.Errorf("cannot notify resource changed %v: %w", notify.GetResourceId().ToString(), err))
		}
	}
	if errors.ErrorOrNil() != nil {
		return nil, grpc.ForwardErrorf(codes.InvalidArgument, "cannot process batch notify resource changed: %w", errors)
	}
	auditContext := commands.NewAuditContext(userID, "", owner)
	return &commands.BatchNotifyResourceChangedResponse{
		AuditContext: auditContext,
	}, nil
}

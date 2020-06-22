package service

import (
	"context"

	raCqrs "github.com/go-ocf/cloud/resource-aggregate/cqrs"
	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/kit/codec/cbor"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/sdk/schema/cloud"
)

func updateCloudStatus(ctx context.Context, raClient pbRA.ResourceAggregateClient, userID, deviceID string, online bool, cmdMeta pbCQRS.CommandMetadata) error {
	status := cloud.Status{
		ResourceTypes: cloud.StatusResourceTypes,
		Interfaces:    cloud.StatusInterfaces,
		Online:        online,
	}
	data, err := cbor.Encode(status)
	if err != nil {
		return err
	}

	request := pbRA.NotifyResourceChangedRequest{
		ResourceId: raCqrs.MakeResourceId(deviceID, cloud.StatusHref),
		Content: &pbRA.Content{
			ContentType:       message.AppOcfCbor.String(),
			CoapContentFormat: int32(message.AppOcfCbor),
			Data:              data,
		},
		Status:          pbRA.Status_OK,
		CommandMetadata: &cmdMeta,
		AuthorizationContext: &pbCQRS.AuthorizationContext{
			DeviceId: deviceID,
		},
	}

	_, err = raClient.NotifyResourceChanged(kitNetGrpc.CtxWithUserID(ctx, userID), &request)
	return err
}

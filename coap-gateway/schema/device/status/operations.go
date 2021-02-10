package status

import (
	"context"
	"time"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/codec/cbor"
)

// Publish publishes the device cloud state resource.
func Publish(ctx context.Context, client pb.ResourceAggregateClient, deviceID string, cmdMetadata *pb.CommandMetadata, authCtx *pb.AuthorizationContext) error {
	_, err := client.PublishResource(ctx, &pb.PublishResourceRequest{
		AuthorizationContext: authCtx,
		ResourceId: &pb.ResourceId{
			DeviceId: deviceID,
			Href:     Href,
		},
		Resource: &pb.Resource{
			Id:            utils.MakeResourceID(deviceID, Href),
			Href:          Href,
			ResourceTypes: ResourceTypes,
			Interfaces:    Interfaces,
			DeviceId:      deviceID,
			Policies:      &pb.Policies{BitFlags: int32(3)},
			Title:         Title,
		},
		CommandMetadata: cmdMetadata,
	})

	return err
}

func update(ctx context.Context, client pb.ResourceAggregateClient, deviceID string, state State, validUntil time.Time, cmdMetadata *pb.CommandMetadata, authCtx *pb.AuthorizationContext) error {
	data, err := cbor.Encode(Status{
		ResourceTypes: ResourceTypes,
		Interfaces:    Interfaces,
		State:         state,
		ValidUntil:    validUntil.Unix(),
	})
	if err != nil {
		return err
	}

	request := pb.NotifyResourceChangedRequest{
		ResourceId: &pb.ResourceId{
			DeviceId: deviceID,
			Href:     Href,
		},
		Content: &pb.Content{
			ContentType:       message.AppOcfCbor.String(),
			CoapContentFormat: int32(message.AppOcfCbor),
			Data:              data,
		},
		Status:               pb.Status_OK,
		CommandMetadata:      cmdMetadata,
		AuthorizationContext: authCtx,
	}

	_, err = client.NotifyResourceChanged(ctx, &request)
	return err
}

// SetOnline set state of the device to online. If validUntil.IsZero() the online state never expire. To refresh online state call again SetOnline.
func SetOnline(ctx context.Context, client pb.ResourceAggregateClient, deviceID string, validUntil time.Time, cmdMetadata *pb.CommandMetadata, authCtx *pb.AuthorizationContext) error {
	return update(ctx, client, deviceID, State_Online, validUntil, cmdMetadata, authCtx)
}

// SetOffline set state of the device to offine.
func SetOffline(ctx context.Context, client pb.ResourceAggregateClient, deviceID string, cmdMetadata *pb.CommandMetadata, authCtx *pb.AuthorizationContext) error {
	return update(ctx, client, deviceID, State_Offline, time.Time{}, cmdMetadata, authCtx)
}

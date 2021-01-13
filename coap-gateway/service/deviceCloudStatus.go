package service

import (
	"context"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/codec/cbor"

	pbCQRS "github.com/plgd-dev/cloud/resource-aggregate/pb"
	pbRA "github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/sdk/schema"
	"github.com/plgd-dev/sdk/schema/cloud"
)

func (client *Client) PublishCloudDeviceStatus(ctx context.Context, deviceID string, authCtx *pbCQRS.AuthorizationContext) error {
	devStatus := schema.ResourceLink{
		Href:          cloud.StatusHref,
		ResourceTypes: cloud.StatusResourceTypes,
		Interfaces:    cloud.StatusInterfaces,
		DeviceID:      deviceID,
		Policy: &schema.Policy{
			BitMask: 3,
		},
		Title: "Cloud device status",
	}
	_, err := client.publishResource(ctx, devStatus, int32(0), client.remoteAddrString(), client.coapConn.Sequence(), authCtx)
	return err
}

func (client *Client) UpdateCloudDeviceStatus(ctx context.Context, deviceID string, authCtx *pbCQRS.AuthorizationContext, online bool) error {
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
		ResourceId: &pbRA.ResourceId{
			DeviceId: deviceID,
			Href:     cloud.StatusHref,
		},
		Content: &pbRA.Content{
			ContentType:       message.AppOcfCbor.String(),
			CoapContentFormat: int32(message.AppOcfCbor),
			Data:              data,
		},
		Status: pbRA.Status_OK,
		CommandMetadata: &pbCQRS.CommandMetadata{
			Sequence:     client.coapConn.Sequence(),
			ConnectionId: client.remoteAddrString(),
		},
		AuthorizationContext: authCtx,
	}

	_, err = client.server.raClient.NotifyResourceChanged(ctx, &request)
	return err
}

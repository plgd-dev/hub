package service

import (
	"context"

	"github.com/go-ocf/kit/codec/cbor"

	gocoap "github.com/go-ocf/go-coap"
	cqrsRA "github.com/go-ocf/ocf-cloud/resource-aggregate/cqrs"
	pbCQRS "github.com/go-ocf/ocf-cloud/resource-aggregate/pb"
	pbRA "github.com/go-ocf/ocf-cloud/resource-aggregate/pb"
	"github.com/go-ocf/sdk/schema/cloud"
)

func (client *Client) PublishCloudDeviceStatus(ctx context.Context, deviceID string, authCtx pbCQRS.AuthorizationContext) error {
	devStatus := pbRA.Resource{
		Id:            cqrsRA.MakeResourceId(deviceID, cloud.StatusHref),
		Href:          cloud.StatusHref,
		ResourceTypes: cloud.StatusResourceTypes,
		Interfaces:    cloud.StatusInterfaces,
		DeviceId:      deviceID,
		Policies: &pbRA.Policies{
			BitFlags: 3,
		},
		Title: "Cloud device status",
	}
	_, err := client.publishResource(ctx, &devStatus, int32(0), client.remoteAddrString(), client.coapConn.Sequence(), authCtx)
	return err
}

func (client *Client) UpdateCloudDeviceStatus(ctx context.Context, deviceID string, authCtx pbCQRS.AuthorizationContext, online bool) error {
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
		ResourceId: cqrsRA.MakeResourceId(deviceID, cloud.StatusHref),
		Content: &pbRA.Content{
			ContentType:       gocoap.AppOcfCbor.String(),
			CoapContentFormat: int32(gocoap.AppOcfCbor),
			Data:              data,
		},
		CommandMetadata: &pbCQRS.CommandMetadata{
			Sequence:     client.coapConn.Sequence(),
			ConnectionId: client.remoteAddrString(),
		},
		AuthorizationContext: &authCtx,
	}

	_, err = client.server.raClient.NotifyResourceChanged(ctx, &request)
	return err
}

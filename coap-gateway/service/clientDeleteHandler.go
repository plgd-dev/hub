package service

import (
	"fmt"

	"github.com/plgd-dev/cloud/coap-gateway/coapconv"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/operations"
	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"google.golang.org/grpc/status"
)

func clientDeleteHandler(req *mux.Message, client *Client) {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle delete resource: %w", authCtx.GetDeviceID(), err), coapCodes.Unauthorized, req.Token)
		return
	}
	deviceID, href, err := URIToDeviceIDHref(req)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle delete resource: %w", authCtx.GetDeviceID(), err), coapCodes.BadRequest, req.Token)
		return
	}

	content, code, err := clientDeleteResourceHandler(req, client, deviceID, href, authCtx.GetUserID())
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot delete resource /%v%v from device: %w", authCtx.GetDeviceID(), deviceID, href, err), code, req.Token)
		return
	}

	if content == nil || len(content.Data) == 0 {
		client.sendResponse(code, req.Token, message.TextPlain, nil)
		return
	}
	mediaType, err := coapconv.MakeMediaType(-1, content.ContentType)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot delete resource /%v%v: %w", authCtx.GetDeviceID(), deviceID, href, err), code, req.Token)
		return
	}
	client.sendResponse(code, req.Token, mediaType, content.Data)
}

func clientDeleteResourceHandler(req *mux.Message, client *Client, deviceID, href, userID string) (*pbGRPC.Content, coapCodes.Code, error) {
	deleteCommand, err := coapconv.NewDeleteResourceRequest(commands.NewResourceID(deviceID, href), req, client.remoteAddrString())
	if err != nil {
		return nil, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.DELETE), err
	}

	operator := operations.New(client.server.resourceSubscriber, client.server.raClient)
	deletedCommand, err := operator.DeleteResource(req.Context, deleteCommand)
	if err != nil {
		return nil, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.DELETE), err
	}
	resp, err := pb.RAResourceDeletedEventToResponse(deletedCommand)
	if err != nil {
		return nil, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.DELETE), err
	}

	return resp.GetContent(), coapconv.StatusToCoapCode(pbGRPC.Status_OK, coapCodes.DELETE), nil
	/*
		processed, err := client.server.rdClient.DeleteResource(kitNetGrpc.CtxWithUserID(req.Context, userID), &pbGRPC.DeleteResourceRequest{
			ResourceId: &commands.ResourceId{
				DeviceId: deviceID,
				Href:     href,
			},
		})
		if err != nil {
			return nil, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.DELETE), err
		}
		return processed.GetContent(), coapconv.StatusToCoapCode(pbGRPC.Status_OK, coapCodes.DELETE), nil
	*/
}

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

func clientCreateHandler(req *mux.Message, client *Client) {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle create resource: %w", authCtx.GetDeviceID(), err), coapCodes.Unauthorized, req.Token)
		return
	}
	deviceID, href, err := URIToDeviceIDHref(req)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle create resource: %w", authCtx.GetDeviceID(), err), coapCodes.BadRequest, req.Token)
		return
	}

	content, code, err := clientCreateDeviceHandler(req, client, deviceID, href)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle create resource /%v%v: %w", authCtx.GetDeviceID(), deviceID, href, err), code, req.Token)
		return
	}
	if content == nil || len(content.Data) == 0 {
		client.sendResponse(code, req.Token, message.TextPlain, nil)
		return
	}
	mediaType, err := coapconv.MakeMediaType(-1, content.ContentType)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot encode response for create resource /%v%v: %w", authCtx.GetDeviceID(), deviceID, href, err), code, req.Token)
		return
	}
	client.sendResponse(code, req.Token, mediaType, content.Data)
}

func clientCreateDeviceHandler(req *mux.Message, client *Client, deviceID, href string) (*pbGRPC.Content, coapCodes.Code, error) {
	createCommand, err := coapconv.NewCreateResourceRequest(commands.NewResourceID(deviceID, href), req, client.remoteAddrString())
	if err != nil {
		return nil, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapconv.Create_Operation), err
	}

	operator := operations.New(client.server.resourceSubscriber, client.server.raClient)
	createdEvent, err := operator.CreateResource(req.Context, createCommand)
	if err != nil {
		return nil, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapconv.Create_Operation), err
	}
	resp, err := pb.RAResourceCreatedEventToResponse(createdEvent)
	if err != nil {
		return nil, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapconv.Create_Operation), err
	}

	return resp.GetContent(), coapconv.StatusToCoapCode(resp.Status, coapconv.Create_Operation), nil
}

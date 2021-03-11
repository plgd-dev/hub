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

//handles resource updates and creation
func clientPostHandler(req *mux.Message, client *Client) {
	resourceInterface := getResourceInterface(req)
	if resourceInterface == coapconv.OCFCreateInterface {
		clientCreateHandler(req, client)
		return
	}
	clientUpdateHandler(req, client)
}

func clientUpdateHandler(req *mux.Message, client *Client) {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle update resource: %w", authCtx.GetDeviceID(), err), coapCodes.Unauthorized, req.Token)
		return
	}
	deviceID, href, err := URIToDeviceIDHref(req)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle update resource: %w", authCtx.GetDeviceID(), err), coapCodes.BadRequest, req.Token)
		return
	}

	code := coapCodes.Changed
	content, err := clientUpdateDeviceHandler(req, client, deviceID, href)
	if err != nil {
		code = coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapconv.Update_Operation)
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle update resource /%v%v: %w", authCtx.GetDeviceID(), deviceID, href, err), code, req.Token)
		return
	}
	if content == nil || len(content.Data) == 0 {
		client.sendResponse(code, req.Token, message.TextPlain, nil)
		return
	}
	mediaType, err := coapconv.MakeMediaType(-1, content.ContentType)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot encode response for update resource /%v%v: %w", authCtx.GetDeviceID(), deviceID, href, err), code, req.Token)
		return
	}
	client.sendResponse(code, req.Token, mediaType, content.Data)
}

func clientUpdateDeviceHandler(req *mux.Message, client *Client, deviceID, href string) (*pbGRPC.Content, error) {
	updateCommand, err := coapconv.NewUpdateResourceRequest(commands.NewResourceID(deviceID, href), req, client.remoteAddrString())
	if err != nil {
		return nil, err
	}

	operator := operations.New(client.server.resourceSubscriber, client.server.raClient)
	updatedEvent, err := operator.UpdateResource(req.Context, updateCommand)
	if err != nil {
		return nil, err
	}
	resp, err := pb.RAResourceUpdatedEventToResponse(updatedEvent)
	if err != nil {
		return nil, err
	}

	return resp.GetContent(), nil
}

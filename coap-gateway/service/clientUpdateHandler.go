package service

import (
	"fmt"
	"time"

	"github.com/go-ocf/cloud/coap-gateway/coapconv"
	pbGRPC "github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/go-ocf/go-coap/v2/message"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/mux"
	"github.com/go-ocf/kit/log"
	"google.golang.org/grpc/status"
)

func clientUpdateHandler(s mux.ResponseWriter, req *mux.Message, client *Client) {
	t := time.Now()
	defer func() {
		log.Debugf("clientUpdateHandler takes %v", time.Since(t))
	}()

	authCtx := client.loadAuthorizationContext()
	deviceID, href, err := URIToDeviceIDHref(req)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle update resource: %w", authCtx.DeviceId, err), coapCodes.BadRequest, req.Token)
		return
	}

	content, code, err := clientUpdateDeviceHandler(req, client, deviceID)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle update resource /%v%v: %w", authCtx.DeviceId, deviceID, href, err), code, req.Token)
		return
	}
	if content == nil || len(content.Data) == 0 {
		client.sendResponse(code, req.Token, message.TextPlain, nil)
		return
	}
	mediaType, err := coapconv.MakeMediaType(-1, content.ContentType)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot encode response for update resource /%v%v: %w", authCtx.DeviceId, deviceID, href, err), code, req.Token)
		return
	}
	client.sendResponse(code, req.Token, mediaType, content.Data)
}

func clientUpdateDeviceHandler(req *mux.Message, client *Client, deviceID string) (*pbGRPC.Content, coapCodes.Code, error) {
	request, err := coapconv.MakeUpdateResourceRequest(deviceID, req)
	if err != nil {
		return nil, coapCodes.BadRequest, fmt.Errorf("cannot update resource of device %v: %w", deviceID, err)
	}

	resp, err := client.server.rdClient.UpdateResourcesValues(req.Context, request)
	if err != nil {
		return nil, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.POST), err
	}
	return resp.GetContent(), coapconv.StatusToCoapCode(resp.Status, coapCodes.POST), nil
}

package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-ocf/cloud/coap-gateway/coapconv"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	"github.com/go-ocf/go-coap/v2/message"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/mux"
	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/gofrs/uuid"
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
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle update resource: %w", authCtx.DeviceId, err), client, coapCodes.BadRequest, req.Token)
		return
	}

	resourceID := resource2UUID(deviceID, href)
	content, code, err := clientUpdateDeviceHandler(req, client, deviceID, resourceID)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle update resource /%v%v: %w", authCtx.DeviceId, deviceID, href, err), client, code, req.Token)
		return
	}
	if content == nil || len(content.Data) == 0 {
		sendResponse(client, code, req.Token, message.TextPlain, nil)
		return
	}
	mediaType, err := coapconv.MakeMediaType(content.CoapContentFormat, content.ContentType)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot encode response for update resource /%v%v: %w", authCtx.DeviceId, deviceID, href, err), client, code, req.Token)
		return
	}
	sendResponse(client, code, req.Token, mediaType, content.Data)
}

func clientUpdateDeviceHandler(req *mux.Message, client *Client, deviceID, resourceID string) (*pbRA.Content, coapCodes.Code, error) {
	authCtx := client.loadAuthorizationContext()
	correlationIDUUID, err := uuid.NewV4()
	if err != nil {
		return nil, coapCodes.InternalServerError, err
	}

	correlationID := correlationIDUUID.String()

	notify := client.server.updateNotificationContainer.Add(correlationID)
	defer client.server.updateNotificationContainer.Remove(correlationID)

	loaded, err := client.server.projection.Register(req.Context, deviceID)
	if err != nil {
		return nil, coapCodes.NotFound, fmt.Errorf("cannot register device to projection: %w", err)
	}
	defer client.server.projection.Unregister(deviceID)
	if !loaded {
		if len(client.server.projection.Models(deviceID, resourceID)) == 0 {
			err = client.server.projection.ForceUpdate(req.Context, deviceID, resourceID)
			if err != nil {
				return nil, coapCodes.NotFound, err
			}
		}
	}

	request := coapconv.MakeUpdateResourceRequest(resourceID, correlationID, authCtx.AuthorizationContext, client.remoteAddrString(), req)

	_, err = client.server.raClient.UpdateResource(kitNetGrpc.CtxWithToken(req.Context, authCtx.AccessToken), &request)
	if err != nil {
		return nil, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.POST), err
	}

	// first wait for notification
	timeoutCtx, cancel := context.WithTimeout(req.Context, client.server.RequestTimeout)
	defer cancel()
	select {
	case processed := <-notify:
		return processed.GetContent(), coapconv.StatusToCoapCode(processed.Status, coapCodes.POST), nil
	case <-timeoutCtx.Done():
		return nil, coapCodes.GatewayTimeout, fmt.Errorf("timeout")
	}
}

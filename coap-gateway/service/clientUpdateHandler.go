package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-ocf/ocf-cloud/coap-gateway/coapconv"
	gocoap "github.com/go-ocf/go-coap"
	coapCodes "github.com/go-ocf/go-coap/codes"
	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	pbRA "github.com/go-ocf/ocf-cloud/resource-aggregate/pb"
	"github.com/gofrs/uuid"
	"google.golang.org/grpc/status"
)

func clientUpdateHandler(s gocoap.ResponseWriter, req *gocoap.Request, client *Client) {
	t := time.Now()
	defer func() {
		log.Debugf("clientUpdateHandler takes %v", time.Since(t))
	}()

	authCtx := client.loadAuthorizationContext()
	deviceID, href, err := URIToDeviceIDHref(req.Msg)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle update resource: %w", authCtx.DeviceId, err), s, client, coapCodes.BadRequest)
		return
	}

	resourceID := resource2UUID(deviceID, href)
	content, code, err := clientUpdateDeviceHandler(req, client, deviceID, resourceID)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle update resource /%v%v: %w", authCtx.DeviceId, deviceID, href, err), s, client, code)
		return
	}
	if content == nil || len(content.Data) == 0 {
		sendResponse(s, client, code, gocoap.TextPlain, nil)
		return
	}
	mediaType, err := coapconv.MakeMediaType(content.CoapContentFormat, content.ContentType)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot encode response for update resource /%v%v: %w", authCtx.DeviceId, deviceID, href, err), s, client, code)
		return
	}
	sendResponse(s, client, code, mediaType, content.Data)
}

func clientUpdateDeviceHandler(req *gocoap.Request, client *Client, deviceID, resourceID string) (*pbRA.Content, coapCodes.Code, error) {
	authCtx := client.loadAuthorizationContext()
	correlationIDUUID, err := uuid.NewV4()
	if err != nil {
		return nil, coapCodes.InternalServerError, err
	}

	correlationID := correlationIDUUID.String()

	notify := client.server.updateNotificationContainer.Add(correlationID)
	defer client.server.updateNotificationContainer.Remove(correlationID)

	loaded, err := client.server.projection.Register(req.Ctx, deviceID)
	if err != nil {
		return nil, coapCodes.NotFound, fmt.Errorf("cannot register device to projection: %w", err)
	}
	defer client.server.projection.Unregister(deviceID)
	if !loaded {
		if len(client.server.projection.Models(deviceID, resourceID)) == 0 {
			err = client.server.projection.ForceUpdate(req.Ctx, deviceID, resourceID)
			if err != nil {
				return nil, coapCodes.NotFound, err
			}
		}
	}

	request := coapconv.MakeUpdateResourceRequest(resourceID, correlationID, authCtx.AuthorizationContext, req)

	_, err = client.server.raClient.UpdateResource(kitNetGrpc.CtxWithToken(req.Ctx, authCtx.AccessToken), &request)
	if err != nil {
		return nil, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.POST), err
	}

	// first wait for notification
	timeoutCtx, cancel := context.WithTimeout(req.Ctx, client.server.RequestTimeout)
	defer cancel()
	select {
	case processed := <-notify:
		return processed.GetContent(), coapconv.StatusToCoapCode(processed.Status, coapCodes.POST), nil
	case <-timeoutCtx.Done():
		return nil, coapCodes.GatewayTimeout, fmt.Errorf("timeout")
	}
}

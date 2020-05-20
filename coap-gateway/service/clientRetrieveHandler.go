package service

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-ocf/cloud/coap-gateway/coapconv"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRS "github.com/go-ocf/cloud/resource-directory/pb/resource-shadow"
	"github.com/go-ocf/go-coap/v2/message"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/mux"
	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/gofrs/uuid"
	"google.golang.org/grpc/status"
)

// URIToDeviceIDHref convert uri to deviceID and href. Expected input "/oic/route/{deviceID}/{Href}".
func URIToDeviceIDHref(msg *mux.Message) (deviceID, href string, err error) {
	wholePath, err := msg.Options.Path()
	if err != nil {
		return "", "", fmt.Errorf("cannot parse deviceID, href from uri: %w", err)
	}
	path := strings.Split(wholePath, "/")
	if len(path) < 4 {
		return "", "", fmt.Errorf("cannot parse deviceID, href from uri")
	}
	return path[2], fixHref(strings.Join(path[3:], "/")), nil
}

func getResourceInterface(msg *mux.Message) string {
	queries, _ := msg.Options.Queries()
	for _, query := range queries {
		if strings.HasPrefix(query, "if=") {
			return strings.TrimLeft(query, "if=")
		}
	}
	return ""
}

func clientRetrieveHandler(s mux.ResponseWriter, req *mux.Message, client *Client) {
	t := time.Now()
	defer func() {
		log.Debugf("clientRetrieveHandler takes %v", time.Since(t))
	}()
	authCtx := client.loadAuthorizationContext()

	deviceID, href, err := URIToDeviceIDHref(req)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle retrieve resource: %w", authCtx.DeviceId, err), client, coapCodes.BadRequest, req.Token)
		return
	}

	var content *pbRA.Content
	var code coapCodes.Code
	resourceInterface := getResourceInterface(req)
	resourceID := resource2UUID(deviceID, href)
	if resourceInterface == "" {
		content, code, err = clientRetrieveFromResourceShadowHandler(kitNetGrpc.CtxWithToken(req.Context, authCtx.AccessToken), client, resourceID)
		if err != nil {
			logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot retrieve resource /%v%v from resource shadow: %w", authCtx.DeviceId, deviceID, href, err), client, code, req.Token)
			return
		}
	} else {
		content, code, err = clientRetrieveFromDeviceHandler(req, client, deviceID, resourceID, resourceInterface)
		if err != nil {
			logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot retrieve resource /%v%v from device: %w", authCtx.DeviceId, deviceID, href, err), client, code, req.Token)
			return
		}
	}

	if content == nil || len(content.Data) == 0 {
		sendResponse(client, code, req.Token, message.TextPlain, nil)
		return
	}
	mediaType, err := coapconv.MakeMediaType(content.CoapContentFormat, content.ContentType)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot retrieve resource /%v%v: %w", authCtx.DeviceId, deviceID, href, err), client, code, req.Token)
		return
	}
	sendResponse(client, code, req.Token, mediaType, content.Data)
}

func clientRetrieveFromResourceShadowHandler(ctx context.Context, client *Client, resourceID string) (*pbRA.Content, coapCodes.Code, error) {
	authCtx := client.loadAuthorizationContext()
	retrieveResourcesValuesClient, err := client.server.rsClient.RetrieveResourcesValues(kitNetGrpc.CtxWithToken(ctx, authCtx.AccessToken), &pbRS.RetrieveResourcesValuesRequest{
		ResourceIdsFilter:    []string{resourceID},
		AuthorizationContext: &authCtx.AuthorizationContext,
	})
	if err != nil {
		return nil, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.GET), err
	}
	defer retrieveResourcesValuesClient.CloseSend()
	for {
		resourceValue, err := retrieveResourcesValuesClient.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.GET), err
		}
		if resourceValue.ResourceId == resourceID && resourceValue.Content != nil {
			return resourceValue.Content, coapCodes.Content, nil
		}
	}
	return nil, coapCodes.NotFound, fmt.Errorf("not found")
}

func clientRetrieveFromDeviceHandler(req *mux.Message, client *Client, deviceID, resourceID, resourceInterface string) (*pbRA.Content, coapCodes.Code, error) {
	authCtx := client.loadAuthorizationContext()
	correlationIDUUID, err := uuid.NewV4()
	if err != nil {
		return nil, coapCodes.InternalServerError, err
	}

	correlationID := correlationIDUUID.String()

	notify := client.server.retrieveNotificationContainer.Add(correlationID)
	defer client.server.retrieveNotificationContainer.Remove(correlationID)

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

	request := coapconv.MakeRetrieveResourceRequest(resourceID, resourceInterface, correlationID, authCtx.AuthorizationContext, client.remoteAddrString(), req)

	_, err = client.server.raClient.RetrieveResource(kitNetGrpc.CtxWithToken(req.Context, authCtx.AccessToken), &request)
	if err != nil {
		return nil, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.GET), err
	}

	// first wait for notification
	timeoutCtx, cancel := context.WithTimeout(req.Context, client.server.RequestTimeout)
	defer cancel()
	select {
	case processed := <-notify:
		return processed.GetContent(), coapconv.StatusToCoapCode(processed.Status, coapCodes.GET), nil
	case <-timeoutCtx.Done():
		return nil, coapCodes.GatewayTimeout, fmt.Errorf("timeout")
	}
}

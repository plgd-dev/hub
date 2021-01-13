package service

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/plgd-dev/cloud/coap-gateway/coapconv"
	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
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
	authCtx := client.loadAuthorizationContext()

	deviceID, href, err := URIToDeviceIDHref(req)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle retrieve resource: %w", authCtx.DeviceId, err), coapCodes.BadRequest, req.Token)
		return
	}

	var content *pbGRPC.Content
	var code coapCodes.Code
	resourceInterface := getResourceInterface(req)
	if resourceInterface == "" {
		content, code, err = clientRetrieveFromResourceShadowHandler(req.Context, client, deviceID, href)
		if err != nil {
			client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot retrieve resource /%v%v from resource shadow: %w", authCtx.DeviceId, deviceID, href, err), code, req.Token)
			return
		}
	} else {
		content, code, err = clientRetrieveFromDeviceHandler(req, client, deviceID, href, resourceInterface)
		if err != nil {
			client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot retrieve resource /%v%v from device: %w", authCtx.DeviceId, deviceID, href, err), code, req.Token)
			return
		}
	}

	if content == nil || len(content.Data) == 0 {
		client.sendResponse(code, req.Token, message.TextPlain, nil)
		return
	}
	mediaType, err := coapconv.MakeMediaType(-1, content.ContentType)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot retrieve resource /%v%v: %w", authCtx.DeviceId, deviceID, href, err), code, req.Token)
		return
	}
	client.sendResponse(code, req.Token, mediaType, content.Data)
}

func clientRetrieveFromResourceShadowHandler(ctx context.Context, client *Client, deviceID, href string) (*pbGRPC.Content, coapCodes.Code, error) {
	retrieveResourcesValuesClient, err := client.server.rdClient.RetrieveResourcesValues(ctx, &pbGRPC.RetrieveResourcesValuesRequest{
		ResourceIdsFilter: []*pbGRPC.ResourceId{
			{
				DeviceId: deviceID,
				Href:     href,
			},
		},
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
		if resourceValue.GetResourceId().GetDeviceId() == deviceID && resourceValue.GetResourceId().GetHref() == href && resourceValue.Content != nil {
			return resourceValue.Content, coapCodes.Content, nil
		}
	}
	return nil, coapCodes.NotFound, fmt.Errorf("not found")
}

func clientRetrieveFromDeviceHandler(req *mux.Message, client *Client, deviceID, href, resourceInterface string) (*pbGRPC.Content, coapCodes.Code, error) {
	authCtx := client.loadAuthorizationContext()
	processed, err := client.server.rdClient.RetrieveResourceFromDevice(kitNetGrpc.CtxWithUserID(req.Context, authCtx.GetUserID()), &pbGRPC.RetrieveResourceFromDeviceRequest{
		ResourceId: &pbGRPC.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		ResourceInterface: resourceInterface,
	})
	if err != nil {
		return nil, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.GET), err
	}
	return processed.GetContent(), coapconv.StatusToCoapCode(pbGRPC.Status_OK, coapCodes.GET), nil
}

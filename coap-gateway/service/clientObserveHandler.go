package service

import (
	"bytes"
	"fmt"
	"io"
	"time"

	pbAS "github.com/go-ocf/cloud/authorization/pb"
	"github.com/go-ocf/cloud/coap-gateway/coapconv"
	pbGRPC "github.com/go-ocf/cloud/grpc-gateway/pb"
	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	"github.com/go-ocf/go-coap/v2/message"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/mux"
	"github.com/go-ocf/go-coap/v2/tcp/message/pool"
	"github.com/go-ocf/kit/log"
	"google.golang.org/grpc/status"
)

func clientObserveHandler(s mux.ResponseWriter, req *mux.Message, client *Client, observe uint32) {
	t := time.Now()
	defer func() {
		log.Debugf("clientObserveHandler takes %v", time.Since(t))
	}()

	authCtx := client.loadAuthorizationContext()
	deviceID, href, err := URIToDeviceIDHref(req)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle observe resource: %v", authCtx.DeviceId, err), coapCodes.BadRequest, req.Token)
		return
	}
	resourceID := resource2UUID(deviceID, href)

	switch observe {
	case 0:
		startResourceObservation(s, req, client, authCtx, deviceID, resourceID)
	case 1:
		stopResourceObservation(s, req, client, authCtx, deviceID, resourceID)
	default:
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot observe resource %v.%v: invalid Observe value", authCtx.DeviceId, deviceID, resourceID), coapCodes.BadRequest, req.Token)
		return
	}

}

func cleanStartResourceObservation(client *Client, deviceID, resourceID string, token []byte) {
	err := client.server.projection.Unregister(deviceID)
	if err != nil {
		log.Errorf("DeviceId: %v: cannot start resource observation - unregister device from projection %v: %v", deviceID, err)
	}
	err = client.server.observeResourceContainer.RemoveByResource(resourceID, client.remoteAddrString(), token)
	if err != nil {
		log.Errorf("DeviceId: %v: cannot start resource observation - remove resource from observation %v: %v", resourceID, err)
	}
}

func SendResourceContentToObserver(client *Client, contentCtx *pbRA.ResourceChanged, observe uint32, deviceID, resourceID string, token message.Token) {
	if contentCtx.GetStatus() != pbRA.Status_OK {
		cleanStartResourceObservation(client, deviceID, resourceID, token)
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot observe resource %v, device response: %v", deviceID, resourceID, contentCtx.GetStatus()), coapconv.StatusToCoapCode(pbGRPC.RAStatus2Status(contentCtx.GetStatus()), coapCodes.GET), token)
		return
	}

	if contentCtx.GetContent() == nil {
		client.sendResponse(coapCodes.Content, token, message.TextPlain, nil)
		return
	}
	mediaType, err := coapconv.MakeMediaType(contentCtx.GetContent().GetCoapContentFormat(), contentCtx.GetContent().GetContentType())
	if err != nil {
		log.Errorf("cannot set content format for observer: %v", err)
		return
	}

	msg := pool.AcquireMessage(client.coapConn.Context())
	msg.SetCode(coapCodes.Content)
	msg.SetContentFormat(mediaType)
	msg.SetObserve(observe)
	msg.SetBody(bytes.NewReader(contentCtx.GetContent().GetData()))
	msg.SetToken(token)
	err = client.coapConn.WriteMessage(msg)
	if err != nil {
		log.Errorf("cannot send observe notification to %v: %v", client.remoteAddrString(), err)
	}
	decodeMsgToDebug(client, msg, "SEND-NOTIFICATION")
}

func startResourceObservation(s mux.ResponseWriter, req *mux.Message, client *Client, authCtx authCtx, deviceID, resourceID string) {
	userIdsFilter := []string(nil)
	if authCtx.UserID != "" {
		userIdsFilter = []string{authCtx.UserID}
	}
	getUserDevicesClient, err := client.server.asClient.GetUserDevices(req.Context, &pbAS.GetUserDevicesRequest{
		UserIdsFilter:   userIdsFilter,
		DeviceIdsFilter: []string{deviceID},
	})
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot start resource observation %v.%v: %v", authCtx.DeviceId, deviceID, resourceID, err), coapCodes.InternalServerError, req.Token)
		return
	}
	var found bool
	defer getUserDevicesClient.CloseSend()
	for {
		userDev, err := getUserDevicesClient.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot start resource observation %v.%v: %v", authCtx.DeviceId, deviceID, resourceID, err), coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.GET), req.Token)
			return
		}
		if userDev.DeviceId == deviceID {
			found = true
			break
		}
	}
	if !found {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot start resource observation %v.%v: unauthorized access", authCtx.DeviceId, deviceID, resourceID), coapCodes.BadRequest, req.Token)
		return
	}

	observeResource := observeResource{
		remoteAddr: client.remoteAddrString(),
		deviceID:   deviceID,
		resourceID: resourceID,
		token:      req.Token,
		client:     client,
	}

	err = client.server.observeResourceContainer.Add(&observeResource)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot start resource observation %v.%v: %v", authCtx.DeviceId, deviceID, resourceID, err), coapCodes.BadRequest, req.Token)
		return
	}

	loaded, err := client.server.projection.Register(req.Context, deviceID)
	if err != nil {
		err1 := client.server.observeResourceContainer.RemoveByResource(resourceID, client.remoteAddrString(), req.Token)
		if err1 != nil {
			log.Errorf("DeviceId: %v: cannot start resource observation - remove resource from observation %v: %v", resourceID, err1)
		}
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot start resource observation %v.%v: cannot register: %v", authCtx.DeviceId, deviceID, resourceID, err), coapCodes.BadRequest, req.Token)
		return
	}
	resourceCtxs := client.server.projection.Models(deviceID, resourceID)
	if len(resourceCtxs) == 0 {
		err := client.server.projection.ForceUpdate(req.Context, deviceID, resourceID)
		if err != nil {
			cleanStartResourceObservation(client, deviceID, resourceID, req.Token)
			client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot start resource observation %v.%v: force update: %v", authCtx.DeviceId, deviceID, resourceID, err), coapCodes.BadRequest, req.Token)
			return
		}
		resourceCtxs = client.server.projection.Models(deviceID, resourceID)
		if len(resourceCtxs) == 0 {
			cleanStartResourceObservation(client, deviceID, resourceID, req.Token)
			client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot start resource observation %v.%v: resource model: %v", authCtx.DeviceId, deviceID, resourceID, err), coapCodes.BadRequest, req.Token)
			return
		}
	}

	if !loaded {
		SendResourceContentToObserver(client, resourceCtxs[0].(*resourceCtx).Content(), observeResource.Observe(), deviceID, resourceID, req.Token)
		return
	}
	// response will be send from projection
}

func stopResourceObservation(s mux.ResponseWriter, req *mux.Message, client *Client, authCtx authCtx, deviceID, resourceID string) {
	err := client.server.observeResourceContainer.RemoveByResource(resourceID, client.remoteAddrString(), req.Token)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot stop resource observation %v.%v: %v", authCtx.DeviceId, deviceID, resourceID, err), coapCodes.BadRequest, req.Token)
		return
	}
	var content *pbRA.ResourceChanged
	resourceCtxs := client.server.projection.Models(deviceID, resourceID)
	if len(resourceCtxs) > 0 {
		content = resourceCtxs[0].(*resourceCtx).Content()
	} else {
		log.Errorf("DeviceId: %v: cannot get content for stop observation %v.%v: %v", authCtx.DeviceId, deviceID, resourceID, err)
	}

	err = client.server.projection.Unregister(deviceID)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot stop resource observation %v.%v: %v", authCtx.DeviceId, deviceID, resourceID, err), coapCodes.NotFound, req.Token)
		return
	}

	SendResourceContentToObserver(client, content, 1, deviceID, resourceID, req.Token)
}

func clientResetObservationHandler(s mux.ResponseWriter, req *mux.Message, client *Client, authCtx pbCQRS.AuthorizationContext) {
	observer, err := client.server.observeResourceContainer.PopByRemoteAddrToken(client.remoteAddrString(), req.Token)
	if err != nil {
		return
	}
	err = client.server.projection.Unregister(observer.deviceID)
	if err != nil {
		log.Errorf("DeviceId: %v: cannot reset resource observation %v.%v: %v", authCtx.DeviceId, observer.deviceID, observer.resourceID, err)
	}
}

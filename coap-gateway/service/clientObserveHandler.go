package service

import (
	"bytes"
	"fmt"
	"io"
	"time"

	pbAS "github.com/go-ocf/cloud/authorization/pb"
	"github.com/go-ocf/cloud/coap-gateway/coapconv"
	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	"github.com/go-ocf/go-coap/v2/message"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/mux"
	"github.com/go-ocf/go-coap/v2/tcp/message/pool"
	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"google.golang.org/grpc/status"
)

func clientObserveHandler(s mux.ResponseWriter, req *message.Message, client *Client, observe uint32) {
	t := time.Now()
	defer func() {
		log.Debugf("clientObserveHandler takes %v", time.Since(t))
	}()

	authCtx := client.loadAuthorizationContext()
	deviceId, href, err := URIToDeviceIDHref(req.Msg)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle observe resource: %v", authCtx.DeviceId, err), s, client, coapCodes.BadRequest)
		return
	}
	resourceId := resource2UUID(deviceId, href)

	switch observe {
	case 0:
		startResourceObservation(s, req, client, authCtx, deviceId, resourceId)
	case 1:
		stopResourceObservation(s, req, client, authCtx, deviceId, resourceId)
	default:
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot observe resource %v.%v: invalid Observe value", authCtx.DeviceId, deviceId, resourceId), s, client, coapCodes.BadRequest)
		return
	}

}

func cleanStartResourceObservation(client *Client, deviceId, resourceId string, token []byte) {
	err := client.server.projection.Unregister(deviceId)
	if err != nil {
		log.Errorf("DeviceId: %v: cannot start resource observation - unregister device from projection %v: %v", deviceId, err)
	}
	err = client.server.observeResourceContainer.RemoveByResource(resourceId, client.remoteAddrString(), token)
	if err != nil {
		log.Errorf("DeviceId: %v: cannot start resource observation - remove resource from observation %v: %v", resourceId, err)
	}
}

func SendResourceContentToObserver(s mux.ResponseWriter, client *Client, contentCtx *pbRA.ResourceChanged, observe uint32, deviceId, resourceId string, token []byte) {
	if contentCtx.GetStatus() != pbRA.Status_OK {
		cleanStartResourceObservation(client, deviceId, resourceId, token)
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot observe resource %v, device response: %v", deviceId, resourceId, contentCtx.GetStatus()), s, client, coapconv.StatusToCoapCode(contentCtx.GetStatus(), coapCodes.GET))
		return
	}

	if contentCtx.GetContent() == nil {
		sendResponse(s, client, coapCodes.Content, message.TextPlain, nil)
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
	err = client.coapConn.WriteMessage(msg)
	if err != nil {
		log.Errorf("cannot send observe notification to %v: %v", client.remoteAddrString(), err)
	}
	decodeMsgToDebug(client, msg, "SEND-NOTIFICATION")
}

func startResourceObservation(s mux.ResponseWriter, req *message.Message, client *Client, authCtx authCtx, deviceId, resourceId string) {
	userIdsFilter := []string(nil)
	if authCtx.UserId != "" {
		userIdsFilter = []string{authCtx.UserId}
	}
	getUserDevicesClient, err := client.server.asClient.GetUserDevices(kitNetGrpc.CtxWithToken(req.Ctx, authCtx.AccessToken), &pbAS.GetUserDevicesRequest{
		UserIdsFilter:   userIdsFilter,
		DeviceIdsFilter: []string{deviceId},
	})
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot start resource observation %v.%v: %v", authCtx.DeviceId, deviceId, resourceId, err), s, client, coapCodes.InternalServerError)
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
			logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot start resource observation %v.%v: %v", authCtx.DeviceId, deviceId, resourceId, err), s, client, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.GET))
			return
		}
		if userDev.DeviceId == deviceId {
			found = true
			break
		}
	}
	if !found {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot start resource observation %v.%v: unauthorized access", authCtx.DeviceId, deviceId, resourceId), s, client, coapCodes.BadRequest)
		return
	}

	observeResource := observeResource{
		remoteAddr:     client.remoteAddrString(),
		deviceId:       deviceId,
		resourceId:     resourceId,
		token:          req.Msg.Token(),
		client:         client,
		responseWriter: s,
	}

	err = client.server.observeResourceContainer.Add(observeResource)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot start resource observation %v.%v: %v", authCtx.DeviceId, deviceId, resourceId, err), s, client, coapCodes.BadRequest)
		return
	}

	loaded, err := client.server.projection.Register(req.Ctx, deviceId)
	if err != nil {
		err1 := client.server.observeResourceContainer.RemoveByResource(resourceId, client.remoteAddrString(), req.Msg.Token())
		if err1 != nil {
			log.Errorf("DeviceId: %v: cannot start resource observation - remove resource from observation %v: %v", resourceId, err1)
		}
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot start resource observation %v.%v: cannot register: %v", authCtx.DeviceId, deviceId, resourceId, err), s, client, coapCodes.BadRequest)
		return
	}
	resourceCtxs := client.server.projection.Models(deviceId, resourceId)
	if len(resourceCtxs) == 0 {
		err := client.server.projection.ForceUpdate(req.Ctx, deviceId, resourceId)
		if err != nil {
			cleanStartResourceObservation(client, deviceId, resourceId, req.Msg.Token())
			logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot start resource observation %v.%v: force update: %v", authCtx.DeviceId, deviceId, resourceId, err), s, client, coapCodes.BadRequest)
			return
		}
		resourceCtxs = client.server.projection.Models(deviceId, resourceId)
		if len(resourceCtxs) == 0 {
			cleanStartResourceObservation(client, deviceId, resourceId, req.Msg.Token())
			logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot start resource observation %v.%v: resource model: %v", authCtx.DeviceId, deviceId, resourceId, err), s, client, coapCodes.BadRequest)
			return
		}
	}

	if !loaded {
		SendResourceContentToObserver(s, client, resourceCtxs[0].(*resourceCtx).Content(), observeResource.Observe(), deviceId, resourceId, req.Msg.Token())
		return
	}
	// response will be send from projection
}

func stopResourceObservation(s mux.ResponseWriter, req *message.Message, client *Client, authCtx authCtx, deviceId, resourceId string) {
	err := client.server.observeResourceContainer.RemoveByResource(resourceId, client.remoteAddrString(), req.Msg.Token())
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot stop resource observation %v.%v: %v", authCtx.DeviceId, deviceId, resourceId, err), s, client, coapCodes.BadRequest)
		return
	}
	var content *pbRA.ResourceChanged
	resourceCtxs := client.server.projection.Models(deviceId, resourceId)
	if len(resourceCtxs) > 0 {
		content = resourceCtxs[0].(*resourceCtx).Content()
	} else {
		log.Errorf("DeviceId: %v: cannot get content for stop observation %v.%v: %v", authCtx.DeviceId, deviceId, resourceId, err)
	}

	err = client.server.projection.Unregister(deviceId)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot stop resource observation %v.%v: %v", authCtx.DeviceId, deviceId, resourceId, err), s, client, coapCodes.NotFound)
		return
	}

	SendResourceContentToObserver(s, client, content, 1, deviceId, resourceId, req.Msg.Token())
}

func clientResetObservationHandler(s mux.ResponseWriter, req *message.Message, client *Client, authCtx pbCQRS.AuthorizationContext) {
	observer, err := client.server.observeResourceContainer.PopByRemoteAddrToken(client.remoteAddrString(), req.Msg.Token())
	if err != nil {
		return
	}
	err = client.server.projection.Unregister(observer.deviceId)
	if err != nil {
		fmt.Errorf("DeviceId: %v: cannot reset resource observation %v.%v: %v", authCtx.DeviceId, observer.deviceId, observer.resourceId, err)
	}
}

package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/coap-gateway/coapconv"
	grpcClient "github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	"google.golang.org/grpc/status"
)

func clientObserveHandler(req *mux.Message, client *Client, observe uint32) {
	t := time.Now()
	defer func() {
		log.Debugf("clientObserveHandler takes %v", time.Since(t))
	}()

	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle observe resource: %w", authCtx.GetDeviceID(), err), coapCodes.Unauthorized, req.Token)
		return
	}
	deviceID, href, err := URIToDeviceIDHref(req)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle observe resource: %w", authCtx.GetDeviceID(), err), coapCodes.BadRequest, req.Token)
		return
	}

	switch observe {
	case 0:
		startResourceObservation(req, client, authCtx, deviceID, href)
	case 1:
		stopResourceObservation(req, client, authCtx, deviceID, href)
	default:
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot observe resource /%v%v: invalid Observe value", authCtx.GetDeviceID(), deviceID, href), coapCodes.BadRequest, req.Token)
		return
	}

}

func SendResourceContentToObserver(client *Client, resourceChanged *events.ResourceChanged, observe uint32, token message.Token) {
	msg := pool.AcquireMessage(client.coapConn.Context())
	msg.SetCode(coapCodes.Content)
	msg.SetObserve(observe)
	msg.SetToken(token)
	if resourceChanged.GetContent() != nil {
		mediaType, err := coapconv.MakeMediaType(-1, resourceChanged.GetContent().GetContentType())
		if err != nil {
			log.Errorf("cannot set content format for observer: %v", err)
			return
		}
		msg.SetContentFormat(mediaType)
		msg.SetBody(bytes.NewReader(resourceChanged.GetContent().GetData()))
	}
	err := client.coapConn.WriteMessage(msg)
	if err != nil {
		log.Errorf("cannot send observe notification to %v: %v", client.remoteAddrString(), err)
	}
	decodeMsgToDebug(client, msg, "SEND-NOTIFICATION")
}

func startResourceObservation(req *mux.Message, client *Client, authCtx *authorizationContext, deviceID, href string) {
	userIdsFilter := []string(nil)
	if authCtx.GetUserID() != "" {
		userIdsFilter = []string{authCtx.GetUserID()}
	}
	getUserDevicesClient, err := client.server.asClient.GetUserDevices(req.Context, &pbAS.GetUserDevicesRequest{
		UserIdsFilter:   userIdsFilter,
		DeviceIdsFilter: []string{deviceID},
	})
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot start resource observation /%v%v: %w", authCtx.GetDeviceID(), deviceID, href, err), coapCodes.InternalServerError, req.Token)
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
			client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot start resource observation /%v%v: %w", authCtx.GetDeviceID(), deviceID, href, err), coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapconv.Retrieve), req.Token)
			return
		}
		if userDev.DeviceId == deviceID {
			found = true
			break
		}
	}
	if !found {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot start resource observation /%v%v: unauthorized access", authCtx.GetDeviceID(), deviceID, href), coapCodes.Unauthorized, req.Token)
		return
	}
	token := req.Token.String()
	client.cancelResourceSubscription(token, true)

	seqNum := uint32(2)
	h := resourceSubscriptionHandlers{
		onChange: func(ctx context.Context, resourceChanged *events.ResourceChanged) error {
			if resourceChanged.GetStatus() != commands.Status_OK {
				client.cancelResourceSubscription(token, false)
				client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot observe resource /%v%v, device response: %v", authCtx.GetDeviceID(), deviceID, href, resourceChanged.GetStatus()), coapconv.StatusToCoapCode(resourceChanged.GetStatus(), coapconv.Retrieve), req.Token)
				return nil
			}
			SendResourceContentToObserver(client, resourceChanged, seqNum, req.Token)
			seqNum++
			return nil
		},
		onError: func(err error) {
			client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot observe resource /%v%v, device response: %w", authCtx.GetDeviceID(), deviceID, href, err), coapCodes.Unauthorized, req.Token)
			client.resourceSubscriptions.Delete(token)
		},
		onClose: func() {
			log.Debugf("resource /%v%v subscription(ResourceUpdatePending, ResourceRetrievePending) was closed", deviceID, href)
			client.resourceSubscriptions.Delete(token)
		},
	}

	sub, err := grpcClient.NewResourceSubscription(req.Context, commands.NewResourceID(deviceID, href), &h, &h, client.server.rdClient)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot observe resource /%v%v: %w", authCtx.GetDeviceID(), deviceID, href, err), coapCodes.BadRequest, req.Token)
		return
	}

	_, loaded := client.resourceSubscriptions.LoadOrStore(token, sub)
	if loaded {
		sub.Cancel()
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot observe resource /%v%v: resource subscription with token %v already exist", authCtx.GetDeviceID(), deviceID, href, token), coapCodes.BadRequest, req.Token)
		return
	}

	// response will be send from projection
}

func stopResourceObservation(req *mux.Message, client *Client, authCtx *authorizationContext, deviceID, href string) {
	token := req.Token.String()
	cancelled, err := client.cancelResourceSubscription(token, true)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot stop resource observation /%v%v: %w", authCtx.GetDeviceID(), deviceID, href, err), coapCodes.BadRequest, req.Token)
		return
	}
	if !cancelled {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot stop resource observation /%v%v: subscription not found", authCtx.GetDeviceID(), deviceID, href), coapCodes.BadRequest, req.Token)
		return
	}
	SendResourceContentToObserver(client, nil, 1, req.Token)
}

func clientResetObservationHandler(req *mux.Message, client *Client) {
	token := req.Token.String()
	cancelled, err := client.cancelResourceSubscription(token, true)
	if err != nil {
		log.Errorf("cannot reset resource observation: %v", err)
		return
	}
	if !cancelled {
		log.Errorf("cannot reset resource observation: not found")
		return
	}
}

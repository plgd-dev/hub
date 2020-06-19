package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	pbAS "github.com/go-ocf/cloud/authorization/pb"
	"github.com/go-ocf/cloud/coap-gateway/coapconv"
	grpcClient "github.com/go-ocf/cloud/grpc-gateway/client"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	pbGRPC "github.com/go-ocf/cloud/grpc-gateway/pb"
	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
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

	switch observe {
	case 0:
		startResourceObservation(s, req, client, authCtx, deviceID, href)
	case 1:
		stopResourceObservation(s, req, client, authCtx, deviceID, href)
	default:
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot observe resource /%v%v: invalid Observe value", authCtx.DeviceId, deviceID, href), coapCodes.BadRequest, req.Token)
		return
	}

}

func SendResourceContentToObserver(client *Client, resourceChanged *pb.Event_ResourceChanged, observe uint32, token message.Token) {
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

func startResourceObservation(s mux.ResponseWriter, req *mux.Message, client *Client, authCtx authCtx, deviceID, href string) {
	userIdsFilter := []string(nil)
	if authCtx.UserID != "" {
		userIdsFilter = []string{authCtx.UserID}
	}
	getUserDevicesClient, err := client.server.asClient.GetUserDevices(req.Context, &pbAS.GetUserDevicesRequest{
		UserIdsFilter:   userIdsFilter,
		DeviceIdsFilter: []string{deviceID},
	})
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot start resource observation /%v%v: %v", authCtx.DeviceId, deviceID, href, err), coapCodes.InternalServerError, req.Token)
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
			client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot start resource observation /%v%v: %v", authCtx.DeviceId, deviceID, href, err), coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.GET), req.Token)
			return
		}
		if userDev.DeviceId == deviceID {
			found = true
			break
		}
	}
	if !found {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot start resource observation /%v%v: unauthorized access", authCtx.DeviceId, deviceID, href), coapCodes.Unauthorized, req.Token)
		return
	}
	token := req.Token.String()
	client.cancelResourceSubscription(token, true)

	seqNum := uint32(2)
	h := resourceSubscriptionHandlers{
		onChange: func(ctx context.Context, resourceChanged *pb.Event_ResourceChanged) error {
			if resourceChanged.GetStatus() != pbGRPC.Status_OK {
				client.cancelResourceSubscription(token, false)
				client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot observe resource /%v%v, device response: %v", authCtx.DeviceId, deviceID, href, resourceChanged.GetStatus()), coapconv.StatusToCoapCode(resourceChanged.GetStatus(), coapCodes.GET), req.Token)
				return nil
			}
			SendResourceContentToObserver(client, resourceChanged, seqNum, req.Token)
			seqNum++
			return nil
		},
		onError: func(err error) {
			client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot observe resource /%v%v, device response: %w", authCtx.DeviceId, deviceID, href, err), coapCodes.Unauthorized, req.Token)
			client.resourceSubscriptions.Delete(token)
		},
		onClose: func() {
			log.Debugf("resource /%v%v subscription(ResourceUpdatePending, ResourceRetrievePending) was closed", deviceID, href)
			client.resourceSubscriptions.Delete(token)
		},
	}

	sub, err := grpcClient.NewResourceSubscription(req.Context, pbGRPC.ResourceId{
		DeviceId: deviceID,
		Href:     href,
	}, &h, &h, client.server.rdClient)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot observe resource /%v%v: %v", authCtx.DeviceId, deviceID, href, err), coapCodes.BadRequest, req.Token)
		return
	}

	_, loaded := client.resourceSubscriptions.LoadOrStore(token, sub)
	if loaded {
		sub.Cancel()
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot observe resource /%v%v: resource subscription with token %v already exist", authCtx.DeviceId, deviceID, href, token), coapCodes.BadRequest, req.Token)
		return
	}

	// response will be send from projection
}

func stopResourceObservation(s mux.ResponseWriter, req *mux.Message, client *Client, authCtx authCtx, deviceID, href string) {
	token := req.Token.String()
	cancelled, err := client.cancelResourceSubscription(token, true)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot stop resource observation /%v%v: %v", authCtx.DeviceId, deviceID, href, err), coapCodes.BadRequest, req.Token)
		return
	}
	if !cancelled {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot stop resource observation /%v%v: subscription not found", authCtx.DeviceId, deviceID, href), coapCodes.BadRequest, req.Token)
		return
	}
	SendResourceContentToObserver(client, nil, 1, req.Token)
}

func clientResetObservationHandler(s mux.ResponseWriter, req *mux.Message, client *Client, authCtx pbCQRS.AuthorizationContext) {
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

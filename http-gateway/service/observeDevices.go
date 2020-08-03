package service

import (
	"context"
	"net/http"

	"github.com/go-ocf/cloud/grpc-gateway/client"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/gorilla/websocket"
)

type DeviceEvent struct {
	DeviceIDs []string `json:"deviceIds"`
	Status    string   `json:"status"`
}

func (requestHandler *RequestHandler) startDevicesObservation(w http.ResponseWriter, r *http.Request) {
	resolver := deviceObservationResolver{requestHandler: requestHandler}
	err := requestHandler.ServeWs(w, r, &resolver)
	if err != nil {
		writeError(w, err)
		return
	}
}

type deviceObservationResolver struct {
	requestHandler *RequestHandler
}

func (d *deviceObservationResolver) StartObservation(r *http.Request, ws *websocket.Conn) (SubscribeSession, error) {
	ob := deviceObservation{
		NewSubscriptionSession(ws),
	}
	ctx := kitNetGrpc.CtxWithToken(context.Background(), getAccessToken(r.Header))
	id, err := d.requestHandler.client.ObserveDevices(ctx, &ob)
	if err != nil {
		return nil, err
	}
	ob.SetSubscriptionId(id)
	return &ob, nil
}

func (d *deviceObservationResolver) StopObservation(subscriptionID string) error {
	err := d.requestHandler.client.StopObservingResource(context.Background(), subscriptionID)
	return err
}

type deviceObservation struct {
	subscribeSession
}

func ToDevicesObservationEvent(e client.DevicesObservationEvent_type) string {
	switch e {
	case client.DevicesObservationEvent_ONLINE:
		return "online"
	case client.DevicesObservationEvent_OFFLINE:
		return "offline"
	case client.DevicesObservationEvent_REGISTERED:
		return "registered"
	case client.DevicesObservationEvent_UNREGISTERED:
		return "unregistered"
	}
	return ""
}

func (d *deviceObservation) Handle(ctx context.Context, event client.DevicesObservationEvent) error {
	if len(event.DeviceIDs) == 0 {
		return nil
	}
	evt := DeviceEvent{
		DeviceIDs: event.DeviceIDs,
		Status:    ToDevicesObservationEvent(event.Event),
	}
	d.Write(evt)
	return nil
}

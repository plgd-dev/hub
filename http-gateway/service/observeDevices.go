package service

import (
	"context"
	"net/http"

	"github.com/go-ocf/sdk/backend"

	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/gorilla/websocket"
)

type DeviceEvent struct {
	DeviceId string `json:"deviceId"`
	Status   string `json:"status"`
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

func ToDevicesObservationEvent(e backend.DevicesObservationEvent_type) string {
	switch e {
	case backend.DevicesObservationEvent_ONLINE:
		return "online"
	case backend.DevicesObservationEvent_OFFLINE:
		return "offline"
	case backend.DevicesObservationEvent_REGISTERED:
		return "registered"
	case backend.DevicesObservationEvent_UNREGISTERED:
		return "unregistered"
	}
	return ""
}

func (d *deviceObservation) Handle(ctx context.Context, event backend.DevicesObservationEvent) error {
	if event.Event == backend.DevicesObservationEvent_REGISTERED ||
		event.Event == backend.DevicesObservationEvent_UNREGISTERED {
		return nil
	}
	evt := DeviceEvent{
		DeviceId: event.DeviceID,
		Status:   ToDevicesObservationEvent(event.Event),
	}
	d.Write(evt)
	return nil
}

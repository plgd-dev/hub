package service

import (
	"context"
	"net/http"

	"github.com/go-ocf/cloud/grpc-gateway/client"
	"github.com/go-ocf/cloud/http-gateway/uri"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/sdk/schema"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type DeviceResourceObservationEvent struct {
	Resource schema.ResourceLink `json:"resource"`
	Event    string              `json:"event"`
}

func (requestHandler *RequestHandler) startDeviceResourcesObservation(w http.ResponseWriter, r *http.Request) {
	resolver := deviceResourcesObservationResolver{requestHandler: requestHandler}
	err := requestHandler.ServeWs(w, r, &resolver)
	if err != nil {
		writeError(w, err)
		return
	}
}

type deviceResourcesObservationResolver struct {
	requestHandler *RequestHandler
}

func (d *deviceResourcesObservationResolver) StartObservation(r *http.Request, ws *websocket.Conn) (SubscribeSession, error) {
	ob := deviceResourcesObservation{
		NewSubscriptionSession(ws),
	}
	vars := mux.Vars(r)
	ctx := kitNetGrpc.CtxWithToken(context.Background(), getAccessToken(r.Header))
	id, err := d.requestHandler.client.ObserveDeviceResources(ctx, vars[uri.DeviceIDKey], &ob)
	if err != nil {
		return nil, err
	}
	ob.subscriptionId = id
	return &ob, nil
}

func (d *deviceResourcesObservationResolver) StopObservation(subscriptionID string) error {
	err := d.requestHandler.client.StopObservingResource(context.Background(), subscriptionID)
	return err
}

type deviceResourcesObservation struct {
	subscribeSession
}

func ToDeviceResourcesObservationEvent(e client.DeviceResourcesObservationEvent_type) string {
	switch e {
	case client.DeviceResourcesObservationEvent_ADDED:
		return "added"
	case client.DeviceResourcesObservationEvent_REMOVED:
		return "removed"
	}
	return ""
}

func (d *deviceResourcesObservation) Handle(ctx context.Context, event client.DeviceResourcesObservationEvent) error {
	evt := DeviceResourceObservationEvent{
		Resource: schema.ResourceLink{
			ResourceTypes: event.Link.GetTypes(),
			Interfaces:    event.Link.GetInterfaces(),
			Href:          event.Link.GetHref(),
			DeviceID:      event.Link.GetDeviceId(),
		},
		Event: ToDeviceResourcesObservationEvent(event.Event),
	}
	d.Write(evt)
	return nil
}

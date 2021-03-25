package service

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/sdk/schema"
)

type DeviceResourceObservationEvent struct {
	Resources []schema.ResourceLink `json:"resources"`
	Event     string                `json:"event"`
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

func (d *deviceResourcesObservationResolver) StartObservation(r *http.Request, ws *websocket.Conn, accessToken string) (SubscribeSession, error) {
	ob := deviceResourcesObservation{
		NewSubscriptionSession(ws),
	}
	vars := mux.Vars(r)
	ctx := kitNetGrpc.CtxWithToken(context.Background(), accessToken)
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
	links := make([]schema.ResourceLink, 0, 32)
	for _, l := range event.Links {
		links = append(links, l.ToSchema())
	}
	evt := DeviceResourceObservationEvent{
		Resources: links,
		Event:     ToDeviceResourcesObservationEvent(event.Event),
	}
	d.Write(evt)
	return nil
}

package service

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
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

func (d *deviceResourcesObservation) HandleResourcePublished(ctx context.Context, val *events.ResourceLinksPublished) error {
	evt := DeviceResourceObservationEvent{
		Resources: toResourceLinks(val.GetResources()),
		Event:     "added",
	}
	d.Write(evt)
	return nil
}

func (d *deviceResourcesObservation) HandleResourceUnpublished(ctx context.Context, val *events.ResourceLinksUnpublished) error {
	links := make([]schema.ResourceLink, 0, 32)
	for _, href := range val.GetHrefs() {
		links = append(links, schema.ResourceLink{
			DeviceID: val.GetDeviceId(),
			Href:     href,
		})
	}
	evt := DeviceResourceObservationEvent{
		Resources: links,
		Event:     "removed",
	}
	d.Write(evt)
	return nil
}

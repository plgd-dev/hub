package service

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/kit/codec/json"
	"github.com/plgd-dev/kit/log"
	kitNetCoap "github.com/plgd-dev/kit/net/coap"
)

func (requestHandler *RequestHandler) startResourceObservation(w http.ResponseWriter, r *http.Request) {
	resolver := resourceObservationResolver{requestHandler: requestHandler}
	err := requestHandler.ServeWs(w, r, resolver)
	if err != nil {
		writeError(w, err)
		return
	}
}

type resourceObservationResolver struct {
	requestHandler *RequestHandler
}

func (d resourceObservationResolver) StartObservation(r *http.Request, ws *websocket.Conn, accessToken string) (SubscribeSession, error) {
	ob := resourceObservation{
		NewSubscriptionSession(ws),
	}
	vars := mux.Vars(r)
	ctx := kitNetGrpc.CtxWithToken(context.Background(), accessToken)
	id, err := d.requestHandler.client.ObserveResource(ctx, vars[uri.DeviceIDKey], vars[uri.HrefKey], &ob)
	if err != nil {
		return nil, err
	}
	ob.subscriptionId = id
	return &ob, nil
}

func (d resourceObservationResolver) StopObservation(subscriptionID string) error {
	err := d.requestHandler.client.StopObservingResource(context.Background(), subscriptionID)
	return err
}

type resourceObservation struct {
	subscribeSession
}

func (d *resourceObservation) Handle(ctx context.Context, body kitNetCoap.DecodeFunc) {
	var evt interface{}
	if err := body(&evt); err != nil {
		d.ws.WriteJSON(errToJsonRes(err))
		return
	}
	log.Debug("send resource data: %s", evt)
	bytes, err := json.Encode(evt)
	if err != nil {
		d.ws.WriteJSON(errToJsonRes(err))
		return
	}
	d.Write(bytes)
}

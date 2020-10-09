package service

import (
	"fmt"
	"net/http"

	"github.com/plgd-dev/sdk/schema"

	"github.com/gorilla/mux"
	"github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/uri"
)

func (requestHandler *RequestHandler) getDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ctx := requestHandler.makeCtx(r)

	sdkDevice, err := requestHandler.client.GetDevice(ctx, vars[uri.DeviceIDKey])
	if err != nil {
		writeError(w, fmt.Errorf("cannot get device: %w", err))
		return
	}

	jsonResponseWriter(w, mapToDevice(sdkDevice))
}

type Status string

const Status_ONLINE Status = "online"
const Status_OFFLINE Status = "offline"

func toStatus(isOnline bool) Status {
	if isOnline {
		return "online"
	}
	return "offline"
}

type Device struct {
	Device schema.Device `json:"device"`
	Status Status        `json:"status"`
}

type RetrieveDeviceWithLinksResponse struct {
	Device
	Links []schema.ResourceLink `json:"links"`
}

func toResourceLinks(s []*pb.ResourceLink) []schema.ResourceLink {
	r := make([]schema.ResourceLink, 0, 16)
	for _, v := range s {
		r = append(r, v.ToSchema())
	}
	return r
}

func mapToDevice(d client.DeviceDetails) RetrieveDeviceWithLinksResponse {
	return RetrieveDeviceWithLinksResponse{
		Device: Device{
			Device: d.Device.ToSchema(),
			Status: toStatus(d.Device.GetIsOnline()),
		},
		Links: toResourceLinks(d.Resources),
	}
}

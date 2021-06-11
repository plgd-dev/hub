package service

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/plgd-dev/sdk/schema"

	"github.com/gorilla/mux"
	"github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
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

type ConnectionStatus struct {
	Value      string `json:"value"`
	ValidUntil int64  `json:"validUntil"`
}

func toConnectionStatus(s *commands.ConnectionStatus) ConnectionStatus {
	connStatusVal := strings.ToLower(s.GetValue().String())
	if connStatusVal == "" {
		connStatusVal = strings.ToLower(commands.ConnectionStatus_OFFLINE.String())
	}
	return ConnectionStatus{
		Value:      connStatusVal,
		ValidUntil: s.GetValidUntil(),
	}
}

type ShadowSynchronization struct {
	Disabled bool `json:"disabled"`
}

func toShadowSynchronization(s *commands.ShadowSynchronization) ShadowSynchronization {
	return ShadowSynchronization{
		Disabled: s.GetDisabled(),
	}
}

type Metadata struct {
	ConnectionStatus      ConnectionStatus      `json:"connectionStatus"`
	ShadowSynchronization ShadowSynchronization `json:"shadowSynchronization"`
}

func toMetadata(m *pb.Device_Metadata) Metadata {
	return Metadata{
		ConnectionStatus:      toConnectionStatus(m.GetStatus()),
		ShadowSynchronization: toShadowSynchronization(m.GetShadowSynchronization()),
	}
}

type Device struct {
	Device   schema.Device `json:"device"`
	Metadata Metadata      `json:"metadata"`
}

type RetrieveDeviceWithLinksResponse struct {
	Device
	Links []schema.ResourceLink `json:"links,omitempty"`
}

func toResourceLinks(s []*commands.Resource) []schema.ResourceLink {
	r := make([]schema.ResourceLink, 0, 16)
	for _, v := range s {
		r = append(r, v.ToSchema())
	}
	return r
}

func mapToDevice(d *client.DeviceDetails) RetrieveDeviceWithLinksResponse {
	return RetrieveDeviceWithLinksResponse{
		Device: Device{
			Device:   d.Device.ToSchema(),
			Metadata: toMetadata(d.Device.GetMetadata()),
		},
		Links: toResourceLinks(d.Resources),
	}
}

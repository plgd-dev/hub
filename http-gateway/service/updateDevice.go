package service

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/plgd-dev/kit/codec/json"
)

type UpdateDevice struct {
	ShadowSynchronization ShadowSynchronization `json:"shadowSynchronization"`
}

func (requestHandler *RequestHandler) updateDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ctx := requestHandler.makeCtx(r)

	correlationUUID := r.Header.Get(uri.CorrelationIDHeader)
	if correlationUUID == "" {
		writeError(w, status.Errorf(codes.InvalidArgument, "'%v' not found in header", uri.CorrelationIDHeader))
		return
	}

	var body UpdateDevice
	if err := json.ReadFrom(r.Body, &body); err != nil {
		writeError(w, status.Errorf(codes.InvalidArgument, "invalid json body: %v", err))
		return
	}

	_, err := requestHandler.raClient.UpdateDeviceMetadata(ctx, &commands.UpdateDeviceMetadataRequest{
		DeviceId: vars[uri.DeviceIDKey],
		Update: &commands.UpdateDeviceMetadataRequest_ShadowSynchronization{
			ShadowSynchronization: &commands.ShadowSynchronization{
				Disabled: body.ShadowSynchronization.Disabled,
			},
		},
		CorrelationId: correlationUUID,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: r.RemoteAddr,
		},
	})
	if err != nil {
		writeError(w, fmt.Errorf("cannot update device metadata: %w", err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

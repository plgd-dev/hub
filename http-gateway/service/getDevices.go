package service

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	"github.com/plgd-dev/kit/grpcconv"
	grpcStatus "google.golang.org/grpc/status"
)

func (requestHandler *RequestHandler) getDevices(w http.ResponseWriter, r *http.Request) {
	var typeFilter []string
	typeFilterString := r.URL.Query().Get(uri.TypeFilterQueryKey)
	if typeFilterString != "" {
		typeFilter = strings.Split(strings.ReplaceAll(typeFilterString, " ", ","), ",")
	}
	ctx := requestHandler.makeCtx(r)

	sdkDevices, err := requestHandler.client.GetDevices(ctx, client.WithResourceTypes(typeFilter...))
	if err != nil {
		if IsMappedTo(err, http.StatusNotFound) {
			jsonResponseWriter(w, []Device{})
			return
		}
		writeError(w, fmt.Errorf("cannot get devices: %w", err))
		return
	}

	devices := make([]RetrieveDeviceWithLinksResponse, len(sdkDevices))
	var i int
	for _, sdkDevice := range sdkDevices {
		devices[i] = mapToDevice(sdkDevice)
		i++
	}

	jsonResponseWriter(w, devices)
}

type grpcErr interface {
	GRPCStatus() *grpcStatus.Status
}

func IsMappedTo(err error, code int) bool {
	var grpcErr grpcErr
	if errors.As(err, &grpcErr) {
		return grpcconv.ToHTTPCode(grpcErr.GRPCStatus().Code(), code) == code
	}
	return false
}

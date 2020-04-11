package service

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-ocf/sdk/backend"

	"github.com/go-ocf/cloud/http-gateway/uri"
	"github.com/go-ocf/kit/grpcconv"
	grpcStatus "google.golang.org/grpc/status"
)

func (requestHandler *RequestHandler) getDevices(w http.ResponseWriter, r *http.Request) {
	var typeFilter []string
	typeFilterString := r.URL.Query().Get(uri.TypeFilterQueryKey)
	if typeFilterString != "" {
		typeFilter = strings.Split(strings.ReplaceAll(typeFilterString, " ", ","), ",")
	}
	ctx, cancel := requestHandler.makeCtx(r)
	defer cancel()

	sdkDevices, err := requestHandler.client.GetDevices(ctx, backend.WithResourceTypes(typeFilter...))
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

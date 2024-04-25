package service

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/google/go-querystring/query"
	"github.com/gorilla/mux"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
)

const errFmtFromTwin = "cannot get resource('%v') from twin: %w"

func getHeaderETag(r *http.Request) string {
	etagStr := r.Header.Get(uri.ETagHeaderKey)
	if etagStr != "" {
		return etagStr
	}
	return r.Header.Get(strings.ToLower(uri.ETagHeaderKey))
}

func getETags(r *http.Request) [][]byte {
	etagStr := getHeaderETag(r)
	if etagStr != "" {
		if etag, err := base64.StdEncoding.DecodeString(etagStr); err == nil {
			return [][]byte{etag}
		}
	}
	etagQ := r.URL.Query()[uri.ETagQueryKey]
	if len(etagQ) == 0 {
		return nil
	}
	etags := make([][]byte, 0, len(etagQ))
	for _, etag := range etagQ {
		e, err := base64.StdEncoding.DecodeString(etag)
		if err == nil {
			etags = append(etags, e)
		}
	}
	return etags
}

func (requestHandler *RequestHandler) getResourceFromTwin(r *http.Request, resourceID *pb.ResourceIdFilter) (*httptest.ResponseRecorder, error) {
	type Options struct {
		ResourceIDFilter []string `url:"httpResourceIdFilter"`
	}
	opt := Options{
		ResourceIDFilter: []string{resourceID.ToString()},
	}

	v, err := query.Values(opt)
	if err != nil {
		return nil, err
	}
	r.URL.Path = uri.Resources
	r.URL.RawQuery = v.Encode()
	rec := httptest.NewRecorder()
	requestHandler.mux.ServeHTTP(rec, r)
	return rec, nil
}

func (requestHandler *RequestHandler) serveResourceRequest(r *http.Request, deviceID, resourceHref, twin, resourceInterface string) (*httptest.ResponseRecorder, error) {
	resourceID := pb.ResourceIdFilter{
		ResourceId: commands.NewResourceID(deviceID, resourceHref),
		Etag:       getETags(r),
	}

	if (twin == "" || strings.ToLower(twin) == "true") && resourceInterface == "" {
		rec, err := requestHandler.getResourceFromTwin(r, &resourceID)
		if err != nil {
			return nil, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, errFmtFromTwin, &resourceID, err)
		}
		return rec, nil
	}

	query := r.URL.Query()
	if !query.Has(uri.ETagQueryKey) {
		etag := getHeaderETag(r)
		if etag != "" {
			query.Set(uri.ETagQueryKey, etag)
		}
		r.URL.RawQuery = query.Encode()
	}

	rec := httptest.NewRecorder()
	requestHandler.mux.ServeHTTP(rec, r)
	return rec, nil
}

func (requestHandler *RequestHandler) getResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars[uri.DeviceIDKey]
	resourceHref := vars[uri.ResourceHrefKey]
	twin := r.URL.Query().Get(uri.TwinQueryKey)
	resourceInterface := r.URL.Query().Get(uri.ResourceInterfaceQueryKey)
	rec, err := requestHandler.serveResourceRequest(r, deviceID, resourceHref, twin, resourceInterface)
	if err != nil {
		serverMux.WriteError(w, err)
		return
	}
	toSimpleResponse(w, rec, func(w http.ResponseWriter, err error) {
		serverMux.WriteError(w, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot get resource('%v/%v') from the device: %v", deviceID, resourceHref, err))
	}, streamResponseKey)
}

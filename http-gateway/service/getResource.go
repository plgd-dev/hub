package service

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/google/go-querystring/query"
	"github.com/gorilla/mux"
	"github.com/plgd-dev/device/v2/pkg/codec/json"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
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

func parseBoolQuery(str string) bool {
	val, err := strconv.ParseBool(str)
	if err != nil {
		return false
	}
	return val
}

func (requestHandler *RequestHandler) serveResourceRequest(r *http.Request, deviceID, resourceHref, twin, resourceInterface string) (*httptest.ResponseRecorder, error) {
	resourceID := pb.ResourceIdFilter{
		ResourceId: commands.NewResourceID(deviceID, resourceHref),
		Etag:       getETags(r),
	}

	if (twin == "" || parseBoolQuery(twin)) && resourceInterface == "" {
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

func jsonGetValueOnPath(v interface{}, path ...string) (interface{}, error) {
	for idx, p := range path {
		if v == nil {
			return nil, fmt.Errorf("doesn't contains %v", strings.Join(path[:idx+1], "."))
		}
		m, ok := v.(map[interface{}]interface{})
		if !ok {
			return nil, fmt.Errorf("%v is not a map but %T", strings.Join(path[:idx+1], "."), v)
		}
		v, ok = m[p]
		if !ok {
			return nil, fmt.Errorf("doesn't contains %v", strings.Join(path[:idx+1], "."))
		}
	}
	return v, nil
}

func isContentEmpty(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	var ct commands.Content
	err := protojson.Unmarshal(data, &ct)
	if err != nil {
		return false
	}
	return len(ct.GetData()) == 0 && ct.GetCoapContentFormat() == -1
}

func (requestHandler *RequestHandler) filterOnlyContent(rec *httptest.ResponseRecorder, contentPath ...string) (resetContent bool) {
	if rec.Code == http.StatusNotModified {
		rec.Body.Reset()
		return false
	}
	if rec.Code != http.StatusOK {
		return false
	}
	var v map[interface{}]interface{}
	err := json.Decode(rec.Body.Bytes(), &v)
	if err != nil {
		requestHandler.logger.Debugf("filter only content: cannot decode response : %v", err)
		return false
	}
	content, err := jsonGetValueOnPath(v, contentPath...)
	if err != nil {
		requestHandler.logger.With("body", v).Debugf("filter only content: %v", err)
		return false
	}
	body, err := json.Encode(content)
	if err != nil {
		requestHandler.logger.With("body", v).Debugf("filter only content: cannot encode 'content' object: %v", err)
		return false
	}
	rec.Body.Reset()
	if !isContentEmpty(body) {
		_, _ = rec.Body.Write(body)
		return false
	}
	return true
}

func (requestHandler *RequestHandler) getResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars[uri.DeviceIDKey]
	resourceHref := vars[uri.ResourceHrefKey]
	twin := r.URL.Query().Get(uri.TwinQueryKey)
	onlyContent := r.URL.Query().Get(uri.OnlyContentQueryKey)
	resourceInterface := r.URL.Query().Get(uri.ResourceInterfaceQueryKey)
	rec, err := requestHandler.serveResourceRequest(r, deviceID, resourceHref, twin, resourceInterface)
	if err != nil {
		serverMux.WriteError(w, err)
		return
	}
	allowEmptyContent := false
	if parseBoolQuery(onlyContent) {
		allowEmptyContent = requestHandler.filterOnlyContent(rec, "result", "data", "content")
	}
	toSimpleResponse(w, rec, allowEmptyContent, func(w http.ResponseWriter, err error) {
		serverMux.WriteError(w, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot get resource('%v/%v') from the device: %v", deviceID, resourceHref, err))
	}, streamResponseKey)
}

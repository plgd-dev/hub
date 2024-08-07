package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/google/go-querystring/query"
	"github.com/gorilla/mux"
	jsoniter "github.com/json-iterator/go"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
)

func isNotModifiedResponse(result interface{}) bool {
	m, ok := result.(map[string]interface{})
	if !ok {
		return false
	}
	data, ok := m["data"]
	if ok {
		var tmp map[string]interface{}
		tmp, ok = data.(map[string]interface{})
		if !ok {
			return false
		}
		_, ok = tmp["status"]
		if ok {
			m = tmp
		}
	}
	v, ok := m["status"]
	if !ok {
		return false
	}
	statusVal, ok := v.(string)
	if !ok {
		return false
	}
	return statusVal == commands.Status_NOT_MODIFIED.String()
}

func writeSimpleResponse(w http.ResponseWriter, rec *httptest.ResponseRecorder, bodyForEncode interface{}, writeError func(w http.ResponseWriter, err error)) {
	if isNotModifiedResponse(bodyForEncode) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	encoder := jsoniter.NewEncoder(w)
	// copy everything from response recorder to actual response writer
	for k, v := range rec.Header() {
		if k == pkgHttp.ContentLengthHeaderKey {
			// jsoniter.Encode writes extra '\n' at the end of the response
			// which breaks a Content-Length check when the response is written
			continue
		}
		w.Header()[k] = v
	}
	w.WriteHeader(rec.Code)
	err := encoder.Encode(bodyForEncode)
	if err != nil {
		writeError(w, pkgGrpc.ForwardErrorf(codes.Internal, "cannot encode response: %v", err))
	}
}

func getResponse(rec *httptest.ResponseRecorder, allowEmpty bool, responseKeys ...string) (interface{}, error) {
	if len(rec.Body.Bytes()) == 0 && allowEmpty {
		return nil, nil
	}
	iter := json.NewDecoder(bytes.NewReader(rec.Body.Bytes()))
	datas := make([]interface{}, 0, 1)
	for {
		var v interface{}
		err := iter.Decode(&v)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		datas = append(datas, v)
	}
	if len(datas) == 0 {
		return nil, pkgGrpc.ForwardErrorf(codes.NotFound, "not found")
	}
	if len(datas) != 1 {
		return nil, pkgGrpc.ForwardErrorf(codes.InvalidArgument, "invalid number of responses")
	}
	result := datas[0]
	for _, key := range responseKeys {
		m, ok := result.(map[string]interface{})
		if !ok {
			break
		}
		tmp, ok := m[key]
		if ok {
			result = tmp
		} else {
			break
		}
	}
	return result, nil
}

func toSimpleResponse(w http.ResponseWriter, rec *httptest.ResponseRecorder, allowEmpty bool, writeError func(w http.ResponseWriter, err error), responseKeys ...string) {
	result, err := getResponse(rec, allowEmpty, responseKeys...)
	if err != nil {
		writeError(w, err)
		return
	}
	writeSimpleResponse(w, rec, result, writeError)
}

func (requestHandler *RequestHandler) serveDevicesRequest(r *http.Request) (string, *httptest.ResponseRecorder, error) {
	vars := mux.Vars(r)
	deviceID := vars[uri.DeviceIDKey]
	type Options struct {
		DeviceIDFilter []string `url:"deviceIdFilter"`
	}
	opt := Options{
		DeviceIDFilter: []string{deviceID},
	}
	v, err := query.Values(opt)
	if err != nil {
		return deviceID, nil, err
	}
	r.URL.Path = uri.Devices
	r.URL.RawQuery = v.Encode()
	rec := httptest.NewRecorder()
	requestHandler.mux.ServeHTTP(rec, r)
	return deviceID, rec, nil
}

func (requestHandler *RequestHandler) getDevice(w http.ResponseWriter, r *http.Request) {
	deviceID, rec, err := requestHandler.serveDevicesRequest(r)
	if err != nil {
		serverMux.WriteError(w, pkgGrpc.ForwardErrorf(codes.InvalidArgument, "cannot get device('%v'): %v", deviceID, err))
		return
	}
	toSimpleResponse(w, rec, false, func(w http.ResponseWriter, err error) {
		serverMux.WriteError(w, pkgGrpc.ForwardErrorf(codes.InvalidArgument, "cannot get device('%v'): %v", deviceID, err))
	}, streamResponseKey)
}

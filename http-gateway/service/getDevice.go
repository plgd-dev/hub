package service

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/google/go-querystring/query"
	"github.com/gorilla/mux"
	jsoniter "github.com/json-iterator/go"
	"github.com/plgd-dev/cloud/http-gateway/uri"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"google.golang.org/grpc/codes"
)

func toSimpleResponse(w http.ResponseWriter, rec *httptest.ResponseRecorder, writeError func(w http.ResponseWriter, err error), responseKeys ...string) {
	iter := json.NewDecoder(bytes.NewReader(rec.Body.Bytes()))
	datas := make([]interface{}, 0, 1)
	for {
		var v interface{}
		err := iter.Decode(&v)
		if err == io.EOF {
			break
		}
		if err != nil {
			writeError(w, err)
			return
		}
		datas = append(datas, v)
	}
	if len(datas) == 0 {
		writeError(w, kitNetGrpc.ForwardErrorf(codes.NotFound, "not found"))
		return
	}
	if len(datas) != 1 {
		writeError(w, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "invalid number of responses"))
		return
	}

	encoder := jsoniter.NewEncoder(w)
	var result interface{}
	result = datas[0]
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
	// copy everything from response recorder
	// to actual response writer
	for k, v := range rec.Header() {
		w.Header()[k] = v
	}
	w.WriteHeader(rec.Code)
	err := encoder.Encode(result)
	if err != nil {
		writeError(w, kitNetGrpc.ForwardErrorf(codes.Internal, "cannot encode response: %v", err))
	}
}

func (requestHandler *RequestHandler) getDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars[uri.DeviceIDKey]
	type Options struct {
		DeviceIdFilter []string `url:"deviceIdFilter"`
	}
	opt := Options{
		DeviceIdFilter: []string{deviceID},
	}
	v, err := query.Values(opt)
	if err != nil {
		writeError(w, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot get device('%v'): %v", deviceID, err))
		return
	}
	r.URL.Path = uri.Devices
	r.URL.RawQuery = v.Encode()
	rec := httptest.NewRecorder()
	requestHandler.mux.ServeHTTP(rec, r)

	toSimpleResponse(w, rec, func(w http.ResponseWriter, err error) {
		writeError(w, kitNetGrpc.ForwardErrorf(codes.InvalidArgument, "cannot get device('%v'): %v", deviceID, err))
	}, streamResponseKey)
}

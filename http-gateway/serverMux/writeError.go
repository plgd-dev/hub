package serverMux

import (
	"errors"
	"fmt"
	netHttp "net/http"

	"github.com/plgd-dev/go-coap/v3/message"
	coapStatus "github.com/plgd-dev/go-coap/v3/message/status"
	"github.com/plgd-dev/hub/v2/coap-gateway/coapconv"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/http"
	"google.golang.org/genproto/googleapis/rpc/status"
	grpcCodes "google.golang.org/grpc/codes"
	grpcStatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

type grpcErr interface {
	GRPCStatus() *grpcStatus.Status
}

type sdkErr interface {
	GetCode() grpcCodes.Code
}

// WriteError encodes error as google.golang.org/genproto/googleapis/rpc/status
func WriteError(w netHttp.ResponseWriter, err error) {
	if err == nil {
		w.WriteHeader(netHttp.StatusNoContent)
		return
	}
	log.Errorf("%v", err)

	w.Header().Set("Content-Type", message.AppJSON.String())
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.ErrToStatus(err))

	var gErr grpcErr
	var s *status.Status
	if errors.As(err, &gErr) {
		s = gErr.GRPCStatus().Proto()
	}
	var coapStatus coapStatus.Status
	if s == nil && errors.As(err, &coapStatus) {
		s = grpcStatus.New(coapconv.ToGrpcCode(coapStatus.Code(), grpcCodes.Internal), err.Error()).Proto()
	}

	var sErr sdkErr
	if s == nil && errors.As(err, &sErr) {
		s = grpcStatus.New(sErr.GetCode(), err.Error()).Proto()
	}
	if s == nil {
		s = grpcStatus.New(grpcCodes.Unknown, err.Error()).Proto()
	}

	v := protojson.MarshalOptions{
		EmitUnpopulated: true,
	}
	errStr, err2 := v.Marshal(s)
	if err2 != nil {
		log.Errorf("cannot marshal grpc error(%v): %v", err, err2)
		return
	}
	if _, err = fmt.Fprintln(w, string(errStr)); err != nil {
		log.Errorf("failed to write grpc error(%v): %v", errStr, err)
		return
	}
}

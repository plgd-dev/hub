package server

import (
	context "context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func tryToMoveData(req interface{}) interface{} {
	switch v := req.(type) {
	case interface {
		GetData() *events.ResourceUpdated
	}:
		return v.GetData()
	case interface {
		GetData() *events.ResourceRetrieved
	}:
		return v.GetData()
	case interface {
		GetData() *events.ResourceChanged
	}:
		return v.GetData()
	case interface {
		GetData() *events.ResourceDeleted
	}:
		return v.GetData()
	case interface {
		GetData() *events.ResourceCreated
	}:
		return v.GetData()
	case interface {
		GetData() *events.DeviceMetadataUpdated
	}:
		return v.GetData()
	}
	return req
}

func FillLoggerWithDeviceIDHrefCorrelationID(logger log.Logger, req interface{}) log.Logger {
	if req == nil {
		return logger
	}
	req = tryToMoveData(req)
	if d, ok := req.(interface{ GetDeviceId() string }); ok && d.GetDeviceId() != "" {
		logger = logger.With(log.DeviceIDKey, d.GetDeviceId())
	}
	if r, ok := req.(interface{ GetResourceId() *commands.ResourceId }); ok {
		logger = logger.With(log.DeviceIDKey, r.GetResourceId().GetDeviceId(), log.ResourceHrefKey, r.GetResourceId().GetHref())
	}
	if r, ok := req.(interface{ GetCorrelationId() string }); ok && r.GetCorrelationId() != "" {
		logger = logger.With(log.CorrelationIDKey, r.GetCorrelationId())
	}
	if r, ok := req.(interface{ GetAuditContext() *commands.AuditContext }); ok && r.GetAuditContext().GetCorrelationId() != "" {
		logger = logger.With(log.CorrelationIDKey, r.GetAuditContext().GetCorrelationId())
	}

	return logger
}

func DecodeToJsonObject(m interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	marshaler := serverMux.NewJsonMarshaler()
	marshaler.EmitUnpopulated = false
	data, err := marshaler.Marshal(m)
	if err == nil && len(data) > 2 {
		var v map[string]interface{}
		err = json.Unmarshal(data, &v)
		if err == nil && len(v) > 0 {
			return v
		}
	}
	return nil
}

type jwtMember struct {
	Sub string `json:"sub,omitempty"`
}

type logGrpcMessage struct {
	// request
	JWT     *jwtMember `json:"jwt,omitempty"`
	Method  string     `json:"method,omitempty"`
	Service string     `json:"service,omitempty"`

	// response
	Code  string `json:"code,omitempty"`
	Error string `json:"error,omitempty"`

	// request/response/stream
	Body map[string]interface{} `json:"body,omitempty"`

	// stream
	// for pairing streams
	StreamID string `json:"streamId,omitempty"`
}

func (m logGrpcMessage) IsEmpty() bool {
	return m.Body == nil && m.JWT == nil && m.Method == "" && m.Service == "" && m.Code == "" && m.Error == "" && m.StreamID == ""
}

func parseServiceAndMethod(fullMethod string) (string, string) {
	elems := strings.SplitAfterN(fullMethod, "/", 3)
	if len(elems) == 3 {
		service := strings.ReplaceAll(elems[1], "/", "")
		method := strings.ReplaceAll(elems[2], "/", "")
		return service, method
	}
	return "", fullMethod
}

func logUnary(ctx context.Context, logger log.Logger, req interface{}, resp interface{}, code *codes.Code, err error, fullMethod string, dumpBody bool, startTime time.Time, duration time.Duration, streamID string) log.Logger {
	if !startTime.IsZero() {
		logger = logger.With(log.StartTimeKey, startTime)
	}
	if duration > 0 {
		logger = logger.With(log.DurationMSKey, log.DurationToMilliseconds(duration))
	}
	deadline, ok := ctx.Deadline()
	if ok {
		logger = logger.With(log.DeadlineKey, deadline)
	}
	logger = FillLoggerWithDeviceIDHrefCorrelationID(logger, req)
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		logger = logger.With(log.TraceIDKey, spanCtx.TraceID().String())
	}

	reqData := logGrpcMessage{
		StreamID: streamID,
	}
	reqData.Service, reqData.Method = parseServiceAndMethod(fullMethod)
	sub := getSub(ctx)
	if sub != "" {
		reqData.JWT = &jwtMember{
			Sub: sub,
		}
	}
	if dumpBody {
		reqData.Body = DecodeToJsonObject(req)
	}
	if !reqData.IsEmpty() {
		logger = logger.With(log.RequestKey, reqData)
	}

	respData := logGrpcMessage{}
	if code != nil {
		respData.Code = code.String()
	}
	if err != nil {
		respData.Error = err.Error()
	}
	if dumpBody {
		respData.Body = DecodeToJsonObject(resp)
	}
	if !respData.IsEmpty() {
		logger = logger.With(log.ResponseKey, respData)
	}

	return logger
}

func handleUnary(ctx context.Context, logger log.Logger, dumpBody bool, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	startTime := time.Now()
	resp, err := handler(ctx, req)
	code := status.Code(err)
	level := DefaultCodeToLevel(code)
	if !logger.Check(level) {
		return resp, err
	}
	duration := time.Since(startTime)
	logger = logUnary(ctx, logger, req, resp, &code, err, info.FullMethod, dumpBody, startTime, duration, "")
	logger.GetLogFunc(level)("finished unary call with code " + code.String())
	return resp, err
}

func NewLogUnaryServerInterceptor(logger log.Logger, dumpBody bool) grpc.UnaryServerInterceptor {
	logger = logger.With(log.ProtocolKey, "GRPC")
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handleUnary(ctx, logger, dumpBody, req, info, handler)
	}
}

// serverStream wraps around the embedded grpc.ServerStream, and intercepts the RecvMsg and
// SendMsg method call.
type logServerStream struct {
	grpc.ServerStream
	logger   log.Logger
	dumpBody bool
	streamID string
	service  string
	method   string
	sub      string
}

func (w *logServerStream) SendMsg(m interface{}) error {
	err := w.ServerStream.SendMsg(m)
	code := status.Code(err)
	level := DefaultCodeToLevel(code)
	if errors.Is(err, io.EOF) {
		level = log.DebugLevel
	}
	if !w.logger.Check(level) {
		return err
	}
	logger := w.getLoggerForStreamMessage(m, err)
	logger.Debug("sent a streaming message")
	return err
}

func (w *logServerStream) getLoggerForStreamMessage(m interface{}, err error) log.Logger {
	r := logGrpcMessage{
		Service:  w.service,
		Method:   w.method,
		StreamID: w.streamID,
	}
	if err != nil {
		r.Error = err.Error()
	}
	if w.sub != "" {
		r.JWT = &jwtMember{
			Sub: w.sub,
		}
	}
	if w.dumpBody {
		r.Body = DecodeToJsonObject(m)
	}
	logger := w.logger
	logger = FillLoggerWithDeviceIDHrefCorrelationID(logger, m)
	if !r.IsEmpty() {
		logger = logger.With(log.MessageKey, r)
	}
	return logger
}

func (w *logServerStream) RecvMsg(m interface{}) error {
	err := w.ServerStream.RecvMsg(m)
	code := status.Code(err)
	level := DefaultCodeToLevel(code)
	if errors.Is(err, io.EOF) {
		return err
	}
	if !w.logger.Check(level) {
		return err
	}
	logger := w.getLoggerForStreamMessage(m, err)
	logger.Debug("received a streaming message")
	return err
}

func getSub(ctx context.Context) string {
	if sub, err := kitNetGrpc.OwnerFromTokenMD(ctx, "sub"); err == nil {
		return sub
	}
	return ""
}

func wrapServerStreamNew(logger log.Logger, dumpBody bool, fullMethod string, ss grpc.ServerStream) *logServerStream {
	service, method := parseServiceAndMethod(fullMethod)
	sub := getSub(ss.Context())
	return &logServerStream{
		ServerStream: ss,
		logger:       logger,
		dumpBody:     dumpBody,
		streamID:     uuid.New().String(),
		service:      service,
		method:       method,
		sub:          sub,
	}
}

func logStartStream(ctx context.Context, logger log.Logger, startTime time.Time, fullMethod, streamID string) {
	if logger.Check(log.DebugLevel) {
		logger = logUnary(ctx, logger, nil, nil, nil, nil, fullMethod, false, startTime, 0, streamID)
		logger.Debug("started streaming call")
	}
}

func handleStream(logger log.Logger, dumpBody bool, srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	startTime := time.Now()
	logServerStream := wrapServerStreamNew(logger, dumpBody, info.FullMethod, stream)
	logStartStream(logServerStream.Context(), logger, startTime, info.FullMethod, logServerStream.streamID)
	err := handler(srv, logServerStream)
	code := status.Code(err)
	level := DefaultCodeToLevel(code)
	if !logger.Check(level) {
		return err
	}
	duration := time.Since(startTime)
	logger = logUnary(stream.Context(), logger, nil, nil, &code, err, info.FullMethod, false, startTime, duration, logServerStream.streamID)
	logger.GetLogFunc(level)("finished streaming call with code " + code.String())
	return err
}

// StreamServerInterceptor returns a new streaming server interceptor that adds zap.Logger to the context.
func NewLogStreamServerInterceptor(logger log.Logger, dumpBody bool) grpc.StreamServerInterceptor {
	logger = logger.With(log.ProtocolKey, "GRPC")
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handleStream(logger, dumpBody, srv, stream, info, handler)
	}
}

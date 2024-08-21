package server

import (
	context "context"
	"math"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const (
	grpcPrefixKey = "grpc"
	requestKey    = "request"
)

var defaultCodeToLevel = map[codes.Code]zapcore.Level{
	codes.OK:                 log.DebugLevel,
	codes.Canceled:           log.DebugLevel,
	codes.Unknown:            log.ErrorLevel,
	codes.InvalidArgument:    log.DebugLevel,
	codes.DeadlineExceeded:   log.WarnLevel,
	codes.NotFound:           log.DebugLevel,
	codes.AlreadyExists:      log.DebugLevel,
	codes.PermissionDenied:   log.WarnLevel,
	codes.Unauthenticated:    log.DebugLevel,
	codes.ResourceExhausted:  log.WarnLevel,
	codes.FailedPrecondition: log.WarnLevel,
	codes.Aborted:            log.WarnLevel,
	codes.OutOfRange:         log.WarnLevel,
	codes.Unimplemented:      log.ErrorLevel,
	codes.Internal:           log.ErrorLevel,
	codes.Unavailable:        log.WarnLevel,
	codes.DataLoss:           log.ErrorLevel,
}

// DefaultCodeToLevel is the default implementation of gRPC return codes and interceptor log level for server side.
func DefaultCodeToLevel(code codes.Code) zapcore.Level {
	lvl, ok := defaultCodeToLevel[code]
	if ok {
		return lvl
	}
	return log.ErrorLevel
}

func setLogBasicLabels(m map[string]interface{}, req interface{}) {
	if d, ok := req.(interface{ GetDeviceId() string }); ok && d.GetDeviceId() != "" {
		log.SetLogValue(m, log.DeviceIDKey, d.GetDeviceId())
	}
	if r, ok := req.(interface{ GetResourceId() *commands.ResourceId }); ok {
		log.SetLogValue(m, log.DeviceIDKey, r.GetResourceId().GetDeviceId())
		log.SetLogValue(m, log.ResourceHrefKey, r.GetResourceId().GetHref())
	}
	if r, ok := req.(interface{ GetCorrelationId() string }); ok && r.GetCorrelationId() != "" {
		log.SetLogValue(m, log.CorrelationIDKey, r.GetCorrelationId())
	}
}

func setLogFilterLabels(m map[string]interface{}, req interface{}) {
	if req == nil {
		return
	}
	if r, ok := req.(interface {
		GetCommandFilter() []pb.GetPendingCommandsRequest_Command
	}); ok {
		commandFiler := make([]string, 0, len(r.GetCommandFilter()))
		for _, f := range r.GetCommandFilter() {
			commandFiler = append(commandFiler, f.String())
		}
		log.SetLogValue(m, log.CommandFilterKey, commandFiler)
	}
	if r, ok := req.(interface{ GetResourceIdFilter() []string }); ok {
		log.SetLogValue(m, log.ResourceIDFilterKey, r.GetResourceIdFilter())
	}
	if r, ok := req.(interface{ GetDeviceIdFilter() []string }); ok {
		log.SetLogValue(m, log.DeviceIDFilterKey, r.GetDeviceIdFilter())
	}
	if r, ok := req.(interface{ GetTypeFilter() []string }); ok {
		log.SetLogValue(m, log.TypeFilterKey, r.GetTypeFilter())
	}
}

func setLogSubscriptionLabels(m map[string]interface{}, sub *pb.SubscribeToEvents) {
	switch sub.GetAction().(type) {
	case *pb.SubscribeToEvents_CreateSubscription_:
		m[log.SubActionKey] = "createSubscription"
	case *pb.SubscribeToEvents_CancelSubscription_:
		m[log.SubActionKey] = "cancelSubscription"
	}
	setLogFilterLabels(m, sub.GetCreateSubscription())
	eventFilter := make([]string, 0, len(sub.GetCreateSubscription().GetEventFilter()))
	for _, e := range sub.GetCreateSubscription().GetEventFilter() {
		eventFilter = append(eventFilter, e.String())
	}
	log.SetLogValue(m, log.EventFilterKey, eventFilter)
	log.SetLogValue(m, log.CorrelationIDKey, sub.GetCorrelationId())
}

// CodeGenRequestFieldExtractor is a function that relies on code-generated functions that export log fields from requests.
// These are usually coming from a protoc-plugin that generates additional information based on custom field options.
func CodeGenRequestFieldExtractor(fullMethod string, req interface{}) map[string]interface{} {
	m := grpc_ctxtags.CodeGenRequestFieldExtractor(fullMethod, req)
	if m == nil {
		m = make(map[string]interface{})
	}
	setLogBasicLabels(m, req)
	setLogFilterLabels(m, req)
	if sub, ok := req.(*pb.SubscribeToEvents); ok {
		setLogSubscriptionLabels(m, sub)
	}
	method := strings.SplitAfterN(fullMethod, "/", 3)
	if len(method) == 3 {
		m["service"] = strings.ReplaceAll(method[1], "/", "")
		m[log.MethodKey] = strings.ReplaceAll(method[2], "/", "")
	}
	m[log.StartTimeKey] = time.Now()

	if len(m) > 0 {
		return m
	}
	return nil
}

func defaultMessageProducer(ctx context.Context, ctxLogger context.Context, msg string, level zapcore.Level, code codes.Code, err error, duration zapcore.Field) {
	req := make(map[string]interface{})
	resp := make(map[string]interface{})
	resp["code"] = code.String()
	if err != nil {
		resp[log.ErrorKey] = err.Error()
	}

	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		req[log.TraceIDKey] = spanCtx.TraceID().String()
	}

	if sub, err := pkgGrpc.OwnerFromTokenMD(ctx, "sub"); err == nil {
		req[log.JWTKey] = map[string]string{
			log.SubKey: sub,
		}
	}
	tags := grpc_ctxtags.Extract(ctx)
	newTags := grpc_ctxtags.NewTags()
	newTags.Set(log.DurationMSKey, math.Float32frombits(uint32(duration.Integer))) //nolint:gosec
	newTags.Set(log.ProtocolKey, "GRPC")
	for k, v := range tags.Values() {
		if strings.EqualFold(k, grpcPrefixKey+"."+requestKey+"."+log.StartTimeKey) {
			newTags.Set(log.StartTimeKey, v)
			continue
		}
		if strings.EqualFold(k, grpcPrefixKey+"."+requestKey+"."+log.DeviceIDKey) {
			newTags.Set(log.DeviceIDKey, v)
			continue
		}
		if strings.EqualFold(k, grpcPrefixKey+"."+requestKey+"."+log.CorrelationIDKey) {
			newTags.Set(log.CorrelationIDKey, v)
			continue
		}
		if strings.HasPrefix(k, grpcPrefixKey+"."+requestKey+".") {
			req[strings.TrimPrefix(k, grpcPrefixKey+"."+requestKey+".")] = v
		}
	}
	if len(req) > 0 {
		newTags.Set(log.RequestKey, req)
	}
	if len(resp) > 0 {
		newTags.Set(log.ResponseKey, resp)
	}
	if deadline, ok := ctx.Deadline(); ok {
		newTags.Set(log.DeadlineKey, deadline)
	}
	ctx = grpc_ctxtags.SetInContext(ctxLogger, newTags)

	ctxzap.Extract(ctx).Check(level, msg).Write()
}

func MakeDefaultMessageProducer(logger *zap.Logger) func(ctx context.Context, msg string, level zapcore.Level, code codes.Code, err error, duration zapcore.Field) {
	ctxLogger := ctxzap.ToContext(context.Background(), logger)
	return func(ctx context.Context, msg string, level zapcore.Level, code codes.Code, err error, duration zapcore.Field) {
		defaultMessageProducer(ctx, ctxLogger, msg, level, code, err, duration)
	}
}

type GetDeviceIDPb interface {
	GetDeviceId() string
}

func setGrpcRequest(ctx context.Context, v interface{}) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}
	marshaler := serverMux.NewJsonMarshaler()
	marshaler.JSONPb.MarshalOptions.EmitUnpopulated = false
	data, err := marshaler.Marshal(v)
	if err == nil && len(data) > 2 {
		span.SetAttributes(attribute.String("grpc.request", string(data)))
	}
}

// serverStream wraps around the embedded grpc.ServerStream, and intercepts the RecvMsg and
// SendMsg method call.
type serverStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *serverStream) Context() context.Context {
	return w.ctx
}

func (w *serverStream) RecvMsg(m interface{}) error {
	err := w.ServerStream.RecvMsg(m)
	if err != nil {
		return err
	}
	setGrpcRequest(w.ctx, m)
	return err
}

func wrapServerStream(ctx context.Context, ss grpc.ServerStream) *serverStream {
	return &serverStream{
		ServerStream: ss,
		ctx:          ctx,
	}
}

func MakeDefaultOptions(auth pkgGrpc.AuthInterceptors, logger log.Logger, tracerProvider trace.TracerProvider) ([]grpc.ServerOption, error) {
	streamInterceptors := []grpc.StreamServerInterceptor{
		func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			if !info.IsClientStream {
				return handler(srv, wrapServerStream(ss.Context(), ss))
			}
			return handler(srv, ss)
		},
	}
	unaryInterceptors := []grpc.UnaryServerInterceptor{
		func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
			setGrpcRequest(ctx, req)
			return handler(ctx, req)
		},
	}

	cfg := logger.Config()
	if cfg.EncoderConfig.EncodeTime.TimeEncoder == nil {
		cfg.EncoderConfig.EncodeTime = log.MakeDefaultConfig().EncoderConfig.EncodeTime
	}
	streamInterceptors = append(streamInterceptors, NewLogStreamServerInterceptor(logger, cfg.DumpBody))
	unaryInterceptors = append(unaryInterceptors, NewLogUnaryServerInterceptor(logger, cfg.DumpBody))

	streamInterceptors = append(streamInterceptors, auth.Stream())
	unaryInterceptors = append(unaryInterceptors, auth.Unary())

	return []grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler(otelgrpc.WithTracerProvider(tracerProvider))),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			streamInterceptors...,
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			unaryInterceptors...,
		)),
	}, nil
}

type config struct {
	disableTokenForwarding bool
	whiteListedMethods     []string
}

type Option func(*config)

func WithDisabledTokenForwarding() Option {
	return func(c *config) {
		c.disableTokenForwarding = true
	}
}

func WithWhiteListedMethods(method ...string) Option {
	return func(c *config) {
		c.whiteListedMethods = append(c.whiteListedMethods, method...)
	}
}

func NewAuth(validator pkgGrpc.Validator, opts ...Option) pkgGrpc.AuthInterceptors {
	interceptor := pkgGrpc.ValidateJWTWithValidator(validator, func(context.Context, string) jwt.ClaimsValidator {
		return pkgJwt.NewScopeClaims()
	})
	var cfg config
	for _, o := range opts {
		o(&cfg)
	}
	return pkgGrpc.MakeAuthInterceptors(func(ctx context.Context, method string) (context.Context, error) {
		ctx, err := interceptor(ctx, method)
		if err != nil {
			log.Errorf("auth interceptor %v: %w", method, err)
			return ctx, err
		}

		if !cfg.disableTokenForwarding {
			if token, err := pkgGrpc.TokenFromMD(ctx); err == nil {
				ctx = pkgGrpc.CtxWithToken(ctx, token)
			}
		}

		return ctx, nil
	}, cfg.whiteListedMethods...)
}

package server

import (
	context "context"
	"math"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	pkgMath "github.com/plgd-dev/hub/v2/internal/math"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
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

type loggerCtxMarker struct{}

var loggerCtxMarkerKey = &loggerCtxMarker{}

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

func toZapFields(fields logging.Fields) []zap.Field {
	fs := make([]zap.Field, 0, len(fields)/2)
	fieldsIterator := fields.Iterator()
	for fieldsIterator.Next() {
		key, value := fieldsIterator.At()
		switch v := value.(type) {
		case string:
			fs = append(fs, zap.String(key, v))
		case int:
			fs = append(fs, zap.Int(key, v))
		case bool:
			fs = append(fs, zap.Bool(key, v))
		default:
			fs = append(fs, zap.Any(key, v))
		}
	}
	return fs
}

func defaultMessageFields(ctx context.Context, req, resp map[string]interface{}, duration zapcore.Field) logging.Fields {
	fields := logging.ExtractFields(ctx)
	var newFields logging.Fields
	newFields = append(newFields, log.DurationMSKey, math.Float32frombits(pkgMath.CastTo[uint32](duration.Integer)))
	newFields = append(newFields, log.ProtocolKey, "GRPC")
	fieldsIterator := fields.Iterator()
	for fieldsIterator.Next() {
		k, v := fieldsIterator.At()
		if strings.EqualFold(k, grpcPrefixKey+"."+requestKey+"."+log.StartTimeKey) {
			newFields = append(newFields, log.StartTimeKey, v)
			continue
		}
		if strings.EqualFold(k, grpcPrefixKey+"."+requestKey+"."+log.DeviceIDKey) {
			newFields = append(newFields, log.DeviceIDKey, v)
			continue
		}
		if strings.EqualFold(k, grpcPrefixKey+"."+requestKey+"."+log.CorrelationIDKey) {
			newFields = append(newFields, log.CorrelationIDKey, v)
			continue
		}
		if strings.HasPrefix(k, grpcPrefixKey+"."+requestKey+".") {
			req[strings.TrimPrefix(k, grpcPrefixKey+"."+requestKey+".")] = v
		}
	}
	if len(req) > 0 {
		newFields = append(newFields, log.RequestKey, req)
	}
	if len(resp) > 0 {
		newFields = append(newFields, log.ResponseKey, resp)
	}
	if deadline, ok := ctx.Deadline(); ok {
		newFields = append(newFields, log.DeadlineKey, deadline)
	}
	return newFields
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

	logging.InjectFields(ctxLogger, defaultMessageFields(ctx, req, resp, duration))

	logger, ok := ctx.Value(loggerCtxMarkerKey).(*zap.Logger)
	if !ok || logger == nil {
		return
	}
	loggerWithFields := logger.With(toZapFields(logging.ExtractFields(ctx))...)
	loggerWithFields.Check(level, msg).Write()
}

func MakeDefaultMessageProducer(logger *zap.Logger) func(ctx context.Context, msg string, level zapcore.Level, code codes.Code, err error, duration zapcore.Field) {
	ctxLogger := context.WithValue(context.Background(), loggerCtxMarkerKey, logger)
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
		grpc.ChainStreamInterceptor(streamInterceptors...),
		grpc.ChainUnaryInterceptor(unaryInterceptors...),
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

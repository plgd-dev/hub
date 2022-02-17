package server

import (
	context "context"
	"strings"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var defaultCodeToLevel = map[codes.Code]zapcore.Level{
	codes.OK:                 zap.DebugLevel,
	codes.Canceled:           zap.DebugLevel,
	codes.Unknown:            zap.ErrorLevel,
	codes.InvalidArgument:    zap.DebugLevel,
	codes.DeadlineExceeded:   zap.WarnLevel,
	codes.NotFound:           zap.DebugLevel,
	codes.AlreadyExists:      zap.DebugLevel,
	codes.PermissionDenied:   zap.WarnLevel,
	codes.Unauthenticated:    zap.DebugLevel,
	codes.ResourceExhausted:  zap.WarnLevel,
	codes.FailedPrecondition: zap.WarnLevel,
	codes.Aborted:            zap.WarnLevel,
	codes.OutOfRange:         zap.WarnLevel,
	codes.Unimplemented:      zap.ErrorLevel,
	codes.Internal:           zap.ErrorLevel,
	codes.Unavailable:        zap.WarnLevel,
	codes.DataLoss:           zap.ErrorLevel,
}

// DefaultCodeToLevel is the default implementation of gRPC return codes and interceptor log level for server side.
func DefaultCodeToLevel(code codes.Code) zapcore.Level {
	lvl, ok := defaultCodeToLevel[code]
	if ok {
		return lvl
	}
	return zap.ErrorLevel
}

func setLogBasicLables(m map[string]interface{}, req interface{}) {
	if d, ok := req.(interface{ GetDeviceId() string }); ok {
		m[log.DeviceIDKey] = d.GetDeviceId()
	}
	if r, ok := req.(interface{ GetResourceId() *commands.ResourceId }); ok {
		m[log.DeviceIDKey] = r.GetResourceId().GetDeviceId()
		m[log.ResourceHrefKey] = r.GetResourceId().GetHref()
	}
	if r, ok := req.(interface{ GetCorrelationId() string }); ok {
		m[log.CorrelationIDKey] = r.GetCorrelationId()
	}
}

func setLogFiltersLables(m map[string]interface{}, req interface{}) {
	if req == nil {
		return
	}
	if r, ok := req.(interface {
		GetCommandFilter() []pb.GetPendingCommandsRequest_Command
	}); ok && len(r.GetCommandFilter()) > 0 {
		commandFiler := make([]string, 0, len(r.GetCommandFilter()))
		for _, f := range r.GetCommandFilter() {
			commandFiler = append(commandFiler, f.String())
		}
		m[log.CommandFilterKey] = commandFiler
	}
	if r, ok := req.(interface{ GetResourceIdFilter() []string }); ok && len(r.GetResourceIdFilter()) > 0 {
		m[log.ResourceIDFilterKey] = r.GetResourceIdFilter()
	}
	if r, ok := req.(interface{ GetDeviceIdFilter() []string }); ok && len(r.GetDeviceIdFilter()) > 0 {
		m[log.DeviceIDFilterKey] = r.GetDeviceIdFilter()
	}
	if r, ok := req.(interface{ GetTypeFilter() []string }); ok && len(r.GetTypeFilter()) > 0 {
		m[log.TypeFilterKey] = r.GetTypeFilter()
	}
}

func setLogSubscriptionLabels(m map[string]interface{}, sub *pb.SubscribeToEvents) {
	switch sub.GetAction().(type) {
	case *pb.SubscribeToEvents_CreateSubscription_:
		m[log.SubActionKey] = "createSubscription"
	case *pb.SubscribeToEvents_CancelSubscription_:
		m[log.SubActionKey] = "cancelSubscription"
	}
	setLogFiltersLables(m, sub.GetCreateSubscription())
	if len(sub.GetCreateSubscription().GetEventFilter()) > 0 {
		eventFilter := make([]string, 0, len(sub.GetCreateSubscription().GetEventFilter()))
		for _, e := range sub.GetCreateSubscription().GetEventFilter() {
			eventFilter = append(eventFilter, e.String())
		}
		m[log.EventFilterKey] = eventFilter
	}
	if sub.GetCorrelationId() != "" {
		m[log.CorrelationIDKey] = sub.GetCorrelationId()
	}
}

// CodeGenRequestFieldExtractor is a function that relies on code-generated functions that export log fields from requests.
// These are usually coming from a protoc-plugin that generates additional information based on custom field options.
func CodeGenRequestFieldExtractor(fullMethod string, req interface{}) map[string]interface{} {
	m := grpc_ctxtags.CodeGenRequestFieldExtractor(fullMethod, req)
	if m == nil {
		m = make(map[string]interface{})
	}
	setLogBasicLables(m, req)
	setLogFiltersLables(m, req)
	if sub, ok := req.(*pb.SubscribeToEvents); ok {
		setLogSubscriptionLabels(m, sub)
	}

	if len(m) > 0 {
		return m
	}
	return nil
}

// DefaultMessageProducer writes the default message
func DefaultMessageProducer(ctx context.Context, msg string, level zapcore.Level, code codes.Code, err error, duration zapcore.Field) {
	// re-extract logger from newCtx, as it may have extra fields that changed in the holder.
	fields := []zapcore.Field{
		zap.Error(err),
		zap.String("grpc.code", code.String()),
		duration,
	}
	if sub, err := kitNetGrpc.OwnerFromTokenMD(ctx, "sub"); err != nil {
		fields = append(fields, zap.String(log.JWTSubKey, sub))
	}
	tags := grpc_ctxtags.Extract(ctx)
	newTags := grpc_ctxtags.NewTags()
	for k, v := range tags.Values() {
		if !strings.HasPrefix(k, "grpc.request.plgd") {
			newTags.Set(k, v)
			continue
		}
		newTags.Set(strings.TrimPrefix(k, "grpc.request."), v)
	}
	ctx = grpc_ctxtags.SetInContext(ctx, newTags)
	ctxzap.Extract(ctx).Check(level, msg).Write(fields...)
}

type GetDeviceIDPb interface {
	GetDeviceId() string
}

func MakeDefaultOptions(auth kitNetGrpc.AuthInterceptors, logger log.Logger) ([]grpc.ServerOption, error) {
	streamInterceptors := []grpc.StreamServerInterceptor{}
	unaryInterceptors := []grpc.UnaryServerInterceptor{}
	zapLogger, ok := logger.Unwrap().(*zap.SugaredLogger)
	if ok {
		cfg := logger.Config()
		if cfg.EncoderConfig.EncodeTime.TimeEncoder == nil {
			cfg.EncoderConfig.EncodeTime = log.MakeDefaultConfig().EncoderConfig.EncodeTime
		}
		streamInterceptors = append(streamInterceptors, grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(CodeGenRequestFieldExtractor)),
			grpc_zap.StreamServerInterceptor(zapLogger.Desugar(), grpc_zap.WithTimestampFormat(cfg.EncoderConfig.EncodeTime.TimeEncoder.TimeString()), grpc_zap.WithLevels(DefaultCodeToLevel), grpc_zap.WithMessageProducer(DefaultMessageProducer)))
		unaryInterceptors = append(unaryInterceptors, grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(CodeGenRequestFieldExtractor)),
			grpc_zap.UnaryServerInterceptor(zapLogger.Desugar(), grpc_zap.WithTimestampFormat(cfg.EncoderConfig.EncodeTime.TimeEncoder.TimeString()), grpc_zap.WithLevels(DefaultCodeToLevel), grpc_zap.WithMessageProducer(DefaultMessageProducer)))
	}
	streamInterceptors = append(streamInterceptors, auth.Stream())
	unaryInterceptors = append(unaryInterceptors, auth.Unary())

	return []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			streamInterceptors...,
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			unaryInterceptors...,
		)),
	}, nil

}

type cfg struct {
	disableTokenForwarding bool
	whiteListedMethods     []string
}

type Option func(*cfg)

func WithDisabledTokenForwarding() Option {
	return func(c *cfg) {
		c.disableTokenForwarding = true
	}
}

func WithWhiteListedMethods(method ...string) Option {
	return func(c *cfg) {
		c.whiteListedMethods = append(c.whiteListedMethods, method...)
	}
}

func NewAuth(validator kitNetGrpc.Validator, opts ...Option) kitNetGrpc.AuthInterceptors {
	interceptor := kitNetGrpc.ValidateJWTWithValidator(validator, func(ctx context.Context, method string) kitNetGrpc.Claims {
		return jwt.NewScopeClaims()
	})
	var cfg cfg
	for _, o := range opts {
		o(&cfg)
	}
	return kitNetGrpc.MakeAuthInterceptors(func(ctx context.Context, method string) (context.Context, error) {
		ctx, err := interceptor(ctx, method)
		if err != nil {
			log.Errorf("auth interceptor %v: %w", method, err)
			return ctx, err
		}

		if !cfg.disableTokenForwarding {
			if token, err := kitNetGrpc.TokenFromMD(ctx); err == nil {
				ctx = kitNetGrpc.CtxWithToken(ctx, token)
			}
		}

		return ctx, nil
	}, cfg.whiteListedMethods...)
}

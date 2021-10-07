package server

import (
	context "context"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/plgd-dev/hub/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/pkg/security/jwt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
)

func MakeDefaultOptions(auth kitNetGrpc.AuthInterceptors, logger log.Logger) ([]grpc.ServerOption, error) {
	streamInterceptors := []grpc.StreamServerInterceptor{}
	unaryInterceptors := []grpc.UnaryServerInterceptor{}
	zapLogger, ok := logger.(*zap.SugaredLogger)
	if ok && zapLogger.Desugar().Core().Enabled(zapcore.DebugLevel) {
		streamInterceptors = append(streamInterceptors, grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.StreamServerInterceptor(zapLogger.Desugar()))
		unaryInterceptors = append(unaryInterceptors, grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.UnaryServerInterceptor(zapLogger.Desugar()))
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

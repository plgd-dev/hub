package log

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pion/logging"
	pkgX509 "github.com/plgd-dev/hub/v2/pkg/security/x509"
	"github.com/ugorji/go/codec"
	"go.opentelemetry.io/otel"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var log atomic.Pointer[Logger]

const (
	// DebugLevel logs are typically voluminous, and are usually disabled in
	// production.
	DebugLevel = zap.DebugLevel
	// InfoLevel is the default logging priority.
	InfoLevel = zap.InfoLevel
	// WarnLevel logs are more important than Info, but don't need individual
	// human review.
	WarnLevel = zap.WarnLevel
	// ErrorLevel logs are high-priority. If an application is running smoothly,
	// it shouldn't generate any error-level logs.
	ErrorLevel = zap.ErrorLevel
	// DPanicLevel logs are particularly important errors. In development the
	// logger panics after writing the message.
	DPanicLevel = zap.DPanicLevel
	// PanicLevel logs a message, then panics.
	PanicLevel = zap.PanicLevel
	// FatalLevel logs a message, then calls os.Exit(1).
	FatalLevel = zap.FatalLevel
)

func MakeDefaultConfig() Config {
	return Config{
		Level:    InfoLevel,
		Encoding: "json",
		Stacktrace: StacktraceConfig{
			Enabled: false,
			Level:   WarnLevel,
		},
		EncoderConfig: EncoderConfig{
			EncodeTime: TimeEncoderWrapper{
				TimeEncoder: RFC3339NanoTimeEncoder{},
			},
		},
	}
}

type RFC3339NanoTimeEncoder struct{}

func (e RFC3339NanoTimeEncoder) Encode(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	zapcore.RFC3339NanoTimeEncoder(t, enc)
}

func (e RFC3339NanoTimeEncoder) String() string {
	return "rfc3339nano"
}

func (e RFC3339NanoTimeEncoder) TimeString() string {
	return time.RFC3339Nano
}

type RFC3339TimeEncoder struct{}

func (e RFC3339TimeEncoder) String() string {
	return "rfc3339"
}

func (e RFC3339TimeEncoder) TimeString() string {
	return time.RFC3339
}

func (e RFC3339TimeEncoder) Encode(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	zapcore.RFC3339TimeEncoder(t, enc)
}

type ISO8601TimeEncoder struct{}

func (e ISO8601TimeEncoder) String() string {
	return "iso8601"
}

func (e ISO8601TimeEncoder) TimeString() string {
	return "2006-01-02T15:04:05.000Z0700"
}

func (e ISO8601TimeEncoder) Encode(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	zapcore.ISO8601TimeEncoder(t, enc)
}

type EpochMillisTimeEncoder struct{}

func (e EpochMillisTimeEncoder) Encode(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	zapcore.EpochMillisTimeEncoder(t, enc)
}

func (e EpochMillisTimeEncoder) String() string {
	return "millis"
}

func (e EpochMillisTimeEncoder) TimeString() string {
	return time.StampMilli
}

type EpochNanosTimeEncoder struct{}

func (e EpochNanosTimeEncoder) Encode(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	zapcore.EpochNanosTimeEncoder(t, enc)
}

func (e EpochNanosTimeEncoder) String() string {
	return "nanos"
}

func (e EpochNanosTimeEncoder) TimeString() string {
	return time.StampNano
}

type EpochTimeEncoder struct{}

func (e EpochTimeEncoder) Encode(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	zapcore.EpochTimeEncoder(t, enc)
}

func (e EpochTimeEncoder) String() string {
	return ""
}

func (e EpochTimeEncoder) TimeString() string {
	return time.Stamp
}

type TimeEncoder interface {
	Encode(time.Time, zapcore.PrimitiveArrayEncoder)
	String() string
	TimeString() string
}

type TimeEncoderWrapper struct {
	TimeEncoder TimeEncoder
}

func (t *TimeEncoderWrapper) UnmarshalText(text []byte) error {
	switch string(text) {
	case "rfc3339nano", "RFC3339Nano":
		t.TimeEncoder = RFC3339NanoTimeEncoder{}
	case "rfc3339", "RFC3339":
		t.TimeEncoder = RFC3339TimeEncoder{}
	case "iso8601", "ISO8601":
		t.TimeEncoder = ISO8601TimeEncoder{}
	case "millis":
		t.TimeEncoder = EpochMillisTimeEncoder{}
	case "nanos":
		t.TimeEncoder = EpochNanosTimeEncoder{}
	default:
		t.TimeEncoder = EpochTimeEncoder{}
	}
	return nil
}

func (t TimeEncoderWrapper) MarshalText() ([]byte, error) {
	if t.TimeEncoder == nil {
		return []byte(EpochTimeEncoder{}.String()), nil
	}
	return []byte(t.TimeEncoder.String()), nil
}

type EncoderConfig struct {
	EncodeTime TimeEncoderWrapper `json:"timeEncoder" yaml:"timeEncoder"`
}

type StacktraceConfig struct {
	Enabled bool          `yaml:"enabled" json:"enabled" description:"enable stacktrace"`
	Level   zapcore.Level `json:"level" yaml:"level" description:"from level"`
}

func (c *StacktraceConfig) Validate() error {
	if c.Level < DebugLevel || c.Level > FatalLevel {
		return fmt.Errorf("level('%v')", c.Level)
	}
	return nil
}

// Config configuration for setup logging.
type Config struct {
	// Level is the minimum enabled logging level. Note that this is a dynamic
	// level, so calling Config.Level.SetLevel will atomically change the log
	// level of all loggers descended from this config.
	Level zapcore.Level `json:"level" yaml:"level"`
	// Encoding sets the logger's encoding. Valid values are "json" (default) and
	// "console", as well as any third-party encodings registered via
	// RegisterEncoder.
	Encoding   string           `json:"encoding" yaml:"encoding"`
	Stacktrace StacktraceConfig `json:"stacktrace" yaml:"stacktrace"`
	// An EncoderConfig allows users to configure the concrete encoders supplied by
	// zapcore.
	EncoderConfig EncoderConfig `json:"encoderConfig" yaml:"encoderConfig"`
	// zap.Config    `yaml:",inline"`

	// DumpBody if true, dump request/response/stream body to log.
	DumpBody bool `json:"dumpBody" yaml:"dumpBody"`
}

func (c *Config) Validate() error {
	if c.Level < DebugLevel || c.Level > FatalLevel {
		return fmt.Errorf("level('%v')", c.Level)
	}
	if err := c.Stacktrace.Validate(); err != nil {
		return fmt.Errorf("stacktrace.%w", err)
	}
	return nil
}

func init() {
	Setup(MakeDefaultConfig())
}

// Setup changes log configuration for the application.
// Call ASAP in main after parse args/env.
func Setup(config Config) {
	logger := NewLogger(config)
	Set(logger)
}

type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})
	Debugf(template string, args ...interface{})
	Infof(template string, args ...interface{})
	Warnf(template string, args ...interface{})
	Errorf(template string, args ...interface{})
	Fatalf(template string, args ...interface{})
	With(args ...interface{}) Logger
	Unwrap() interface{}
	LogAndReturnError(err error) error
	Config() Config
	Check(lvl zapcore.Level) bool
	GetLogFunc(lvl zapcore.Level) func(args ...interface{})
	DTLSLoggerFactory() logging.LoggerFactory
}

type otelErrorHandler struct {
	logger Logger
}

func (h *otelErrorHandler) Handle(err error) {
	if errors.Is(err, context.DeadlineExceeded) {
		return
	}
	_ = h.logger.LogAndReturnError(fmt.Errorf("opentelemetry collector client: %w", err))
}

// Set logger for global log fuctions
func Set(logger Logger) {
	log.Store(&logger)
	otel.SetErrorHandler(&otelErrorHandler{logger: logger})
}

type WrapSuggarLogger struct {
	logger *zap.SugaredLogger
	config Config
}

func (l *WrapSuggarLogger) Config() Config {
	return l.config
}

func (l *WrapSuggarLogger) Check(lvl zapcore.Level) bool {
	return l.logger.Desugar().Core().Enabled(lvl)
}

var getLogFuncMap = map[zapcore.Level]func(l *WrapSuggarLogger) func(args ...interface{}){
	DebugLevel: func(l *WrapSuggarLogger) func(args ...interface{}) { return l.Debug },
	InfoLevel:  func(l *WrapSuggarLogger) func(args ...interface{}) { return l.Info },
	WarnLevel:  func(l *WrapSuggarLogger) func(args ...interface{}) { return l.Warn },
	ErrorLevel: func(l *WrapSuggarLogger) func(args ...interface{}) { return l.Error },
	FatalLevel: func(l *WrapSuggarLogger) func(args ...interface{}) { return l.Fatal },
}

var emptyLogFunc = func(...interface{}) {
	// do nothing
}

func (l *WrapSuggarLogger) GetLogFunc(lvl zapcore.Level) func(args ...interface{}) {
	if f, ok := getLogFuncMap[lvl]; ok {
		return f(l)
	}
	return emptyLogFunc
}

func (l *WrapSuggarLogger) with(args ...interface{}) *WrapSuggarLogger {
	return &WrapSuggarLogger{
		logger: l.logger.With(args...),
	}
}

func (l *WrapSuggarLogger) With(args ...interface{}) Logger {
	return l.with(args...)
}

func (l *WrapSuggarLogger) Unwrap() interface{} {
	return l.logger
}

type grpcErr interface {
	GRPCStatus() *status.Status
}

func (l *WrapSuggarLogger) PreProcessArgs(args ...interface{}) *WrapSuggarLogger {
	for _, arg := range args {
		if arg == nil {
			continue
		}
		err, ok := arg.(error)
		if !ok {
			continue
		}
		var x509Err *pkgX509.Error
		if errors.As(err, &x509Err) {
			if crt := x509Err.LeafCertificateInfo(); crt != nil {
				return l.with(X509Key, crt)
			}
		}
	}
	return l
}

func (l *WrapSuggarLogger) Debug(args ...interface{}) {
	l.PreProcessArgs(args...).logger.Debug(args...)
}

func (l *WrapSuggarLogger) Info(args ...interface{}) {
	l.PreProcessArgs(args...).logger.Info(args...)
}

func (l *WrapSuggarLogger) Warn(args ...interface{}) {
	l.PreProcessArgs(args...).logger.Warn(args...)
}

func (l *WrapSuggarLogger) error(args ...interface{}) {
	l.PreProcessArgs(args...).logger.Error(args...)
}

func (l *WrapSuggarLogger) Error(args ...interface{}) {
	var template string
	var err error
	for _, arg := range args {
		if err != nil {
			template = fmt.Sprintf("%s %v", template, arg)
			continue
		}
		if err1, ok := arg.(error); ok {
			err = err1
			if template == "" {
				template = "%w"
			} else {
				template += " %w"
			}
		} else {
			template = fmt.Sprintf("%s %v", template, arg)
			continue
		}
	}
	if err != nil {
		_ = l.LogAndReturnError(fmt.Errorf(template, err))
		return
	}
	l.error(args...)
}

func (l *WrapSuggarLogger) Fatal(args ...interface{}) {
	l.PreProcessArgs(args...).logger.Fatal(args...)
}

func (l *WrapSuggarLogger) Debugf(template string, args ...interface{}) {
	l.PreProcessArgs(args...).logger.Debugf(template, args...)
}

func (l *WrapSuggarLogger) Infof(template string, args ...interface{}) {
	l.PreProcessArgs(args...).logger.Infof(template, args...)
}

func (l *WrapSuggarLogger) Warnf(template string, args ...interface{}) {
	l.PreProcessArgs(args...).logger.Warnf(template, args...)
}

// Errorf uses fmt.Sprintf to log a templated message.
func (l *WrapSuggarLogger) Errorf(template string, args ...interface{}) {
	err := fmt.Errorf(template, args...)
	_ = l.LogAndReturnError(err)
}

func (l *WrapSuggarLogger) Fatalf(template string, args ...interface{}) {
	l.PreProcessArgs(args...).logger.Fatalf(template, args...)
}

func (l *WrapSuggarLogger) LogAndReturnError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, io.EOF) {
		l.Debugf("%v", err)
		return err
	}
	if errors.Is(err, io.ErrClosedPipe) {
		l.Debugf("%v", err)
		return err
	}
	if errors.Is(err, context.Canceled) {
		l.Debugf("%v", err)
		return err
	}
	if errors.Is(err, context.DeadlineExceeded) {
		l.Warnf("%v", err)
		return err
	}
	var x509Err *pkgX509.Error
	if errors.As(err, &x509Err) {
		l.Warnf("%v", err)
		return err
	}
	if strings.Contains(err.Error(), `write: broken pipe`) {
		l.Debugf("%v", err)
		return err
	}
	if strings.Contains(err.Error(), `use of closed network connection`) {
		l.Debugf("%v", err)
		return err
	}
	var gErr grpcErr
	if errors.As(err, &gErr) {
		if gErr.GRPCStatus().Code() == codes.Canceled {
			l.Debugf("%v", err)
			return err
		}
		if gErr.GRPCStatus().Code() == codes.DeadlineExceeded {
			l.Warnf("%v", err)
			return err
		}
	}
	l.error(err)
	return err
}

// NewLogger creates logger
func NewLogger(config Config) *WrapSuggarLogger {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	if config.EncoderConfig.EncodeTime.TimeEncoder == nil {
		config.EncoderConfig.EncodeTime = MakeDefaultConfig().EncoderConfig.EncodeTime
	}
	encoderConfig.EncodeTime = config.EncoderConfig.EncodeTime.TimeEncoder.Encode

	encoderConfig.NewReflectedEncoder = func(w io.Writer) zapcore.ReflectedEncoder {
		var h codec.JsonHandle
		h.Canonical = true
		return codec.NewEncoder(w, &h)
	}
	// First, define our level-handling logic.
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		if lvl < config.Level {
			return false
		}
		return lvl >= zapcore.ErrorLevel
	})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		if lvl < config.Level {
			return false
		}
		return lvl < zapcore.ErrorLevel
	})

	// High-priority output should also go to standard error, and low-priority
	// output should also go to standard out.
	consoleDebugging := zapcore.Lock(os.Stdout)
	consoleErrors := zapcore.Lock(os.Stderr)

	// Optimize the Kafka output for machine consumption and the console output
	// for human operators.
	var encoder zapcore.Encoder
	if config.Encoding == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Join the outputs, encoders, and level-handling functions into
	// zapcore.Cores, then tee the four cores together.
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, consoleErrors, highPriority),
		zapcore.NewCore(encoder, consoleDebugging, lowPriority),
	)
	opts := make([]zap.Option, 0, 16)
	if config.Stacktrace.Enabled {
		opts = append(opts, zap.AddStacktrace(zap.NewAtomicLevelAt(WarnLevel)))
	}

	// From a zapcore.Core, it's easy to construct a Logger.
	logger := zap.New(core, opts...)
	return &WrapSuggarLogger{logger: logger.Sugar(), config: config}
}

func Get() Logger {
	return *log.Load()
}

// Debug uses fmt.Sprint to construct and log a message.
func Debug(args ...interface{}) {
	Get().Debug(args...)
}

// Info uses fmt.Sprint to construct and log a message.
func Info(args ...interface{}) {
	Get().Info(args...)
}

// Warn uses fmt.Sprint to construct and log a message.
func Warn(args ...interface{}) {
	Get().Warn(args...)
}

// Error uses fmt.Sprint to construct and log a message.
func Error(args ...interface{}) {
	Get().Error(args...)
}

// Fatal uses fmt.Sprint to construct and log a message, then calls os.Exit.
func Fatal(args ...interface{}) {
	Get().Fatal(args...)
}

// Debugf uses fmt.Sprintf to log a templated message.
func Debugf(template string, args ...interface{}) {
	Get().Debugf(template, args...)
}

// Infof uses fmt.Sprintf to log a templated message.
func Infof(template string, args ...interface{}) {
	Get().Infof(template, args...)
}

// Warnf uses fmt.Sprintf to log a templated message.
func Warnf(template string, args ...interface{}) {
	Get().Warnf(template, args...)
}

// Errorf uses fmt.Sprintf to log a templated message.
func Errorf(template string, args ...interface{}) {
	Get().Errorf(template, args...)
}

// Fatalf uses fmt.Sprintf to log a templated message, then calls os.Exit.
func Fatalf(template string, args ...interface{}) {
	Get().Fatalf(template, args...)
}

package log

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log atomic.Value

type RFC3339NanoTimeEncoder struct {
}

func (e RFC3339NanoTimeEncoder) Encode(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	zapcore.RFC3339NanoTimeEncoder(t, enc)
}

func (e RFC3339NanoTimeEncoder) String() string {
	return "rfc3339nano"
}

type RFC3339TimeEncoder struct {
}

func (e RFC3339TimeEncoder) String() string {
	return "rfc3339"
}

func (e RFC3339TimeEncoder) Encode(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	zapcore.RFC3339TimeEncoder(t, enc)
}

type ISO8601TimeEncoder struct {
}

func (e ISO8601TimeEncoder) String() string {
	return "iso8601"
}

func (e ISO8601TimeEncoder) Encode(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	zapcore.ISO8601TimeEncoder(t, enc)
}

type EpochMillisTimeEncoder struct {
}

func (e EpochMillisTimeEncoder) Encode(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	zapcore.EpochMillisTimeEncoder(t, enc)
}

func (e EpochMillisTimeEncoder) String() string {
	return "millis"
}

type EpochNanosTimeEncoder struct {
}

func (e EpochNanosTimeEncoder) Encode(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	zapcore.EpochNanosTimeEncoder(t, enc)
}

func (e EpochNanosTimeEncoder) String() string {
	return "nanos"
}

type EpochTimeEncoder struct {
}

func (e EpochTimeEncoder) Encode(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	zapcore.EpochTimeEncoder(t, enc)
}

func (e EpochTimeEncoder) String() string {
	return ""
}

type TimeEncoder interface {
	Encode(time.Time, zapcore.PrimitiveArrayEncoder)
	String() string
}

type TimeEncoderWrapper struct {
	TimeEncoder TimeEncoder
}

func (e *TimeEncoderWrapper) UnmarshalText(text []byte) error {
	switch string(text) {
	case "rfc3339nano", "RFC3339Nano":
		e.TimeEncoder = RFC3339NanoTimeEncoder{}
	case "rfc3339", "RFC3339":
		e.TimeEncoder = RFC3339TimeEncoder{}
	case "iso8601", "ISO8601":
		e.TimeEncoder = ISO8601TimeEncoder{}
	case "millis":
		e.TimeEncoder = EpochMillisTimeEncoder{}
	case "nanos":
		e.TimeEncoder = EpochNanosTimeEncoder{}
	default:
		e.TimeEncoder = EpochTimeEncoder{}
	}
	return nil
}

func (t TimeEncoderWrapper) MarshalText() ([]byte, error) {
	return []byte(t.TimeEncoder.String()), nil
}

type EncoderConfig struct {
	EncodeTime TimeEncoderWrapper `json:"timeEncoder" yaml:"timeEncoder"`
}

// Config configuration for setup logging.
type Config struct {
	// Deprecated: replaced by level
	Debug bool `yaml:"debug" json:"debug" description:"enable debug logs"`
	// Level is the minimum enabled logging level. Note that this is a dynamic
	// level, so calling Config.Level.SetLevel will atomically change the log
	// level of all loggers descended from this config.
	Level zap.AtomicLevel `json:"level" yaml:"level"`
	// DisableStacktrace completely disables automatic stacktrace capturing. By
	// default, stacktraces are captured for WarnLevel and above logs in
	// development and ErrorLevel and above in production.
	DisableStacktrace bool `json:"disableStacktrace" yaml:"disableStacktrace"`
	// Encoding sets the logger's encoding. Valid values are "json" and
	// "console", as well as any third-party encodings registered via
	// RegisterEncoder.
	Encoding string `json:"encoding" yaml:"encoding"`

	// An EncoderConfig allows users to configure the concrete encoders supplied by
	// zapcore.
	EncoderConfig EncoderConfig `json:"encoderConfig" yaml:"encoderConfig"`
	//zap.Config    `yaml:",inline"`
}

func MakeDefaultConfig() Config {
	return Config{
		Debug:             false,
		Level:             zap.NewAtomicLevelAt(zap.InfoLevel),
		Encoding:          "json",
		DisableStacktrace: true,
		EncoderConfig: EncoderConfig{
			EncodeTime: TimeEncoderWrapper{
				TimeEncoder: RFC3339NanoTimeEncoder{},
			},
		},
	}
}

func init() {
	config := zap.NewProductionConfig()
	logger, err := config.Build()
	if err != nil {
		panic("Unable to create logger")
	}
	Set(logger.Sugar())
}

// Setup changes log configuration for the application.
// Call ASAP in main after parse args/env.
func Setup(config Config) {
	if err := Build(config); err != nil {
		panic(err)
	}
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
}

// Set logger for global log fuctions
func Set(logger Logger) {
	log.Store(logger)
}

// NewLogger creates logger
func NewLogger(config Config) (Logger, error) {
	var encoderConfig zapcore.EncoderConfig
	if config.Debug {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeTime = config.EncoderConfig.EncodeTime.TimeEncoder.Encode
	} else {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeTime = config.EncoderConfig.EncodeTime.TimeEncoder.Encode
	}
	// First, define our level-handling logic.
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		if lvl < config.Level.Level() {
			return false
		}
		return lvl >= zapcore.ErrorLevel
	})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		if lvl < config.Level.Level() {
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
	if config.Encoding == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// Join the outputs, encoders, and level-handling functions into
	// zapcore.Cores, then tee the four cores together.
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, consoleErrors, highPriority),
		zapcore.NewCore(encoder, consoleDebugging, lowPriority),
	)
	opts := make([]zap.Option, 0, 16)
	if !config.DisableStacktrace {
		opts = append(opts, zap.AddStacktrace(zap.NewAtomicLevelAt(zap.ErrorLevel)))
	}

	// From a zapcore.Core, it's easy to construct a Logger.
	logger := zap.New(core, opts...)
	return logger.Sugar(), nil
}

// Build is a panic-free version of Setup.
func Build(config Config) error {
	logger, err := NewLogger(config)
	if err != nil {
		return fmt.Errorf("logger creation failed: %w", err)
	}
	Set(logger)
	return nil
}

func Get() Logger {
	return log.Load().(Logger)
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
	err := fmt.Errorf(template, args...)
	_ = LogAndReturnError(err)
}

// Fatalf uses fmt.Sprintf to log a templated message, then calls os.Exit.
func Fatalf(template string, args ...interface{}) {
	Get().Fatalf(template, args...)
}

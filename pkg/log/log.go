package log

import (
	"fmt"
	"sync/atomic"

	"go.uber.org/zap"
)

var log atomic.Value

// Config configuration for setup logging.
type Config struct {
	Debug bool `yaml:"debug" json:"debug" description:"enable debug logs"`
}

func init() {
	config := zap.NewProductionConfig()
	logger, err := config.Build()
	if err != nil {
		panic("Unable to create logger")
	}
	log.Store(logger.Sugar())
}

// Setup changes log configuration for the application.
// Call ASAP in main after parse args/env.
func Setup(config Config) {
	if err := Build(config); err != nil {
		panic(err)
	}
}

// Set logger for global log fuctions
func Set(logger *zap.Logger) {
	log.Store(logger.Sugar())
}

// NewLogger creates logger
func NewLogger(config Config) (*zap.Logger, error) {
	var cfg zap.Config
	if config.Debug {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}
	return cfg.Build()
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

func Get() *zap.SugaredLogger {
	return log.Load().(*zap.SugaredLogger)
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

// DPanic uses fmt.Sprint to construct and log a message. In development, the
// logger then panics. (See DPanicLevel for details.)
func DPanic(args ...interface{}) {
	Get().DPanic(args...)
}

// Panic uses fmt.Sprint to construct and log a message, then panics.
func Panic(args ...interface{}) {
	Get().Panic(args...)
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

// DPanicf uses fmt.Sprintf to log a templated message. In development, the
// logger then panics. (See DPanicLevel for details.)
func DPanicf(template string, args ...interface{}) {
	Get().DPanicf(template, args...)
}

// Panicf uses fmt.Sprintf to log a templated message, then panics.
func Panicf(template string, args ...interface{}) {
	Get().Panicf(template, args...)
}

// Fatalf uses fmt.Sprintf to log a templated message, then calls os.Exit.
func Fatalf(template string, args ...interface{}) {
	Get().Fatalf(template, args...)
}

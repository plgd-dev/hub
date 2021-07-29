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
	var cfg zap.Config
	if config.Debug {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}
	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}
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
	LogAndReturnError(err)
}

// Fatalf uses fmt.Sprintf to log a templated message, then calls os.Exit.
func Fatalf(template string, args ...interface{}) {
	Get().Fatalf(template, args...)
}

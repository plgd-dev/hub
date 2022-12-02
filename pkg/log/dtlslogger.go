package log

import "github.com/pion/logging"

type DTLSLogger struct {
	logger Logger
}

func (l DTLSLogger) Trace(msg string) {
	l.logger.Debug(msg)
}

func (l DTLSLogger) Tracef(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

func (l DTLSLogger) Debug(msg string) {
	l.logger.Debug(msg)
}

func (l DTLSLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

func (l DTLSLogger) Info(msg string) {
	l.logger.Info(msg)
}

func (l DTLSLogger) Infof(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

func (l DTLSLogger) Warn(msg string) {
	l.logger.Warn(msg)
}

func (l DTLSLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

func (l DTLSLogger) Error(msg string) {
	l.logger.Error(msg)
}

func (l DTLSLogger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

type DTLSLoggerFactory struct {
	logger *WrapSuggarLogger
}

func (f DTLSLoggerFactory) NewLogger(scope string) logging.LeveledLogger {
	logger := f.logger.With("dtls", scope)
	return DTLSLogger{
		logger: logger,
	}
}

func (l *WrapSuggarLogger) DTLSLoggerFactory() logging.LoggerFactory {
	return DTLSLoggerFactory{logger: l}
}

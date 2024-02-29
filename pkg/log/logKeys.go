package log

import "time"

var (
	CorrelationIDKey    = "correlationId"
	DeviceIDKey         = "deviceId"
	ResourceHrefKey     = "href"
	CommandFilterKey    = "commandFilter"
	DeviceIDFilterKey   = "deviceIdFilter"
	ResourceIDFilterKey = "resourceIdFilter"
	TypeFilterKey       = "typeFilter"
	EventFilterKey      = "eventFilter"
	SubActionKey        = "subscriptionAction"
	PlgdKey             = "plgd"
	JWTKey              = "jwt"
	SubKey              = "sub"
	RequestKey          = "request"
	ResponseKey         = "response"
	StartTimeKey        = "startTime"
	DeadlineKey         = "deadline"
	MethodKey           = "method"
	ProtocolKey         = "protocol"
	BodyKey             = "body"
	DurationMSKey       = "durationMs"
	ErrorKey            = "error"
	X509Key             = "x509"
	TraceIDKey          = "traceId"
	MessageKey          = "message"
	SubjectsKey         = "subjects"
	CertManagerKey      = "certManager"
	LocalEndpointsKey   = "localEndpoints"
)

func DurationToMilliseconds(duration time.Duration) float32 {
	return float32(duration.Nanoseconds()/1000) / 1000
}

func SetLogValue(m map[string]interface{}, key string, val interface{}) {
	if val == nil {
		return
	}
	if v, ok := val.(string); ok && v == "" {
		return
	}
	if v, ok := val.([]string); ok && len(v) == 0 {
		return
	}
	if v, ok := val.(map[string]interface{}); ok && len(v) == 0 {
		return
	}
	if v, ok := val.(map[interface{}]interface{}); ok && len(v) == 0 {
		return
	}
	m[key] = val
}

package log

import "time"

var CorrelationIDKey = "correlationId"
var DeviceIDKey = "deviceId"
var ResourceHrefKey = "href"
var JWTSubKey = "jwt.sub"
var CommandFilterKey = "commandFilter"
var DeviceIDFilterKey = "deviceIdFilter"
var ResourceIDFilterKey = "resourceIdFilter"
var TypeFilterKey = "typeFilter"
var EventFilterKey = "eventFilter"
var SubActionKey = "subscriptionAction"
var PlgdKey = "plgd"
var JWTKey = "jwt"
var SubKey = "sub"
var RequestKey = "request"
var ResponseKey = "response"
var StartTimeKey = "startTime"
var DeadlineKey = "deadline"
var MethodKey = "method"
var ProtocolKey = "protocol"
var BodyKey = "body"
var DurationMSKey = "durationMs"
var ErrorKey = "error"

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

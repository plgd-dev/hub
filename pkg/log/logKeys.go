package log

import "time"

var CorrelationIDKey = "plgd.correlationId"
var DeviceIDKey = "plgd.deviceId"
var ResourceHrefKey = "plgd.resource.href"
var JWTSubKey = "jwt.sub"
var SubActionKey = "plgd.sub.action"
var SubEventsKey = "plgd.sub.events"
var SubDeviceIDsKey = "plgd.sub.deviceIds"
var SubResourceIDsKey = "plgd.sub.resourceIds"

func DurationToMilliseconds(duration time.Duration) float32 {
	return float32(duration.Nanoseconds()/1000) / 1000
}

func DurationKey(protocol string) string {
	return protocol + ".time_ms"
}

func StartTimeKey(protocol string) string {
	return protocol + ".start_time"
}

func ServiceKey(protocol string) string {
	return protocol + ".service"
}

func HrefKey(protocol string) string {
	return protocol + ".href"
}

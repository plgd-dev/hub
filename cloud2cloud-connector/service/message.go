package service

import "github.com/plgd-dev/go-coap/v3/message"

func stringToSupportedMediaType(t string) int32 {
	switch t {
	case message.AppCBOR.String():
		return int32(message.AppCBOR)
	case message.AppOcfCbor.String():
		return int32(message.AppOcfCbor)
	case message.AppJSON.String():
		return int32(message.AppJSON)
	}
	return int32(-1)
}

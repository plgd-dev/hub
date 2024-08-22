package coapconv

import (
	"fmt"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/plgd-dev/kit/v2/codec/json"
)

type EncodeFunc = func(v interface{}) ([]byte, error)

// GetAccept returns expected content format by client
func GetAccept(opts message.Options) message.MediaType {
	ct, err := opts.GetUint32(message.Accept)
	if err != nil {
		return message.AppOcfCbor
	}
	return message.MediaType(ct) //nolint:gosec
}

// GetEncoder returns encoder by accept
func GetEncoder(accept message.MediaType) (EncodeFunc, error) {
	switch accept {
	case message.AppJSON:
		return json.Encode, nil
	case message.AppCBOR, message.AppOcfCbor:
		return cbor.Encode, nil
	default:
		return nil, fmt.Errorf("unsupported type (%v)", accept)
	}
}

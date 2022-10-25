package client

import (
	"bytes"
	"context"
	"fmt"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Codec encodes/decodes according to the CoAP content format/media type.
type Codec = interface {
	ContentFormat() message.MediaType
	Encode(v interface{}) ([]byte, error)
	Decode(m *pool.Message, v interface{}) error
}

func ContentTypeToMediaType(contentType string) (message.MediaType, error) {
	switch contentType {
	case message.TextPlain.String():
		return message.TextPlain, nil
	case message.AppCBOR.String():
		return message.AppCBOR, nil
	case message.AppOcfCbor.String():
		return message.AppOcfCbor, nil
	case message.AppJSON.String():
		return message.AppJSON, nil
	default:
		return message.TextPlain, fmt.Errorf("unknown content type '%v'", contentType)
	}
}

func DecodeContentWithCodec(codec Codec, contentType string, data []byte, response interface{}) error {
	if response == nil {
		return nil
	}
	if val, ok := response.(*[]byte); ok && len(data) == 0 {
		*val = data
		return nil
	}
	if val, ok := response.(*interface{}); ok && len(data) == 0 {
		*val = nil
		return nil
	}
	mediaType, err := ContentTypeToMediaType(contentType)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "cannot convert response contentype %v to mediatype: %v", contentType, err)
	}
	opts := make(message.Options, 0, 1)
	opts, _, _ = opts.SetContentFormat(make([]byte, 4), mediaType)
	msg := pool.NewMessage(context.Background())
	msg.ResetOptionsTo(opts)
	msg.SetBody(bytes.NewReader(data))
	err = codec.Decode(msg, response)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "cannot decode response: %v", err)
	}

	return err
}

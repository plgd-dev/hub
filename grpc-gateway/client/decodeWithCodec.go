package client

import (
	"bytes"
	"fmt"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/net/coap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

func DecodeContentWithCodec(codec coap.Codec, contentType string, data []byte, response interface{}) error {
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
	msg := &message.Message{
		Options: opts,
		Body:    bytes.NewReader(data),
	}
	err = codec.Decode(msg, response)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "cannot decode response: %v", err)
	}

	return err
}

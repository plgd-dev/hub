package client

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-ocf/go-coap"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
)

func ContentTypeToMediaType(contentType string) (coap.MediaType, error) {
	switch contentType {
	case coap.TextPlain.String():
		return coap.TextPlain, nil
	case coap.AppCBOR.String():
		return coap.AppCBOR, nil
	case coap.AppOcfCbor.String():
		return coap.AppOcfCbor, nil
	case coap.AppJSON.String():
		return coap.AppJSON, nil
	default:
		return coap.TextPlain, fmt.Errorf("unknown content type '%v'", contentType)
	}
}

func DecodeContentWithCodec(codec kitNetCoap.Codec, contentType string, data []byte, response interface{}) error {
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
	msg := coap.NewTcpMessage(coap.MessageParams{
		Payload: data,
	})
	msg.SetOption(coap.ContentFormat, mediaType)
	err = codec.Decode(msg, response)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "cannot decode response: %v", err)
	}

	return err
}

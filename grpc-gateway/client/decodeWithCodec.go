package client

import (
	"fmt"

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

func DecodeContentWithCodec(codec kitNetmessage.Codec, contentType string, data []byte, response interface{}) error {
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
	msg := message.NewTcpMessage(message.MessageParams{
		Payload: data,
	})
	msg.SetOption(message.ContentFormat, mediaType)
	err = codec.Decode(msg, response)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "cannot decode response: %v", err)
	}

	return err
}

package commands

import (
	"fmt"
	"strings"

	extCodes "github.com/plgd-dev/cloud/grpc-gateway/pb/codes"
	"github.com/plgd-dev/cloud/grpc-gateway/pb/errdetails"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/codec/cbor"
	"github.com/plgd-dev/kit/codec/json"
	"google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type EventContent interface {
	GetContent() *Content
	GetStatus() Status
}

func (c *Content) ToJSON() (*Content, error) {
	switch {
	case c.GetCoapContentFormat() == -1 && c.GetContentType() == "" && len(c.GetData()) == 0:
		return &Content{
			CoapContentFormat: -1,
		}, nil
	case c.GetCoapContentFormat() == int32(message.AppJSON) || c.GetContentType() == message.AppJSON.String():
		return c, nil
	case c.GetCoapContentFormat() == int32(message.AppCBOR) || c.GetCoapContentFormat() == int32(message.AppOcfCbor) || c.GetContentType() == message.AppCBOR.String() || c.GetContentType() == message.AppOcfCbor.String():
		json, err := cbor.ToJSON(c.GetData())
		if err != nil {
			return nil, fmt.Errorf("cannot convert content(%+v) to json: %w", c, err)
		}
		return &Content{
			ContentType:       message.AppJSON.String(),
			CoapContentFormat: int32(message.AppJSON),
			Data:              []byte(json),
		}, nil
	}
	return nil, fmt.Errorf("conversion of content(%+v) to json is not supported", c)
}

func (c *Content) ToCBOR() (*Content, error) {
	switch {
	case c.GetCoapContentFormat() == -1 && c.GetContentType() == "" && len(c.GetData()) == 0:
		return &Content{
			CoapContentFormat: -1,
		}, nil
	case c.GetCoapContentFormat() == int32(message.AppCBOR) || c.GetCoapContentFormat() == int32(message.AppOcfCbor) || c.GetContentType() == message.AppCBOR.String() || c.GetContentType() == message.AppOcfCbor.String():
		return c, nil
	case c.GetCoapContentFormat() == int32(message.AppJSON) || c.GetContentType() == message.AppJSON.String():
		cbor, err := json.ToCBOR(string(c.GetData()))
		if err != nil {
			return nil, fmt.Errorf("cannot convert content(%+v) to cbor: %w", c, err)
		}
		return &Content{
			ContentType:       message.AppOcfCbor.String(),
			CoapContentFormat: int32(message.AppOcfCbor),
			Data:              []byte(cbor),
		}, nil
	}
	return nil, fmt.Errorf("conversion of content(%+v) to cbor is not supported", c)
}

func EventContentToContent(ec EventContent) (*Content, error) {
	var content *Content
	c := ec.GetContent()
	if c != nil {
		contentType := c.GetContentType()
		if contentType == "" && c.GetCoapContentFormat() >= 0 {
			contentType = message.MediaType(c.GetCoapContentFormat()).String()
		}
		content = &Content{
			Data:        c.GetData(),
			ContentType: contentType,
		}
	}
	statusCode := ec.GetStatus().ToGrpcCode()
	switch statusCode {
	case codes.OK:
	case extCodes.Accepted:
	case extCodes.Created:
	default:
		s := status.New(statusCode, "response from device")
		if content != nil {
			newS, err := s.WithDetails(&errdetails.DeviceError{
				Content: &errdetails.Content{
					Data:        content.GetData(),
					ContentType: content.GetContentType(),
				},
			})
			if err == nil {
				s = newS
			}
		}
		return nil, s.Err()
	}
	return content, nil
}

const applicationMimeType = "application"

func GetContentEncoder(accept string) (func(ec *Content) (*Content, error), error) {
	a := strings.Split(accept, ",")
	if len(a) == 0 || (len(a) == 1 && a[0] == "") {
		return func(c *Content) (*Content, error) {
			return c, nil
		}, nil
	}
	var encode func(ec *Content) (*Content, error)
	for _, v := range a {
		switch v {
		case message.AppJSON.String():
			encode = func(ec *Content) (*Content, error) {
				return ec.ToJSON()
			}
		case message.AppCBOR.String():
			return func(ec *Content) (*Content, error) {
				return ec.ToCBOR()
			}, nil
		case message.AppOcfCbor.String():
			return func(ec *Content) (*Content, error) {
				return ec.ToCBOR()
			}, nil
		case applicationMimeType + "/*":
			return func(ec *Content) (*Content, error) {
				return ec.ToCBOR()
			}, nil
		}
	}
	if encode != nil {
		return encode, nil
	}
	return nil, fmt.Errorf("invalid accept header(%v)", accept)
}

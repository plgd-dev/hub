package client

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/plgd-dev/kit/v2/codec/json"
)

// GeneralMessageCodec encodes in application/vnd.ocf+cbor and decodes json/coap/text.
type GeneralMessageCodec struct{}

// ContentFormat used for encoding.
func (GeneralMessageCodec) ContentFormat() message.MediaType { return message.AppOcfCbor }

// Encode encodes v and returns bytes.
func (GeneralMessageCodec) Encode(v interface{}) ([]byte, error) {
	return cbor.Encode(v)
}

// Decode the CBOR payload of a COAP message.
func (GeneralMessageCodec) Decode(m *pool.Message, v interface{}) error {
	if v == nil {
		return nil
	}
	mt, err := m.Options().ContentFormat()
	if err != nil {
		return fmt.Errorf("cannot get content format: %w", err)
	}
	var decoder func(w io.Reader, v interface{}) error
	switch mt {
	case message.AppCBOR, message.AppOcfCbor:
		decoder = cbor.ReadFrom
	case message.AppJSON:
		decoder = json.ReadFrom
	case message.TextPlain:
		decoder = func(w io.Reader, v interface{}) error {
			data, err := ioutil.ReadAll(w)
			if err != nil {
				return err
			}
			if s, ok := v.(*string); ok {
				*s = string(data)
			}
			if s, ok := v.(*[]byte); ok {
				*s = data
			}
			return fmt.Errorf("invalid type of v(%T)", v)
		}
	default:
		return fmt.Errorf("unsupported content format: %v", mt)
	}

	if m.Body() == nil {
		return fmt.Errorf("unexpected empty body")
	}

	if err := decoder(m.Body(), v); err != nil {
		p, _ := m.Options().Path()
		return fmt.Errorf("decoding failed for the message %v on %v", m.Token(), p)
	}
	return nil
}

package message

import (
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/plgd-dev/kit/v2/codec/json"
)

type JsonCoapMessage struct {
	Code          string      `json:"code"`
	Path          string      `json:"path,omitempty"`
	Token         string      `json:"token"`
	Queries       []string    `json:"queries,omitempty"`
	Observe       *uint32     `json:"observe,omitempty"`
	ContentFormat string      `json:"contentFormat,omitempty"`
	Body          interface{} `json:"body,omitempty"`
}

func decodeBody(mt message.MediaType, body []byte) interface{} {
	if body == nil {
		return nil
	}
	var res interface{}
	switch mt {
	case message.AppCBOR, message.AppOcfCbor:
		err := cbor.Decode(body, &res)
		if err != nil {
			return nil
		}
		return res
	case message.TextPlain:
		return string(body)
	case message.AppJSON:
		err := json.Decode(body, &res)
		if err != nil {
			return nil
		}
		return res
	case message.AppXML:
		return string(body)
	default:
		return string(body)
	}
}

func ToJson(m *pool.Message, withBody bool) JsonCoapMessage {
	path, err := m.Path()
	if err != nil {
		path = ""
	}
	queries, err := m.Queries()
	if err != nil {
		queries = nil
	}
	var obs *uint32
	o, err := m.Observe()
	if err == nil {
		obs = &o
	}
	var contentFormat string
	var body interface{}
	var data []byte
	if withBody {
		data = readBody(m.Body())
	}
	ct, err := m.ContentFormat()
	if err == nil {
		contentFormat = ct.String()
		body = decodeBody(ct, data)
	} else if len(data) > 0 {
		body = string(data)
	}

	msg := JsonCoapMessage{
		Code:          m.Code().String(),
		Path:          path,
		Token:         m.Token().String(),
		Queries:       queries,
		Observe:       obs,
		ContentFormat: contentFormat,
		Body:          body,
	}
	return msg
}

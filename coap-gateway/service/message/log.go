package message

import (
	"io"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/plgd-dev/kit/v2/codec/json"
)

type JsonCoapMessage struct {
	Code          string      `json:"code,omitempty"`
	Path          string      `json:"href,omitempty"`
	Token         string      `json:"token,omitempty"`
	Queries       []string    `json:"queries,omitempty"`
	Observe       *uint32     `json:"observe,omitempty"`
	ContentFormat string      `json:"contentFormat,omitempty"`
	Body          interface{} `json:"body,omitempty"`
}

func (c JsonCoapMessage) IsEmpty() bool {
	return c.Code == "" && c.Path == "" && c.Token == "" && len(c.Queries) == 0 && c.Observe == nil && c.ContentFormat == "" && c.Body == nil
}

func readBody(r io.ReadSeeker) []byte {
	if r == nil {
		return nil
	}
	v, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil
	}
	_, err = r.Seek(0, io.SeekStart)
	if err != nil {
		return nil
	}
	body, err := io.ReadAll(r)
	if err != nil {
		return nil
	}
	_, _ = r.Seek(v, io.SeekStart)
	if len(body) == 0 {
		return nil
	}
	return body
}

func decodeData(mt message.MediaType, data []byte) interface{} {
	if len(data) == 0 {
		return nil
	}
	var res interface{}
	switch mt {
	case message.AppCBOR, message.AppOcfCbor:
		err := cbor.Decode(data, &res)
		if err != nil {
			return nil
		}
		return res
	case message.TextPlain:
		return string(data)
	case message.AppJSON:
		err := json.Decode(data, &res)
		if err != nil {
			return nil
		}
		return res
	case message.AppXML:
		return string(data)
	default:
		return string(data)
	}
}

func ToJson(m *pool.Message, withBody, withToken bool) JsonCoapMessage {
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
	var body interface{}
	var data []byte
	if withBody {
		data = readBody(m.Body())
	}
	ct, err := m.ContentFormat()
	if err == nil {
		body = decodeData(ct, data)
	} else if len(data) > 0 {
		body = string(data)
	}
	var token string
	if withToken {
		token = m.Token().String()
	}

	msg := JsonCoapMessage{
		Code:    m.Code().String(),
		Path:    path,
		Token:   token,
		Queries: queries,
		Observe: obs,
		Body:    body,
	}
	return msg
}

package service

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	"github.com/plgd-dev/hub/pkg/log"
	"github.com/plgd-dev/kit/v2/codec/cbor"
)

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
	body, err := ioutil.ReadAll(r)
	if err != nil {
		return nil
	}
	_, _ = r.Seek(v, io.SeekStart)
	if len(body) == 0 {
		return nil
	}
	return body
}

func writeBody(mt message.MediaType, body []byte) string {
	if body == nil {
		return "body is EMPTY"
	}
	switch mt {
	case message.AppCBOR, message.AppOcfCbor:
		s, err := cbor.ToJSON(body)
		if err != nil {
			log.Errorf("cannot encode %v to JSON: %w", body, err)
		}
		return fmt.Sprintf("CBOR:\n%v", s)
	case message.TextPlain:
		return fmt.Sprintf("TXT:\n%v", string(body))
	case message.AppJSON:
		return fmt.Sprintf("JSON:\n%v", string(body))
	case message.AppXML:
		return fmt.Sprintf("XML:\n%v", string(body))
	default:
		return fmt.Sprintf("RAW(%v):\n%v", mt, body)
	}
}

func decodeMsgToDebug(client *Client, resp *pool.Message, tag string) {
	if !client.server.config.Log.DumpCoapMessages {
		return
	}
	buf := bytes.NewBuffer(make([]byte, 0, 2048))
	path, _ := resp.Options().Path()
	queries, _ := resp.Options().Queries()

	fmt.Fprintf(buf, "\n-------------------%v------------------\n", tag)
	fmt.Fprintf(buf, "DeviceId: %v\n", getDeviceID(client))
	fmt.Fprintf(buf, "Token: %v\n", resp.Token())
	fmt.Fprintf(buf, "Path: %v\n", path)
	fmt.Fprintf(buf, "Code: %v\n", resp.Code())
	fmt.Fprintf(buf, "Query: %v\n", queries)

	if observe, err := resp.Options().Observe(); err == nil {
		fmt.Fprintf(buf, "Observe: %v\n", observe)
	}
	body := readBody(resp.Body())
	if mt, err := resp.Options().ContentFormat(); err == nil {
		fmt.Fprintf(buf, "ContentFormat: %v\n", mt)
		fmt.Fprint(buf, writeBody(mt, body))
	} else {
		if len(body) == 0 {
			fmt.Fprintf(buf, "body is EMPTY")
		} else {
			// https://tools.ietf.org/html/rfc7252#section-5.5.2
			fmt.Fprintf(buf, "error Message:\n%v", string(body))
		}
	}
	log.Debug(buf.String())
}

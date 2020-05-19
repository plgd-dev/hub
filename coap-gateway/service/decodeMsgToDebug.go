package service

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/go-coap/v2/mux"
	"github.com/go-ocf/kit/codec/cbor"
	"github.com/go-ocf/kit/log"
)

func decodeMsgToDebug(client *Client, resp *mux.Message, tag string) {
	buf := bytes.NewBuffer(make([]byte, 0, 2048))
	path, _ := resp.Options.Path()
	queries, _ := resp.Options.Queries()

	fmt.Fprintf(buf, "\n-------------------%v------------------\n", tag)
	fmt.Fprintf(buf, "DeviceId: %v\n", getDeviceId(client))
	fmt.Fprintf(buf, "Token: %v\n", resp.Token)
	fmt.Fprintf(buf, "Path: %v\n", path)
	fmt.Fprintf(buf, "Code: %v\n", resp.Code)
	fmt.Fprintf(buf, "Type: %v\n", resp.Type)
	fmt.Fprintf(buf, "Query: %v\n", queries)

	if observe, err := resp.Options.Observe(); err == nil {
		fmt.Fprintf(buf, "Observe: %v\n", observe)
	}
	if mt, err := resp.Options.ContentFormat(); err == nil {
		fmt.Fprintf(buf, "ContentFormat: %v\n", mt)
		body, _ := ioutil.ReadAll(resp.Body)
		switch mediaType {
		case message.AppCBOR, message.AppOcfCbor:
			s, err := cbor.ToJSON(resp.Payload())
			if err != nil {
				log.Errorf("Cannot encode %v to JSON: %v", body, err)
			}
			fmt.Fprintf(buf, "CBOR:\n%v", s)
		case message.TextPlain:
			fmt.Fprintf(buf, "TXT:\n%v", string(body))
		case message.AppJSON:
			fmt.Fprintf(buf, "JSON:\n%v", string(body))
		case message.AppXML:
			fmt.Fprintf(buf, "XML:\n%v", string(body))
		default:
			fmt.Fprintf(buf, "RAW(%v):\n%v", mediaType, body)
		}
	} else {
		fmt.Fprintf(buf, "RAW:\n%v", body)
	}
	log.Debug(buf.String())
}

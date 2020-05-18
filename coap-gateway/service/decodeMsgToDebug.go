package service

import (
	"bytes"
	"fmt"

	gocoap "github.com/go-ocf/go-coap"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/kit/codec/cbor"
	"github.com/go-ocf/kit/log"
)

func decodeMsgToDebug(client *Client, resp gocoap.Message, tag string) {
	buf := bytes.NewBuffer(make([]byte, 0, 2048))
	fmt.Fprintf(buf, "\n-------------------%v------------------\n", tag)
	fmt.Fprintf(buf, "DeviceId: %v\n", getDeviceId(client))
	fmt.Fprintf(buf, "Token: %v\n", resp.Token())
	fmt.Fprintf(buf, "Path: %v\n", resp.PathString())
	fmt.Fprintf(buf, "Code: %v\n", resp.Code())
	fmt.Fprintf(buf, "Type: %v\n", resp.Type())
	fmt.Fprintf(buf, "Query: %v\n", resp.Options(gocoap.URIQuery))
	fmt.Fprintf(buf, "ContentFormat: %v\n", resp.Options(gocoap.ContentFormat))
	if resp.Code() == coapCodes.GET || resp.Code() == coapCodes.Content {
		fmt.Fprintf(buf, "Observe: %v\n", resp.Option(gocoap.Observe))
	}
	if mediaType, ok := resp.Option(gocoap.ContentFormat).(gocoap.MediaType); ok {
		switch mediaType {
		case gocoap.AppCBOR, gocoap.AppOcfCbor:
			s, err := cbor.ToJSON(resp.Payload())
			if err != nil {
				log.Errorf("Cannot encode %v to JSON: %v", resp.Payload(), err)
			}
			fmt.Fprintf(buf, "CBOR:\n%v", s)
		case gocoap.TextPlain:
			fmt.Fprintf(buf, "TXT:\n%v", string(resp.Payload()))
		case gocoap.AppJSON:
			fmt.Fprintf(buf, "JSON:\n%v", string(resp.Payload()))
		case gocoap.AppXML:
			fmt.Fprintf(buf, "XML:\n%v", string(resp.Payload()))
		default:
			fmt.Fprintf(buf, "RAW(%v):\n%v", mediaType, resp.Payload())
		}
	} else {
		fmt.Fprintf(buf, "RAW:\n%v", resp.Payload())
	}
	log.Debug(buf.String())
}

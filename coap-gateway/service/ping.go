package service

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/go-ocf/kit/codec/cbor"

	gocoap "github.com/go-ocf/go-coap"
	coapCodes "github.com/go-ocf/go-coap/codes"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/net/coap"
)

type oicwkping struct {
	IntervalArray []int64 `json:"inarray,omitempty"`
	Interval      int64   `json:"in,omitempty"`
}

func getPingConfiguration(s gocoap.ResponseWriter, req *gocoap.Request, client *Client) {
	t := time.Now()
	defer func() {
		log.Debugf("resourcePingGetConfiguration takes %v", time.Since(t))
	}()

	ping := oicwkping{
		IntervalArray: []int64{1},
	}

	accept := coap.GetAccept(req.Msg)
	encode, err := coap.GetEncoder(accept)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot send ping configuration: %v", err), s, client, coapCodes.InternalServerError)
		return
	}

	out, err := encode(ping)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot send ping configuration: %v", err), s, client, coapCodes.InternalServerError)
		return
	}

	//return not fount to disable ping from client
	sendResponse(s, client, coapCodes.Content, accept, out)
}

func ping(s gocoap.ResponseWriter, req *gocoap.Request, client *Client) {
	t := time.Now()
	defer func() {
		log.Debugf("resourcePing takes %v", time.Since(t))
	}()
	deviceId := client.loadAuthorizationContext().DeviceId
	if deviceId == "" {
		deviceId = "unknown"
	}

	var ping oicwkping
	err := cbor.Decode(req.Msg.Payload(), &ping)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId %v: cannot handle ping: %v", deviceId, err), s, client, coapCodes.BadRequest)
		return
	}
	if ping.Interval == 0 {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId %v: cannot handle ping: invalid interval value", deviceId), s, client, coapCodes.BadRequest)
		return
	}

	client.server.oicPingCache.Set(client.remoteAddrString(), client, time.Duration(float64(ping.Interval)*float64(time.Minute)*1.3))

	//return not fount to disable ping from client
	sendResponse(s, client, coapCodes.Valid, gocoap.TextPlain, nil)
}

func pingOnEvicted(key string, v interface{}) {
	if client, ok := v.(*Client); ok {
		if atomic.LoadInt32(&client.isClosed) == 0 {
			client.Close()
			deviceId := client.loadAuthorizationContext().DeviceId
			if deviceId == "" {
				deviceId = "unknown"
			}
			log.Errorf("DeviceId %v: ping timeout", deviceId)
		}
	}
}

func resourcePingHandler(s gocoap.ResponseWriter, req *gocoap.Request, client *Client) {
	switch req.Msg.Code() {
	case coapCodes.GET:
		getPingConfiguration(s, req, client)
	case coapCodes.POST:
		ping(s, req, client)
	default:
		logAndWriteErrorResponse(fmt.Errorf("Forbidden request from %v", req.Client.RemoteAddr()), s, client, coapCodes.Forbidden)
	}
}

package service

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/go-ocf/kit/codec/cbor"

	"github.com/go-ocf/go-coap/v2/message"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/mux"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/net/coap"
)

type oicwkping struct {
	IntervalArray []int64 `json:"inarray,omitempty"`
	Interval      int64   `json:"in,omitempty"`
}

func getPingConfiguration(s mux.ResponseWriter, req *mux.Message, client *Client) {
	t := time.Now()
	defer func() {
		log.Debugf("resourcePingGetConfiguration takes %v", time.Since(t))
	}()

	ping := oicwkping{
		IntervalArray: []int64{1},
	}

	accept, err := req.Options.Accept()
	if err != nil {
		accept = message.AppOcfCbor
	}
	encode, err := coap.GetEncoder(accept)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot send ping configuration: %v", err), client, coapCodes.InternalServerError, req.Token)
		return
	}

	out, err := encode(ping)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot send ping configuration: %v", err), client, coapCodes.InternalServerError, req.Token)
		return
	}

	//return not fount to disable ping from client
	sendResponse(client, coapCodes.Content, req.Token, accept, out)
}

func ping(s mux.ResponseWriter, req *mux.Message, client *Client) {
	t := time.Now()
	defer func() {
		log.Debugf("resourcePing takes %v", time.Since(t))
	}()
	deviceID := client.loadAuthorizationContext().DeviceId
	if deviceID == "" {
		deviceID = "unknown"
	}

	var ping oicwkping
	err := cbor.ReadFrom(req.Body, &ping)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId %v: cannot handle ping: %v", deviceID, err), client, coapCodes.BadRequest, req.Token)
		return
	}
	if ping.Interval == 0 {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId %v: cannot handle ping: invalid interval value", deviceID), client, coapCodes.BadRequest, req.Token)
		return
	}

	client.server.oicPingCache.Set(client.remoteAddrString(), client, time.Duration(float64(ping.Interval)*float64(time.Minute)*1.3))

	//return not fount to disable ping from client
	sendResponse(client, coapCodes.Valid, req.Token, message.TextPlain, nil)
}

func pingOnEvicted(key string, v interface{}) {
	if client, ok := v.(*Client); ok {
		if atomic.LoadInt32(&client.isClosed) == 0 {
			client.Close()
			deviceID := client.loadAuthorizationContext().DeviceId
			if deviceID == "" {
				deviceID = "unknown"
			}
			log.Errorf("DeviceId %v: ping timeout", deviceID)
		}
	}
}

func resourcePingHandler(s mux.ResponseWriter, req *mux.Message, client *Client) {
	switch req.Code {
	case coapCodes.GET:
		getPingConfiguration(s, req, client)
	case coapCodes.POST:
		ping(s, req, client)
	default:
		logAndWriteErrorResponse(fmt.Errorf("Forbidden request from %v", client.remoteAddrString()), client, coapCodes.Forbidden, req.Token)
	}
}

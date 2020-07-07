package service

import (
	"fmt"
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

	accept := coap.GetAccept(req.Options)
	encode, err := coap.GetEncoder(accept)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot send ping configuration: %v", err), coapCodes.InternalServerError, req.Token)
		return
	}

	out, err := encode(ping)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot send ping configuration: %v", err), coapCodes.InternalServerError, req.Token)
		return
	}

	//return not fount to disable ping from client
	client.sendResponse(coapCodes.Content, req.Token, accept, out)
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
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId %v: cannot handle ping: %v", deviceID, err), coapCodes.BadRequest, req.Token)
		return
	}
	if ping.Interval == 0 {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId %v: cannot handle ping: invalid interval value", deviceID), coapCodes.BadRequest, req.Token)
		return
	}

	client.server.oicPingCache.Set(client.remoteAddrString(), client, time.Duration(float64(ping.Interval)*float64(time.Minute)*1.3))

	//return not fount to disable ping from client
	client.sendResponse(coapCodes.Valid, req.Token, message.TextPlain, nil)
}

func pingOnEvicted(key string, v interface{}) {
	if client, ok := v.(*Client); ok {
		select {
		case <-client.coapConn.Context().Done():
		default:
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
		client.logAndWriteErrorResponse(fmt.Errorf("Forbidden request from %v", client.remoteAddrString()), coapCodes.Forbidden, req.Token)
	}
}

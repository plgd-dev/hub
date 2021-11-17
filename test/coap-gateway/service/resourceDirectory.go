package service

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/hub/coap-gateway/coapconv"
	"github.com/plgd-dev/kit/v2/codec/cbor"
)

type PublishRequest struct {
	DeviceID       string               `json:"di"`
	Links          schema.ResourceLinks `json:"links"`
	TimeToLive     int                  `json:"ttl"`
	SequenceNumber uint64               `json:"-"`
}

type UnpublishRequest struct {
	DeviceID    string
	InstanceIDs []int64
}

func makePublishRequest() PublishRequest {
	return PublishRequest{
		TimeToLive: -1,
	}
}

// fixHref always lead by "/"
func fixHref(href string) string {
	backslash := regexp.MustCompile(`\/+`)
	p := backslash.ReplaceAllString(href, "/")
	p = strings.TrimRight(p, "/")
	if len(p) > 0 && p[0] == '/' {
		return p
	}
	return "/" + p
}

func resourceDirectoryPublishHandler(req *mux.Message, client *Client) {
	p := makePublishRequest()
	err := cbor.ReadFrom(req.Body, &p)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot read publish request body received: %w", err), coapCodes.BadRequest, req.Token)
		return
	}

	for i, link := range p.Links {
		p.Links[i].DeviceID = p.DeviceID
		p.Links[i].Href = fixHref(link.Href)
	}
	p.SequenceNumber = req.SequenceNumber

	if err := client.handler.PublishResources(p); err != nil {
		client.logAndWriteErrorResponse(err, coapCodes.InternalServerError, req.Token)
		return
	}

	accept := coapconv.GetAccept(req.Options)
	encode, err := coapconv.GetEncoder(accept)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("unable to get encoder for accepted type %v: %w", accept, err), coapCodes.InternalServerError, req.Token)
		return
	}
	out, err := encode(p)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("unable to encode publish response: %w", err), coapCodes.InternalServerError, req.Token)
		return
	}

	client.sendResponse(coapCodes.Changed, req.Token, accept, out)
}

func parseUnpublishRequestFromQuery(queries []string) (UnpublishRequest, error) {
	req := UnpublishRequest{}
	for _, q := range queries {
		values, err := url.ParseQuery(q)
		if err != nil {
			return UnpublishRequest{}, fmt.Errorf("cannot parse unpublish query: %w", err)
		}
		if di := values.Get("di"); di != "" {
			req.DeviceID = di
		}

		if ins := values.Get("ins"); ins != "" {
			i, err := strconv.Atoi(ins)
			if err != nil {
				return UnpublishRequest{}, fmt.Errorf("cannot convert %v to number", ins)
			}
			req.InstanceIDs = append(req.InstanceIDs, int64(i))
		}
	}

	if req.DeviceID == "" {
		return UnpublishRequest{}, fmt.Errorf("deviceID not found")
	}

	return req, nil
}

func resourceDirectoryUnpublishHandler(req *mux.Message, client *Client) {
	queries, err := req.Options.Queries()
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot query string from unpublish request from device %v: %w", client.GetDeviceID(), err), coapCodes.BadRequest, req.Token)
		return
	}

	r, err := parseUnpublishRequestFromQuery(queries)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("unable to parse unpublish request query string from device %v: %w", client.GetDeviceID(), err), coapCodes.BadRequest, req.Token)
		return
	}

	err = client.handler.UnpublishResources(r)
	if err != nil {
		client.logAndWriteErrorResponse(err, coapCodes.InternalServerError, req.Token)
		return

	}

	client.sendResponse(coapCodes.Deleted, req.Token, message.TextPlain, nil)
}

func resourceDirectoryHandler(req *mux.Message, client *Client) {
	switch req.Code {
	case coapCodes.POST:
		resourceDirectoryPublishHandler(req, client)
	case coapCodes.DELETE:
		resourceDirectoryUnpublishHandler(req, client)
	default:
		client.logAndWriteErrorResponse(fmt.Errorf("forbidden request from %v", client.RemoteAddrString()), coapCodes.Forbidden, req.Token)
	}
}

package service

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/kit/codec/cbor"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/net/coap"
	"github.com/plgd-dev/sdk/schema"
)

type wkRd struct {
	DeviceID         string               `json:"di"`
	Links            schema.ResourceLinks `json:"links"`
	TimeToLive       int                  `json:"ttl"`
	TimeToLiveLegacy int                  `json:"lt"`
}

func fixTTL(w wkRd) wkRd {
	// set time to live properly
	if w.TimeToLive <= 0 {
		w.TimeToLive = w.TimeToLiveLegacy
	} else {
		w.TimeToLiveLegacy = w.TimeToLive
	}
	return w
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

func validatePublish(w wkRd) error {
	if w.DeviceID == "" {
		return errors.New("invalid DeviceId")
	}
	if len(w.Links) == 0 {
		return errors.New("empty links")
	}
	if w.TimeToLive <= 0 && w.TimeToLiveLegacy <= 0 {
		return errors.New("invalid TimeToLive")
	}

	return nil
}

func resourceDirectoryPublishHandler(req *mux.Message, client *Client) {
	authCtx, err := client.loadAuthorizationContext()
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot publish resource: %w", authCtx.GetDeviceID(), err), coapCodes.Unauthorized, req.Token)
		return
	}

	var w wkRd
	err = cbor.ReadFrom(req.Body, &w)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot publish resource: %w", authCtx.GetDeviceID(), err), coapCodes.BadRequest, req.Token)
		return
	}

	if err := validatePublish(w); err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot publish resource: %w", authCtx.GetDeviceID(), err), coapCodes.BadRequest, req.Token)
		return
	}

	w = fixTTL(w)
	for _, link := range w.Links {
		link.DeviceID = w.DeviceID
		link.Href = fixHref(link.Href)
		link.InstanceID = getInstanceID(link.Href)
	}

	err = client.publishResourceLinks(req.Context, w.Links, w.DeviceID, int32(w.TimeToLive), client.remoteAddrString(), req.SequenceNumber, authCtx.GetPbData())
	if err != nil {
		log.Errorf("Cannot device %v publish resources %v", w.DeviceID, authCtx.GetDeviceID(), err)

	}

	for _, link := range w.Links {
		observable := link.Policy != nil && link.Policy.BitMask.Has(schema.Observable)
		err := client.observeResource(req.Context, link.GetDeviceID(), link.Href, observable, true)
		if err != nil {
			log.Errorf("DeviceId: %v: cannot observe published resource /%v%v: %v", link.GetDeviceID(), link.GetDeviceID(), link.Href, err)
		}
	}

	accept := coap.GetAccept(req.Options)
	encode, err := coap.GetEncoder(accept)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot publish resource: %w", authCtx.GetDeviceID(), err), coapCodes.InternalServerError, req.Token)
		return
	}
	out, err := encode(w)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot publish resource: %w", authCtx.GetDeviceID(), err), coapCodes.InternalServerError, req.Token)
		return
	}

	client.sendResponse(coapCodes.Changed, req.Token, accept, out)
}

func parseUnpublishQueryString(queries []string) (deviceID string, instanceIDs []int64, err error) {
	for _, q := range queries {
		values, err := url.ParseQuery(q)
		if err != nil {
			return "", nil, fmt.Errorf("cannot parse unpublish query: %w", err)
		}
		if di := values.Get("di"); di != "" {
			deviceID = di
		}

		if ins := values.Get("ins"); ins != "" {
			i, err := strconv.Atoi(ins)
			if err != nil {
				return "", nil, fmt.Errorf("cannot convert %v to number", ins)
			}
			instanceIDs = append(instanceIDs, int64(i))
		}
	}

	if deviceID == "" {
		return "", nil, fmt.Errorf("deviceID not found")
	}

	return
}

func resourceDirectoryUnpublishHandler(req *mux.Message, client *Client) {
	queries, err := req.Options.Queries()
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot get queries: %w", err), coapCodes.BadRequest, req.Token)
		return
	}
	deviceID, inss, err := parseUnpublishQueryString(queries)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot parse queries: %w", err), coapCodes.BadRequest, req.Token)
		return
	}

	rscs := client.getObservedResources(deviceID, inss)
	if len(rscs) == 0 {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot found resources for the DELETE request parameters - with device ID and instance IDs %v, ", queries), coapCodes.BadRequest, req.Token)
		return
	}

	client.unpublishResources(req.Context, rscs)

	client.sendResponse(coapCodes.Deleted, req.Token, message.TextPlain, nil)
}

type resourceDirectorySelector struct {
	SelectionCriteria int `json:"sel"`
}

func resourceDirectoryGetSelector(req *mux.Message, client *Client) {
	var rds resourceDirectorySelector //we want to use sel:0 to prefer cloud RD

	accept := coap.GetAccept(req.Options)
	encode, err := coap.GetEncoder(accept)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot get selector: %w", err), coapCodes.InternalServerError, req.Token)
		return
	}
	out, err := encode(rds)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot get selector: %w", err), coapCodes.InternalServerError, req.Token)
		return
	}

	client.sendResponse(coapCodes.Content, req.Token, accept, out)
}

func resourceDirectoryHandler(req *mux.Message, client *Client) {
	switch req.Code {
	case coapCodes.POST:
		resourceDirectoryPublishHandler(req, client)
	case coapCodes.DELETE:
		resourceDirectoryUnpublishHandler(req, client)
	case coapCodes.GET:
		resourceDirectoryGetSelector(req, client)
	default:
		client.logAndWriteErrorResponse(fmt.Errorf("forbidden request from %v", client.remoteAddrString()), coapCodes.Forbidden, req.Token)
	}
}

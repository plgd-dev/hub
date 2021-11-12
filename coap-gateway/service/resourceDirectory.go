package service

import (
	"errors"
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
	"github.com/plgd-dev/hub/coap-gateway/resource"
	"github.com/plgd-dev/kit/v2/codec/cbor"
)

type wkRd struct {
	DeviceID         string               `json:"di"`
	Links            schema.ResourceLinks `json:"links"`
	TimeToLive       int                  `json:"ttl"`
	TimeToLiveLegacy int                  `json:"lt"`
}

func makeWkRd() wkRd {
	return wkRd{
		TimeToLive:       -1,
		TimeToLiveLegacy: -1,
	}
}

func fixTimeToLive(w wkRd) wkRd {
	// set time to live properly
	if w.TimeToLive < 0 {
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
	if w.TimeToLive < 0 && w.TimeToLiveLegacy < 0 {
		return errors.New("invalid TimeToLive")
	}

	return nil
}

func resourceDirectoryPublishHandler(req *mux.Message, client *Client) {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot load authorization context for device %v: %w", authCtx.GetDeviceID(), err), coapCodes.BadRequest, req.Token)
		return
	}

	w := makeWkRd()
	err = cbor.ReadFrom(req.Body, &w)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot read publish request body received from device %v: %w", authCtx.GetDeviceID(), err), coapCodes.BadRequest, req.Token)
		return
	}

	if err := validatePublish(w); err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("invalid publish request received from device %v: %w", authCtx.GetDeviceID(), err), coapCodes.BadRequest, req.Token)
		return
	}

	w = fixTimeToLive(w)
	for i, link := range w.Links {
		w.Links[i].DeviceID = w.DeviceID
		w.Links[i].Href = fixHref(link.Href)
		w.Links[i].InstanceID = resource.GetInstanceID(link.Href)
	}

	publishedResources, err := client.publishResourceLinks(req.Context, w.Links, w.DeviceID, int32(w.TimeToLive), client.remoteAddrString(), req.SequenceNumber)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("unable to publish resources for device %v: %w", w.DeviceID, err), coapCodes.BadRequest, req.Token)
		return
	}

	if err := client.server.taskQueue.Submit(func() {
		client.resourceObserver.AddResources(req.Context, publishedResources)
	}); err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("unable to observe published resources for device %v: %w", authCtx.GetDeviceID(), err), coapCodes.InternalServerError, req.Token)
		return
	}

	accept := coapconv.GetAccept(req.Options)
	encode, err := coapconv.GetEncoder(accept)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("unable to get encoder for accepted type %v requested by device %v: %w", accept, authCtx.GetDeviceID(), err), coapCodes.InternalServerError, req.Token)
		return
	}
	out, err := encode(w)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("unable to encode publish response for device %v: %w", authCtx.GetDeviceID(), err), coapCodes.InternalServerError, req.Token)
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
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot load authorization context for device %v: %w", authCtx.GetDeviceID(), err), coapCodes.BadRequest, req.Token)
		return
	}

	queries, err := req.Options.Queries()
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot query string from unpublish request from device %v: %w", authCtx.GetDeviceID(), err), coapCodes.BadRequest, req.Token)
		return
	}
	deviceID, inss, err := parseUnpublishQueryString(queries)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("unable to parse unpublish request query string from device %v: %w", authCtx.GetDeviceID(), err), coapCodes.BadRequest, req.Token)
		return
	}

	resources := client.resourceObserver.GetTrackedResources(deviceID, inss)
	if len(resources) == 0 {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot find observed resources using query %v which shall be unpublished from device %v", queries, authCtx.GetDeviceID()), coapCodes.BadRequest, req.Token)
		return
	}

	hrefs := make([]string, 0, len(resources))
	for idx := range resources {
		hrefs = append(hrefs, resources[idx].Href)
	}

	client.unpublishResourceLinks(req.Context, hrefs)
	client.sendResponse(coapCodes.Deleted, req.Token, message.TextPlain, nil)
}

type resourceDirectorySelector struct {
	SelectionCriteria int `json:"sel"`
}

func resourceDirectoryGetSelector(req *mux.Message, client *Client) {
	var rds resourceDirectorySelector //we want to use sel:0 to prefer cloud RD

	accept := coapconv.GetAccept(req.Options)
	encode, err := coapconv.GetEncoder(accept)
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

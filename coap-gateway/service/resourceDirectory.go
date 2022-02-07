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
	uuid "github.com/satori/go.uuid"
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

func resource2UUID(deviceID, href string) string {
	return uuid.NewV5(uuid.NamespaceURL, deviceID+href).String()
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

func resourceDirectoryPublishHandler(s mux.ResponseWriter, req *mux.Message, client *Client) {
	authCtx := client.loadAuthorizationContext()

	w := wkRd{
		TimeToLive:       -1,
		TimeToLiveLegacy: -1,
	}
	err := cbor.ReadFrom(req.Body, &w)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot publish resource: %w", authCtx.DeviceId, err), coapCodes.BadRequest, req.Token)
		return
	}

	if err := validatePublish(w); err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot publish resource: %w", authCtx.DeviceId, err), coapCodes.BadRequest, req.Token)
		return
	}

	// set time to live properly
	w = fixTTL(w)

	links := make(schema.ResourceLinks, 0, len(w.Links))
	for _, resource := range w.Links {
		if resource.DeviceID == "" {
			resource.DeviceID = w.DeviceID
		}
		resource, err := client.publishResource(req.Context, resource, int32(w.TimeToLive), client.remoteAddrString(), req.SequenceNumber, authCtx.AuthorizationContext)
		if err != nil {
			// publish resource is not critical, it cause unaccessible resource
			log.Errorf("DeviceId %v: cannot handle coap req to publish resource: %v", authCtx.DeviceId, err)
		} else {
			links = append(links, resource)
		}
	}

	if len(links) == 0 {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot publish resource: empty links", authCtx.DeviceId), coapCodes.BadRequest, req.Token)
		return
	}

	w.Links = links
	for _, link := range links {
		observable := link.Policy != nil && link.Policy.BitMask.Has(schema.Observable)
		err := client.observeResource(req.Context, link.GetDeviceID(), link.Href, observable, true)
		if err != nil {
			log.Errorf("DeviceId: %v: cannot observe published resource /%v%v: %v", link.GetDeviceID(), link.GetDeviceID(), link.Href, err)
		}
	}

	accept := coap.GetAccept(req.Options)
	encode, err := coap.GetEncoder(accept)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot publish resource: %w", authCtx.DeviceId, err), coapCodes.InternalServerError, req.Token)
		return
	}
	out, err := encode(w)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot publish resource: %w", authCtx.DeviceId, err), coapCodes.InternalServerError, req.Token)
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

func resourceDirectoryUnpublishHandler(s mux.ResponseWriter, req *mux.Message, client *Client) {
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

func resourceDirectoryGetSelector(s mux.ResponseWriter, req *mux.Message, client *Client) {
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

func resourceDirectoryHandler(s mux.ResponseWriter, req *mux.Message, client *Client) {
	switch req.Code {
	case coapCodes.POST:
		resourceDirectoryPublishHandler(s, req, client)
	case coapCodes.DELETE:
		resourceDirectoryUnpublishHandler(s, req, client)
	case coapCodes.GET:
		resourceDirectoryGetSelector(s, req, client)
	default:
		client.logAndWriteErrorResponse(fmt.Errorf("Forbidden request from %v", client.remoteAddrString()), coapCodes.Forbidden, req.Token)
	}
}

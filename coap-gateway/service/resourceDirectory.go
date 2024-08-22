package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/plgd-dev/device/v2/schema"
	coapMessage "github.com/plgd-dev/go-coap/v3/message"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/hub/v2/coap-gateway/coapconv"
	"github.com/plgd-dev/hub/v2/coap-gateway/resource"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	raService "github.com/plgd-dev/hub/v2/resource-aggregate/service"
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

func parsePublishedResources(data io.ReadSeeker, deviceID string) (wkRd, error) {
	if data == nil {
		return wkRd{}, fmt.Errorf("cannot read publish request body received from device %v: empty body", deviceID)
	}
	w := makeWkRd()
	err := cbor.ReadFrom(data, &w)
	if err != nil {
		return wkRd{}, fmt.Errorf("cannot read publish request body received from device %v: %w", deviceID, err)
	}

	if err := validatePublish(w); err != nil {
		return wkRd{}, fmt.Errorf("invalid publish request received from device %v: %w", deviceID, err)
	}

	w = fixTimeToLive(w)
	for i, link := range w.Links {
		w.Links[i].DeviceID = w.DeviceID
		w.Links[i].Href = fixHref(link.Href)
		w.Links[i].InstanceID = resource.GetInstanceID(link.Href)
	}
	return w, nil
}

func PublishResourceLinks(ctx context.Context, raClient raService.ResourceAggregateClient, links schema.ResourceLinks, deviceID string, ttl int, connectionID string, sequence uint64) ([]*commands.Resource, error) {
	var validUntil time.Time
	if ttl > 0 {
		validUntil = time.Now().Add(time.Second * time.Duration(ttl))
	}

	resources := commands.SchemaResourceLinksToResources(links, validUntil)
	request := commands.PublishResourceLinksRequest{
		Resources: resources,
		DeviceId:  deviceID,
		CommandMetadata: &commands.CommandMetadata{
			Sequence:     sequence,
			ConnectionId: connectionID,
		},
	}
	resp, err := raClient.PublishResourceLinks(ctx, &request)
	if err != nil {
		return nil, fmt.Errorf("error occurred during resource links publish %w", err)
	}

	return resp.GetPublishedResources(), nil
}

func observeResources(ctx context.Context, client *session, w wkRd, sequenceNumber uint64) (coapCodes.Code, error) {
	publishedResources, err := PublishResourceLinks(ctx, client.server.raClient, w.Links, w.DeviceID, w.TimeToLive, client.RemoteAddr().String(), sequenceNumber)
	if err != nil {
		return coapCodes.BadRequest, fmt.Errorf("unable to publish resources for device %v: %w", w.DeviceID, err)
	}

	observeError := func(deviceID string, err error) error {
		return fmt.Errorf("unable to observe published resources for device %v: %w", deviceID, err)
	}
	x := struct {
		ctx                context.Context
		client             *session
		w                  wkRd
		observeError       func(deviceID string, err error) error
		publishedResources []*commands.Resource
	}{
		ctx:                ctx,
		client:             client,
		w:                  w,
		observeError:       observeError,
		publishedResources: publishedResources,
	}
	if err := client.server.taskQueue.Submit(func() {
		obs, ok, errObs := x.client.getDeviceObserver(x.ctx)
		if errObs != nil {
			x.client.Errorf("%w", x.observeError(x.w.DeviceID, errObs))
			return
		}
		if !ok {
			x.client.Errorf("%w", x.observeError(x.w.DeviceID, errors.New("cannot get device observer")))
			return
		}
		if errObs := obs.AddPublishedResources(x.ctx, x.publishedResources); errObs != nil {
			x.client.Errorf("%w", x.observeError(x.w.DeviceID, errObs))
			return
		}
	}); err != nil {
		return coapCodes.InternalServerError, observeError(w.DeviceID, err)
	}
	return 0, nil
}

func resourceDirectoryPublishHandler(req *mux.Message, client *session) (*pool.Message, error) {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, "cannot load authorization context: %w", err)
	}

	w, err := parsePublishedResources(req.Body(), authCtx.GetDeviceID())
	if err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, "%w", err)
	}

	if errCode, errO := observeResources(req.Context(), client, w, req.Sequence()); errO != nil {
		return nil, statusErrorf(errCode, "%w", errO)
	}
	// trigger device subscriber to get pending commands for the resources that have been published
	client.triggerDeviceSubscriber()

	accept := coapconv.GetAccept(req.Options())
	encode, err := coapconv.GetEncoder(accept)
	if err != nil {
		return nil, statusErrorf(coapCodes.InternalServerError, "%w", fmt.Errorf("unable to get encoder for accepted type %v requested: %w", accept, err))
	}
	out, err := encode(w)
	if err != nil {
		return nil, statusErrorf(coapCodes.InternalServerError, "%w", fmt.Errorf("unable to encode publish response: %w", err))
	}

	return client.createResponse(coapCodes.Changed, req.Token(), accept, out), nil
}

func parseUnpublishQueryString(queries []string) (deviceID string, instanceIDs []int64, err error) {
	for _, q := range queries {
		values, err := url.ParseQuery(q)
		if err != nil {
			return "", nil, fmt.Errorf("cannot parse unpublish query: %w", err)
		}
		for _, di := range values["di"] {
			if deviceID != "" {
				return "", nil, fmt.Errorf("unable to parse unpublish query: duplicate in parameter di(%v), previously di(%v)", di, deviceID)
			}
			deviceID = di
		}
		for _, ins := range values["ins"] {
			i, err := strconv.Atoi(ins)
			if err != nil {
				return "", nil, fmt.Errorf("cannot convert %v to number", ins)
			}
			instanceIDs = append(instanceIDs, int64(i))
		}
	}

	if deviceID == "" {
		return "", nil, errors.New("deviceID not found")
	}

	return
}

func resourceDirectoryUnpublishHandler(req *mux.Message, client *session) (*pool.Message, error) {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, "%w", fmt.Errorf("cannot load authorization context: %w", err))
	}

	queries, err := req.Options().Queries()
	if err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, "%w", fmt.Errorf("cannot query string from unpublish request: %w", err))
	}
	deviceID, inss, err := parseUnpublishQueryString(queries)
	if err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, "%w", fmt.Errorf("unable to parse unpublish request queries: %w", err))
	}
	if deviceID != authCtx.GetDeviceID() {
		return nil, statusErrorf(coapCodes.BadRequest, "%w", fmt.Errorf("unable to parse unpublish request query deviceId '%v': invalid deviceID", deviceID))
	}

	resources := client.unpublishResourceLinks(req.Context(), nil, inss)
	if len(resources) == 0 {
		return nil, statusErrorf(coapCodes.BadRequest, "%w", fmt.Errorf("cannot find observed resources using query %v which shall be unpublished", queries))
	}
	return client.createResponse(coapCodes.Deleted, req.Token(), coapMessage.TextPlain, nil), nil
}

type resourceDirectorySelector struct {
	SelectionCriteria int `json:"sel"`
}

func resourceDirectoryGetSelector(req *mux.Message, client *session) (*pool.Message, error) {
	var rds resourceDirectorySelector // we want to use sel:0 to prefer cloud RD

	accept := coapconv.GetAccept(req.Options())
	encode, err := coapconv.GetEncoder(accept)
	if err != nil {
		return nil, statusErrorf(coapCodes.InternalServerError, "%w", fmt.Errorf("cannot get selector: %w", err))
	}
	out, err := encode(rds)
	if err != nil {
		return nil, statusErrorf(coapCodes.InternalServerError, "%w", fmt.Errorf("cannot encode body for get selector: %w", err))
	}

	return client.createResponse(coapCodes.Content, req.Token(), accept, out), nil
}

func resourceDirectoryHandler(req *mux.Message, client *session) (*pool.Message, error) {
	switch req.Code() {
	case coapCodes.POST:
		return resourceDirectoryPublishHandler(req, client)
	case coapCodes.DELETE:
		return resourceDirectoryUnpublishHandler(req, client)
	case coapCodes.GET:
		return resourceDirectoryGetSelector(req, client)
	default:
		return nil, statusErrorf(coapCodes.NotFound, "unsupported method %v", req.Code())
	}
}

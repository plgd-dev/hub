package service

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	pbGRPC "github.com/go-ocf/cloud/grpc-gateway/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	"github.com/go-ocf/go-coap/v2/message"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/mux"
	"github.com/go-ocf/kit/codec/cbor"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/net/coap"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/sdk/schema"
	uuid "github.com/satori/go.uuid"
)

const observable = 2

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

func isObservable(res *pbRA.Resource) bool {
	return res.Policies != nil && res.Policies.BitFlags&observable == observable
}

// fixHref always lead by "/"
func fixHref(href string) string {
	backslash := regexp.MustCompile(`\/+`)
	p := backslash.ReplaceAllString(href, "/")
	p = strings.TrimLeft(p, "/")
	p = strings.TrimRight(p, "/")

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
	if w.TimeToLive <= 0 && w.TimeToLiveLegacy <= 0 {
		return errors.New("invalid TimeToLive")
	}

	return nil
}

func resourceDirectoryPublishHandler(s mux.ResponseWriter, req *mux.Message, client *Client) {
	authCtx := client.loadAuthorizationContext()

	var w wkRd
	err := cbor.ReadFrom(req.Body, &w)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot publish resource: %v", authCtx.DeviceId, err), coapCodes.BadRequest, req.Token)
		return
	}

	if err := validatePublish(w); err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot publish resource: %v", authCtx.DeviceId, err), coapCodes.BadRequest, req.Token)
		return
	}

	// set time to live properly
	w = fixTTL(w)

	links := make(schema.ResourceLinks, 0, len(w.Links))
	for _, resource := range w.Links {
		if resource.DeviceID == "" {
			resource.DeviceID = w.DeviceID
		}
		resource, err := client.publishResource(kitNetGrpc.CtxWithToken(req.Context, authCtx.AccessToken), resource, int32(w.TimeToLive), client.remoteAddrString(), req.SequenceNumber, authCtx.AuthorizationContext)
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
		raLink := pbGRPC.SchemaResourceLinkToProto(link).ToRAProto()
		err := client.observeResource(req.Context, &raLink, true)
		if err != nil {
			log.Errorf("DeviceId: %v: ResourceId: %v cannot observe published resource", link.DeviceID, link.ID)
		}
	}

	accept := coap.GetAccept(req.Options)
	encode, err := coap.GetEncoder(accept)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot publish resource: %v", authCtx.DeviceId, err), coapCodes.InternalServerError, req.Token)
		return
	}
	out, err := encode(w)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot publish resource: %v", authCtx.DeviceId, err), coapCodes.InternalServerError, req.Token)
		return
	}
	client.sendResponse(coapCodes.Changed, req.Token, accept, out)
}

func parseUnpublishQueryString(queries []string) (deviceID string, instanceIDs []int64, err error) {
	for _, q := range queries {
		values, err := url.ParseQuery(q)
		if err != nil {
			return "", nil, fmt.Errorf("cannot parse unpublish query: %v", err)
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

	rscs := make([]*pbRA.Resource, 0, 32)

	rscs = client.getObservedResources(deviceID, inss, rscs)
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
		client.logAndWriteErrorResponse(fmt.Errorf("cannot get selector: %v", err), coapCodes.InternalServerError, req.Token)
		return
	}
	out, err := encode(rds)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot get selector: %v", err), coapCodes.InternalServerError, req.Token)
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

package service

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	gocoap "github.com/go-ocf/go-coap"
	coapCodes "github.com/go-ocf/go-coap/codes"
	"github.com/go-ocf/kit/codec/cbor"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/net/coap"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	pbRA "github.com/go-ocf/ocf-cloud/resource-aggregate/pb"
	uuid "github.com/satori/go.uuid"
)

const observable = 2

type wkRd struct {
	DeviceID         string           `json:"di"`
	Links            []*pbRA.Resource `json:"links"`
	TimeToLive       int              `json:"ttl"`
	TimeToLiveLegacy int              `json:"lt"`
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

func sendResponse(s gocoap.ResponseWriter, client *Client, code coapCodes.Code, contentFormat gocoap.MediaType, payload []byte) {
	msg := s.NewResponse(code)
	if msg != nil {
		if len(payload) > 0 {
			msg.SetOption(gocoap.ContentFormat, contentFormat)
			msg.SetPayload(payload)
		}
		err := s.WriteMsg(msg)
		if err != nil {
			log.Errorf("Cannot send reply to %v: %v", getDeviceId(client), err)
		}
		decodeMsgToDebug(client, msg, "SEND-RESPONSE")
	}
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

func resourceDirectoryPublishHandler(s gocoap.ResponseWriter, req *gocoap.Request, client *Client) {
	authCtx := client.loadAuthorizationContext()

	var w wkRd
	err := cbor.Decode(req.Msg.Payload(), &w)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot publish resource: %v", authCtx.DeviceId, err), s, client, coapCodes.BadRequest)
		return
	}

	if err := validatePublish(w); err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot publish resource: %v", authCtx.DeviceId, err), s, client, coapCodes.BadRequest)
		return
	}

	// set time to live properly
	w = fixTTL(w)

	links := make([]*pbRA.Resource, 0, len(w.Links))
	for _, resource := range w.Links {
		if resource.DeviceId == "" {
			resource.DeviceId = w.DeviceID
		}
		resource, err := client.publishResource(kitNetGrpc.CtxWithToken(req.Ctx, authCtx.AccessToken), resource.Clone(), int32(w.TimeToLive), req.Client.RemoteAddr().String(), req.Sequence, authCtx.AuthorizationContext)
		if err != nil {
			// publish resource is not critical, it cause unaccessible resource
			log.Errorf("DeviceId %v: cannot handle coap req to publish resource: %v", authCtx.DeviceId, err)
		} else {
			links = append(links, resource)
		}
	}

	if len(links) == 0 {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot publish resource: empty links", authCtx.DeviceId), s, client, coapCodes.BadRequest)
		return
	}

	w.Links = links

	for _, res := range links {
		err := client.observeResource(req.Ctx, res, true)
		if err != nil {
			log.Errorf("DeviceId: %v: ResourceId: %v cannot observe published resource", res.DeviceId, res.Id)
		}
	}

	accept := coap.GetAccept(req.Msg)
	encode, err := coap.GetEncoder(accept)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot publish resource: %v", authCtx.DeviceId, err), s, client, coapCodes.InternalServerError)
		return
	}
	out, err := encode(w)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot publish resource: %v", authCtx.DeviceId, err), s, client, coapCodes.InternalServerError)
		return
	}
	sendResponse(s, client, coapCodes.Changed, accept, out)
}

func parseUnpublishQueryString(queries []interface{}) (deviceId string, instanceIDs []int64, err error) {
	for _, query := range queries {
		var q string
		var ok bool
		if q, ok = query.(string); !ok {
			continue
		}

		values, err := url.ParseQuery(q)
		if err != nil {
			return "", nil, fmt.Errorf("cannot parse unpublish query: %v", err)
		}
		if di := values.Get("di"); di != "" {
			deviceId = di
		}

		if ins := values.Get("ins"); ins != "" {
			i, err := strconv.Atoi(ins)
			if err != nil {
				return "", nil, fmt.Errorf("cannot convert %v to number", ins)
			}
			instanceIDs = append(instanceIDs, int64(i))
		}
	}

	if deviceId == "" {
		return "", nil, fmt.Errorf("deviceId not found")
	}

	return
}

func resourceDirectoryUnpublishHandler(s gocoap.ResponseWriter, req *gocoap.Request, client *Client) {
	queries := req.Msg.Options(gocoap.URIQuery)
	deviceID, inss, err := parseUnpublishQueryString(queries)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("Incorrect Unpublish query string - %v", err), s, client, coapCodes.BadRequest)
		return
	}

	rscs := make([]*pbRA.Resource, 0, 32)

	rscs = client.getObservedResources(deviceID, inss, rscs)
	if len(rscs) == 0 {
		logAndWriteErrorResponse(fmt.Errorf("cannot found resources for the DELETE request parameters - with device ID and instance IDs %v, ", queries), s, client, coapCodes.BadRequest)
		return
	}

	client.unpublishResources(rscs)

	sendResponse(s, client, coapCodes.Deleted, gocoap.TextPlain, nil)
}

type resourceDirectorySelector struct {
	SelectionCriteria int `json:"sel"`
}

func resourceDirectoryGetSelector(s gocoap.ResponseWriter, req *gocoap.Request, client *Client) {
	var rds resourceDirectorySelector //we want to use sel:0 to prefer cloud RD

	accept := coap.GetAccept(req.Msg)
	encode, err := coap.GetEncoder(accept)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot get selector: %v", err), s, client, coapCodes.InternalServerError)
		return
	}
	out, err := encode(rds)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot get selector: %v", err), s, client, coapCodes.InternalServerError)
		return
	}

	sendResponse(s, client, coapCodes.Content, accept, out)
}

func resourceDirectoryHandler(s gocoap.ResponseWriter, req *gocoap.Request, client *Client) {
	switch req.Msg.Code() {
	case coapCodes.POST:
		resourceDirectoryPublishHandler(s, req, client)
	case coapCodes.DELETE:
		resourceDirectoryUnpublishHandler(s, req, client)
	case coapCodes.GET:
		resourceDirectoryGetSelector(s, req, client)
	default:
		logAndWriteErrorResponse(fmt.Errorf("Forbidden request from %v", req.Client.RemoteAddr()), s, client, coapCodes.Forbidden)
	}
}

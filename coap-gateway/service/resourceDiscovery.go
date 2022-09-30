package service

import (
	"errors"
	"fmt"
	"io"
	"net/url"

	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/go-coap/v3/message"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/hub/v2/coap-gateway/coapconv"
	"github.com/plgd-dev/hub/v2/coap-gateway/uri"
	pbGRPC "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
)

func makeListDevicesCommand(msg *mux.Message) (*pbGRPC.GetResourceLinksRequest, error) {
	deviceIdFilter := make([]string, 0, 4)
	typeFilter := make([]string, 0, 4)

	queries, _ := msg.Options().Queries()
	for _, query := range queries {
		values, err := url.ParseQuery(query)
		if err != nil {
			return nil, fmt.Errorf("cannot parse list devices query: %w", err)
		}
		if di := values.Get("di"); di != "" {
			deviceIdFilter = append(deviceIdFilter, di)
		}

		if rt := values.Get("rt"); rt != "" {
			typeFilter = append(typeFilter, rt)
		}
	}

	return &pbGRPC.GetResourceLinksRequest{
		DeviceIdFilter: deviceIdFilter,
		TypeFilter:     typeFilter,
	}, nil
}

func makeHref(deviceID, href string) string {
	return fixHref("/" + uri.ResourceRoute + "/" + deviceID + "/" + href)
}

func makeDiscoveryResp(isTLSListener bool, serverAddr string, getResourceLinksClient pbGRPC.GrpcGateway_GetResourceLinksClient) ([]*wkRd, error) {
	deviceRes := make(map[string]*wkRd)
	ep := "coap"
	if isTLSListener {
		ep = "coaps"
	}
	ep = ep + "+tcp://" + serverAddr

	for {
		snapshot, err := getResourceLinksClient.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("cannot create discovery response: %w", err)
		}
		d, ok := deviceRes[snapshot.GetDeviceId()]
		if !ok {
			d = &wkRd{
				DeviceID:         snapshot.GetDeviceId(),
				Links:            make(schema.ResourceLinks, 0, 16),
				TimeToLive:       -1,
				TimeToLiveLegacy: -1,
			}
			deviceRes[snapshot.GetDeviceId()] = d
		}
		links := commands.ResourcesToResourceLinks(snapshot.GetResources())
		for i := range links {
			links[i].Href = makeHref(links[i].DeviceID, links[i].Href)
			links[i].ID = ""
			if links[i].Anchor == "" {
				links[i].Anchor = "ocf://" + links[i].DeviceID
			}
			// override EndpointInformations to cloud address
			links[i].Endpoints = []schema.Endpoint{
				{
					URI:      ep,
					Priority: 1,
				},
			}
			d.Links = append(d.Links, links[i])
		}
	}

	resp := make([]*wkRd, 0, 128)
	for _, rd := range deviceRes {
		resp = append(resp, rd)
	}

	return resp, nil
}

const errFmtDiscoveryResource = "cannot handle resource discovery: %v: %w"

func getDiscoveryResourceErr(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(errFmtDiscoveryResource, "", err)
}

func resourceDirectoryFind(req *mux.Message, client *session) (*pool.Message, error) {
	_, err := client.GetAuthorizationContext()
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, "%w", getDiscoveryResourceErr(err))
	}

	request, err := makeListDevicesCommand(req)
	if err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, "%w", getDiscoveryResourceErr(err))
	}

	getResourceLinksClient, err := client.server.rdClient.GetResourceLinks(req.Context(), request)
	if err != nil {
		return nil, statusErrorf(coapconv.GrpcErr2CoapCode(err, coapconv.Retrieve), "%w", getDiscoveryResourceErr(err))
	}

	discoveryResp, err := makeDiscoveryResp(client.server.tlsEnabled(), client.server.config.APIs.COAP.ExternalAddress, getResourceLinksClient)
	if err != nil {
		return nil, statusErrorf(coapconv.GrpcErr2CoapCode(err, coapconv.Retrieve), "%w", getDiscoveryResourceErr(err))
	}

	coapCode := coapCodes.Content
	var resp interface{}
	accept := coapconv.GetAccept(req.Options())
	switch accept {
	case message.AppOcfCbor:
		links := make([]schema.ResourceLink, 0, 64)
		for _, d := range discoveryResp {
			links = append(links, d.Links...)
		}
		resp = links
	case message.AppCBOR:
		resp = discoveryResp
	}

	encode, err := coapconv.GetEncoder(accept)
	if err != nil {
		return nil, statusErrorf(coapCodes.InternalServerError, "%w", getDiscoveryResourceErr(fmt.Errorf("cannot get encoder for response: %w", err)))
	}
	out, err := encode(resp)
	if err != nil {
		return nil, statusErrorf(coapCodes.InternalServerError, "%w", getDiscoveryResourceErr(fmt.Errorf("cannot encode response: %w", err)))
	}

	return client.createResponse(coapCode, req.Token(), accept, out), nil
}

func resourceDiscoveryHandler(req *mux.Message, client *session) (*pool.Message, error) {
	switch req.Code() {
	case coapCodes.GET:
		return resourceDirectoryFind(req, client)
	default:
		return nil, statusErrorf(coapCodes.NotFound, "unsupported method %v", req.Code())
	}
}

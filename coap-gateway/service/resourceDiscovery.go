package service

import (
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/plgd-dev/cloud/coap-gateway/coapconv"
	"github.com/plgd-dev/cloud/coap-gateway/uri"
	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/net/coap"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/plgd-dev/sdk/schema"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func makeListDevicesCommand(msg *mux.Message) (*pbGRPC.GetResourceLinksRequest, error) {
	deviceIdsFilter := make([]string, 0, 4)
	typeFilter := make([]string, 0, 4)

	queries, _ := msg.Options.Queries()
	for _, query := range queries {
		values, err := url.ParseQuery(query)
		if err != nil {
			return nil, fmt.Errorf("cannot parse list devices query: %w", err)
		}
		if di := values.Get("di"); di != "" {
			deviceIdsFilter = append(deviceIdsFilter, di)
		}

		if rt := values.Get("rt"); rt != "" {
			typeFilter = append(typeFilter, rt)
		}
	}

	return &pbGRPC.GetResourceLinksRequest{
		DeviceIdsFilter: deviceIdsFilter,
		TypeFilter:      typeFilter,
	}, nil
}

func makeHref(deviceID, href string) string {
	return fixHref("/" + uri.ResourceRoute + "/" + deviceID + "/" + href)
}

func makeDiscoveryResp(isTLSListener bool, serverAddr string, getResourceLinksClient pbGRPC.GrpcGateway_GetResourceLinksClient) ([]*wkRd, codes.Code, error) {
	deviceRes := make(map[string]*wkRd)
	ep := "coap"
	if isTLSListener {
		ep = "coaps"
	}
	ep = ep + "+tcp://" + serverAddr

	for {
		link, err := getResourceLinksClient.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, status.Convert(err).Code(), fmt.Errorf("cannot create discovery response: %w", err)
		}
		d, ok := deviceRes[link.GetDeviceId()]
		if !ok {
			d = &wkRd{
				DeviceID: link.GetDeviceId(),
				Links:    make(schema.ResourceLinks, 0, 16),
			}
			deviceRes[link.GetDeviceId()] = d
		}
		resource := link.ToSchema()
		resource.Href = makeHref(resource.DeviceID, resource.Href)
		//set anchor if it is not set
		if resource.Anchor == "" {
			resource.Anchor = "ocf://" + resource.DeviceID
		}
		//override EndpointInformations to cloud address
		resource.Endpoints = []schema.Endpoint{
			{
				URI:      ep,
				Priority: 1,
			},
		}
		d.Links = append(d.Links, resource)
	}

	resp := make([]*wkRd, 0, 128)
	for _, rd := range deviceRes {
		resp = append(resp, rd)
	}

	return resp, codes.OK, nil
}

func resourceDirectoryFind(s mux.ResponseWriter, req *mux.Message, client *Client) {
	t := time.Now()
	defer func() {
		log.Debugf("resourceDirectoryFind takes %v", time.Since(t))
	}()

	authCtx := client.loadAuthorizationContext()
	request, err := makeListDevicesCommand(req)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle resource discovery: %w", authCtx.DeviceId, err), coapCodes.BadRequest, req.Token)
		return
	}

	getResourceLinksClient, err := client.server.rdClient.GetResourceLinks(kitNetGrpc.CtxWithUserID(req.Context, authCtx.GetUserID()), request)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: handle resource discovery: %w", authCtx.DeviceId, err), coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.GET), req.Token)
		return
	}

	discoveryResp, code, err := makeDiscoveryResp(client.server.IsTLSListener, client.server.ExternalAddress, getResourceLinksClient)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: handle resource discovery: %w", authCtx.DeviceId, err), coapconv.GrpcCode2CoapCode(code, coapCodes.GET), req.Token)
		return
	}

	coapCode := coapCodes.Content
	if len(discoveryResp) == 0 {
		coapCode = coapCodes.NotFound
	}

	var resp interface{}
	accept := coap.GetAccept(req.Options)
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

	encode, err := coap.GetEncoder(accept)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot marshal discovery response: %w", authCtx.DeviceId, err), coapCodes.InternalServerError, req.Token)
		return
	}
	out, err := encode(resp)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot marshal discovery response: %w", authCtx.DeviceId, err), coapCodes.InternalServerError, req.Token)
		return
	}

	client.sendResponse(coapCode, req.Token, accept, out)
}

func resourceDiscoveryHandler(s mux.ResponseWriter, req *mux.Message, client *Client) {
	switch req.Code {
	case coapCodes.GET:
		resourceDirectoryFind(s, req, client)
	default:
		client.logAndWriteErrorResponse(fmt.Errorf("Forbidden request from %v", client.remoteAddrString()), coapCodes.Forbidden, req.Token)
	}
}

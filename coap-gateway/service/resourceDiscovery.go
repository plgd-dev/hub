package service

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
	"time"

	"github.com/go-ocf/cloud/coap-gateway/coapconv"
	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRD "github.com/go-ocf/cloud/resource-directory/pb/resource-directory"
	"github.com/go-ocf/go-coap/v2/message"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/mux"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/net/coap"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func makeListDevicesCommand(msg *mux.Message, authCtx pbCQRS.AuthorizationContext) (pbRD.GetResourceLinksRequest, error) {
	deviceIdsFilter := make([]string, 0, 4)
	typeFilter := make([]string, 0, 4)

	queries, _ := msg.Options.Queries()
	for _, query := range queries {
		values, err := url.ParseQuery(query)
		if err != nil {
			return pbRD.GetResourceLinksRequest{}, fmt.Errorf("cannot parse list devices query: %v", err)
		}
		if di := values.Get("di"); di != "" {
			deviceIdsFilter = append(deviceIdsFilter, di)
		}

		if rt := values.Get("rt"); rt != "" {
			typeFilter = append(typeFilter, rt)
		}
	}

	cmd := pbRD.GetResourceLinksRequest{
		AuthorizationContext: &authCtx,
		DeviceIdsFilter:      deviceIdsFilter,
		TypeFilter:           typeFilter,
	}

	return cmd, nil
}

func makeHref(deviceID, href string) string {
	return fixHref("/" + resourceRoute + "/" + deviceID + "/" + href)
}

func makeDiscoveryResp(isTLSListener bool, serverAddr string, getResourceLinksClient pbRD.ResourceDirectory_GetResourceLinksClient) ([]*wkRd, codes.Code, error) {
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
			return nil, status.Convert(err).Code(), fmt.Errorf("cannot create discovery response: %v", err)
		}
		resource := link.Resource
		d, ok := deviceRes[resource.DeviceId]
		if !ok {
			d = &wkRd{
				DeviceID: resource.DeviceId,
				Links:    make([]*pbRA.Resource, 0, 16),
			}
			deviceRes[resource.DeviceId] = d
		}

		resource.Href = makeHref(resource.DeviceId, resource.Href)
		//set anchor if it is not set
		if resource.Anchor == "" {
			resource.Anchor = "ocf://" + resource.DeviceId
		}
		//override EndpointInformations to cloud address
		resource.EndpointInformations = []*pbRA.EndpointInformation{
			{
				Endpoint: ep,
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
	request, err := makeListDevicesCommand(req, authCtx.AuthorizationContext)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle resource discovery: %v", authCtx.DeviceId, err),  coapCodes.BadRequest, req.Token)
		return
	}

	getResourceLinksClient, err := client.server.rdClient.GetResourceLinks(kitNetGrpc.CtxWithToken(req.Context, authCtx.AccessToken), &request)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: handle resource discovery: %v", authCtx.DeviceId, err),  coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.GET), req.Token)
		return
	}

	discoveryResp, code, err := makeDiscoveryResp(client.server.IsTLSListener, client.server.FQDN+":"+strconv.Itoa(int(client.server.ExternalPort)), getResourceLinksClient)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: handle resource discovery: %v", authCtx.DeviceId, err),  coapconv.GrpcCode2CoapCode(code, coapCodes.GET), req.Token)
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
		links := make([]*pbRA.Resource, 0, 64)
		for _, d := range discoveryResp {
			links = append(links, d.Links...)
		}
		resp = links
	case message.AppCBOR:
		resp = discoveryResp
	}

	encode, err := coap.GetEncoder(accept)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot marshal discovery response: %v", authCtx.DeviceId, err),  coapCodes.InternalServerError, req.Token)
		return
	}
	out, err := encode(resp)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot marshal discovery response: %v", authCtx.DeviceId, err),  coapCodes.InternalServerError, req.Token)
		return
	}

	client.sendResponse( coapCode, req.Token, accept, out)
}

func resourceDiscoveryHandler(s mux.ResponseWriter, req *mux.Message, client *Client) {
	switch req.Code {
	case coapCodes.GET:
		resourceDirectoryFind(s, req, client)
	default:
		client.logAndWriteErrorResponse(fmt.Errorf("Forbidden request from %v", client.remoteAddrString()),  coapCodes.Forbidden, req.Token)
	}
}

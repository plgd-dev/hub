package service

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-ocf/cloud/coap-gateway/coapconv"
	gocoap "github.com/go-ocf/go-coap"
	coapCodes "github.com/go-ocf/go-coap/codes"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/net/coap"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRD "github.com/go-ocf/cloud/resource-directory/pb/resource-directory"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func makeListDevicesCommand(msg gocoap.Message, authCtx pbCQRS.AuthorizationContext) (pbRD.GetResourceLinksRequest, error) {
	deviceIdsFilter := make([]string, 0, 4)
	typeFilter := make([]string, 0, 4)

	for _, q := range msg.Options(gocoap.URIQuery) {
		var query string
		var ok bool
		if query, ok = q.(string); !ok {
			continue
		}

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

func makeHref(deviceId, href string) string {
	return fixHref("/" + resourceRoute + "/" + deviceId + "/" + href)
}

func makeDiscoveryResp(serverNetwork, serverAddr string, getResourceLinksClient pbRD.ResourceDirectory_GetResourceLinksClient) ([]*wkRd, codes.Code, error) {
	deviceRes := make(map[string]*wkRd)

	ep := "coap"
	if strings.HasSuffix(serverNetwork, "-tls") {
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
			&pbRA.EndpointInformation{
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

func resourceDirectoryFind(s gocoap.ResponseWriter, req *gocoap.Request, client *Client) {
	t := time.Now()
	defer func() {
		log.Debugf("resourceDirectoryFind takes %v", time.Since(t))
	}()

	authCtx := client.loadAuthorizationContext()
	request, err := makeListDevicesCommand(req.Msg, authCtx.AuthorizationContext)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot handle resource discovery: %v", authCtx.DeviceId, err), s, client, coapCodes.BadRequest)
		return
	}

	getResourceLinksClient, err := client.server.rdClient.GetResourceLinks(kitNetGrpc.CtxWithToken(req.Ctx, authCtx.AccessToken), &request)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: handle resource discovery: %v", authCtx.DeviceId, err), s, client, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.GET))
		return
	}

	discoveryResp, code, err := makeDiscoveryResp(client.server.Net, client.server.FQDN+":"+strconv.Itoa(int(client.server.ExternalPort)), getResourceLinksClient)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: handle resource discovery: %v", authCtx.DeviceId, err), s, client, coapconv.GrpcCode2CoapCode(code, coapCodes.GET))
		return
	}

	coapCode := coapCodes.Content
	if len(discoveryResp) == 0 {
		coapCode = coapCodes.NotFound
	}

	var resp interface{}
	respContentFormat := coap.GetAccept(req.Msg)
	switch respContentFormat {
	case gocoap.AppOcfCbor:
		links := make([]*pbRA.Resource, 0, 64)
		for _, d := range discoveryResp {
			links = append(links, d.Links...)
		}
		resp = links
	case gocoap.AppCBOR:
		resp = discoveryResp
	}

	accept := coap.GetAccept(req.Msg)
	encode, err := coap.GetEncoder(accept)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot marshal discovery response: %v", authCtx.DeviceId, err), s, client, coapCodes.InternalServerError)
		return
	}
	out, err := encode(resp)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("DeviceId: %v: cannot marshal discovery response: %v", authCtx.DeviceId, err), s, client, coapCodes.InternalServerError)
		return
	}

	sendResponse(s, client, coapCode, accept, out)
}

func resourceDiscoveryHandler(s gocoap.ResponseWriter, req *gocoap.Request, client *Client) {
	switch req.Msg.Code() {
	case coapCodes.GET:
		resourceDirectoryFind(s, req, client)
	default:
		logAndWriteErrorResponse(fmt.Errorf("Forbidden request from %v", req.Client.RemoteAddr()), s, client, coapCodes.Forbidden)
	}
}

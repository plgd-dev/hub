package service

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/kit/codec/cbor"
	"github.com/go-ocf/kit/codec/json"
	"github.com/go-ocf/kit/log"
	kitNetHttp "github.com/go-ocf/kit/net/http"
	"github.com/go-ocf/sdk/schema"

	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRD "github.com/go-ocf/cloud/resource-directory/pb/resource-directory"
	pbRS "github.com/go-ocf/cloud/resource-directory/pb/resource-shadow"
)

func toEndpoint(s *pbRA.EndpointInformation) schema.Endpoint {
	return schema.Endpoint{
		URI:      s.GetEndpoint(),
		Priority: uint64(s.GetPriority()),
	}
}

func toEndpoints(s []*pbRA.EndpointInformation) []schema.Endpoint {
	r := make([]schema.Endpoint, 0, 16)
	for _, v := range s {
		r = append(r, toEndpoint(v))
	}
	return r
}

func toPolicy(s *pbRA.Policies) schema.Policy {
	return schema.Policy{
		BitMask: schema.BitMask(s.GetBitFlags()),
	}
}

type RetrieveDeviceWithLinksResponse struct {
	Device
	Links []schema.ResourceLink `json:"links"`
}

func getHref(deviceID, href string) string {
	return "/" + deviceID + href
}

func makeResourceLink(resource *pbRA.Resource) schema.ResourceLink {
	return schema.ResourceLink{
		Href:                  getHref(resource.GetDeviceId(), resource.GetHref()),
		ResourceTypes:         resource.GetResourceTypes(),
		Interfaces:            resource.GetInterfaces(),
		DeviceID:              resource.GetDeviceId(),
		InstanceID:            resource.GetInstanceId(),
		Anchor:                resource.GetAnchor(),
		Policy:                toPolicy(resource.GetPolicies()),
		Title:                 resource.GetTitle(),
		SupportedContentTypes: resource.GetSupportedContentTypes(),
		Endpoints:             toEndpoints(resource.GetEndpointInformations()),
	}
}

func (rh *RequestHandler) GetResourceLinks(ctx context.Context, deviceIdsFilter []string, authorizationContext pbCQRS.AuthorizationContext) (map[string]schema.ResourceLinks, error) {
	client, err := rh.rdClient.GetResourceLinks(ctx, &pbRD.GetResourceLinksRequest{
		DeviceIdsFilter:      deviceIdsFilter,
		AuthorizationContext: &authorizationContext,
	})

	if err != nil {
		return nil, fmt.Errorf("cannot get resource links: %w", err)
	}
	defer client.CloseSend()

	resourceLinks := make(map[string]schema.ResourceLinks)
	for {
		resourceLink, err := client.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("cannot get resource links: %w", err)
		}
		_, ok := resourceLinks[resourceLink.GetResource().GetDeviceId()]
		if !ok {
			resourceLinks[resourceLink.GetResource().GetDeviceId()] = make(schema.ResourceLinks, 0, 32)
		}
		resourceLinks[resourceLink.GetResource().GetDeviceId()] = append(resourceLinks[resourceLink.GetResource().GetDeviceId()], makeResourceLink(resourceLink.GetResource()))
	}
	if len(resourceLinks) == 0 {
		return nil, fmt.Errorf("cannot get resource links: not found")
	}
	return resourceLinks, nil
}

type Representation struct {
	Href           string      `json:"href"`
	Representation interface{} `json:"rep"`
	Status         pbRA.Status `json:"-"`
}

type RetrieveDeviceAllResponse struct {
	Device
	Links []Representation `json:"links"`
}

func normalizeContentType(c *pbRA.Content) string {
	if c.GetContentType() != "" {
		return c.GetContentType()
	}
	switch message.MediaType(c.GetCoapContentFormat()) {
	case message.AppCBOR:
		return message.AppCBOR.String()
	case message.AppOcfCbor:
		return message.AppOcfCbor.String()
	case message.AppJSON:
		return message.AppJSON.String()
	case message.TextPlain:
		return message.TextPlain.String()
	}
	return ""
}

func unmarshalContent(c *pbRA.Content) (interface{}, error) {
	var m interface{}
	switch normalizeContentType(c) {
	case message.AppCBOR.String(), message.AppOcfCbor.String():
		err := cbor.Decode(c.GetData(), &m)
		if err != nil {
			return nil, fmt.Errorf("cannot unmarshal resource content: %w", err)
		}
	case message.AppJSON.String():
		err := json.Decode(c.GetData(), &m)
		if err != nil {
			return nil, fmt.Errorf("cannot unmarshal resource content: %w", err)
		}
	case message.TextPlain.String():
		m = string(c.Data)
	default:
		if c.CoapContentFormat == -1 {
			return c.Data, nil
		}
		return nil, fmt.Errorf("cannot unmarshal resource content: unknown content type (%v/%v)", c.ContentType, c.CoapContentFormat)
	}
	return m, nil
}

func (rh *RequestHandler) RetrieveResourcesValues(ctx context.Context, resourceIdsFilter []string, deviceIdsFilter []string, authorizationContext pbCQRS.AuthorizationContext) (map[string][]Representation, error) {
	client, err := rh.rsClient.RetrieveResourcesValues(ctx, &pbRS.RetrieveResourcesValuesRequest{
		DeviceIdsFilter:      deviceIdsFilter,
		ResourceIdsFilter:    resourceIdsFilter,
		AuthorizationContext: &authorizationContext,
	})

	if err != nil {
		return nil, fmt.Errorf("cannot retrieve resources values: %w", err)
	}
	defer client.CloseSend()

	allResources := make(map[string][]Representation)
	for {
		content, err := client.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("cannot retrieve resources values: %w", err)
		}
		rep, err := unmarshalContent(content.GetContent())
		if err != nil {
			log.Errorf("cannot retrieve resources values: %v", err)
			continue
		}

		_, ok := allResources[content.GetDeviceId()]
		if !ok {
			allResources[content.GetDeviceId()] = make([]Representation, 0, 32)
		}
		allResources[content.GetDeviceId()] = append(allResources[content.GetDeviceId()], Representation{
			Href:           getHref(content.GetDeviceId(), content.GetHref()),
			Representation: rep,
			Status:         content.Status,
		})

	}
	if len(allResources) == 0 {
		return nil, fmt.Errorf("cannot retrieve resources values: not found")
	}
	return allResources, nil
}

func (rh *RequestHandler) RetrieveDeviceWithLinks(ctx context.Context, w http.ResponseWriter, deviceID string, encoder responseWriterEncoderFunc) (int, error) {
	devices, err := rh.GetDevices(ctx, []string{deviceID}, pbCQRS.AuthorizationContext{})
	if err != nil {
		return kitNetHttp.ErrToStatusWithDef(err, http.StatusForbidden), fmt.Errorf("cannot retrieve device(%v) [base]: %w", deviceID, err)
	}
	resourceLink, err := rh.GetResourceLinks(ctx, []string{deviceID}, pbCQRS.AuthorizationContext{})
	if err != nil {
		return kitNetHttp.ErrToStatusWithDef(err, http.StatusForbidden), fmt.Errorf("cannot retrieve device(%v) [base]: %w", deviceID, err)
	}

	resp := RetrieveDeviceWithLinksResponse{
		Device: devices[0],
		Links:  resourceLink[deviceID],
	}

	err = encoder(w, resp, http.StatusOK)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot retrieve devices(%v) [base]: %w", deviceID, err)
	}
	return http.StatusOK, nil
}

type RetrieveDeviceContentAllResponse struct {
	Device
	Links []Representation `json:"links"`
}

func (rh *RequestHandler) RetrieveDeviceWithRepresentations(ctx context.Context, w http.ResponseWriter, deviceID string, encoder responseWriterEncoderFunc) (int, error) {
	devices, err := rh.GetDevices(ctx, []string{deviceID}, pbCQRS.AuthorizationContext{})
	if err != nil {
		return kitNetHttp.ErrToStatusWithDef(err, http.StatusForbidden), fmt.Errorf("cannot retrieve device(%v) [base]: %w", deviceID, err)
	}
	allResources, err := rh.RetrieveResourcesValues(ctx, nil, []string{deviceID}, pbCQRS.AuthorizationContext{})
	if err != nil {
		return kitNetHttp.ErrToStatusWithDef(err, http.StatusForbidden), fmt.Errorf("cannot retrieve device(%v) [all]: %w", deviceID, err)
	}

	resp := RetrieveDeviceContentAllResponse{
		Device: devices[0],
		Links:  allResources[deviceID],
	}

	err = encoder(w, resp, http.StatusOK)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot retrieve devices(%v) [all]: %w", deviceID, err)
	}
	return http.StatusOK, nil
}

func (rh *RequestHandler) RetrieveDeviceWithContentQuery(ctx context.Context, w http.ResponseWriter, routeVars map[string]string, contentQuery string, encoder responseWriterEncoderFunc) (int, error) {
	switch contentQuery {
	case ContentQueryBaseValue:
		return rh.RetrieveDeviceWithLinks(ctx, w, routeVars[deviceIDKey], encoder)
	case ContentQueryAllValue:
		return rh.RetrieveDeviceWithRepresentations(ctx, w, routeVars[deviceIDKey], encoder)
	}
	return http.StatusBadRequest, fmt.Errorf("invalid content query parameter")
}

func (rh *RequestHandler) RetrieveDevice(w http.ResponseWriter, r *http.Request) {
	statusCode, err := retrieveWithCallback(w, r, rh.RetrieveDeviceWithContentQuery)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot retrieve device: %w", err), statusCode, w)
	}
}

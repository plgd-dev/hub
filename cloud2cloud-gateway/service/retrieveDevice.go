package service

import (
	"context"
	"fmt"
	"io"
	"net/http"

	kitNetHttp "github.com/plgd-dev/cloud/pkg/net/http"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/codec/cbor"
	"github.com/plgd-dev/kit/codec/json"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/sdk/schema"

	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
)

func toEndpoint(s *commands.EndpointInformation) schema.Endpoint {
	return schema.Endpoint{
		URI:      s.GetEndpoint(),
		Priority: uint64(s.GetPriority()),
	}
}

func toEndpoints(s []*commands.EndpointInformation) []schema.Endpoint {
	r := make([]schema.Endpoint, 0, 16)
	for _, v := range s {
		r = append(r, toEndpoint(v))
	}
	return r
}

func toPolicy(s *commands.Policies) *schema.Policy {
	return &schema.Policy{
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

func makeResourceLink(resource *commands.Resource) schema.ResourceLink {
	r := pbGRPC.RAResourceToProto(resource).ToSchema()
	r.Href = getHref(resource.GetDeviceId(), resource.GetHref())
	r.ID = ""
	return r
}

func (rh *RequestHandler) GetResourceLinks(ctx context.Context, deviceIdsFilter []string) (map[string]schema.ResourceLinks, error) {
	client, err := rh.rdClient.GetResourceLinks(ctx, &pbGRPC.GetResourceLinksRequest{
		DeviceIdsFilter: deviceIdsFilter,
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
		if resourceLink.GetHref() == commands.StatusHref {
			continue
		}
		_, ok := resourceLinks[resourceLink.GetDeviceId()]
		if !ok {
			resourceLinks[resourceLink.GetDeviceId()] = make(schema.ResourceLinks, 0, 32)
		}
		r := resourceLink.ToSchema()
		r.Href = getHref(resourceLink.GetDeviceId(), resourceLink.GetHref())
		r.ID = ""
		resourceLinks[resourceLink.GetDeviceId()] = append(resourceLinks[resourceLink.GetDeviceId()], r)
	}
	if len(resourceLinks) == 0 {
		return nil, fmt.Errorf("cannot get resource links: not found")
	}
	return resourceLinks, nil
}

type Representation struct {
	Href           string        `json:"href"`
	Representation interface{}   `json:"rep"`
	Status         pbGRPC.Status `json:"-"`
}

type RetrieveDeviceAllResponse struct {
	Device
	Links []Representation `json:"links"`
}

func unmarshalContent(c *pbGRPC.Content) (interface{}, error) {
	var m interface{}
	switch c.GetContentType() {
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
	case "":
		return nil, nil
	default:
		return nil, fmt.Errorf("cannot unmarshal resource content: unknown content type (%v)", c.GetContentType())
	}
	return m, nil
}

func (rh *RequestHandler) RetrieveResourcesValues(ctx context.Context, resourceIdsFilter []*commands.ResourceId, deviceIdsFilter []string) (map[string][]Representation, error) {

	client, err := rh.rdClient.RetrieveResourcesValues(ctx, &pbGRPC.RetrieveResourcesValuesRequest{
		DeviceIdsFilter:   deviceIdsFilter,
		ResourceIdsFilter: resourceIdsFilter,
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
		if content.GetResourceId().GetHref() == commands.StatusHref {
			continue
		}
		rep, err := unmarshalContent(content.GetContent())
		if err != nil {
			log.Errorf("cannot retrieve resources values: %v", err)
			continue
		}

		_, ok := allResources[content.GetResourceId().GetDeviceId()]
		if !ok {
			allResources[content.GetResourceId().GetDeviceId()] = make([]Representation, 0, 32)
		}
		allResources[content.GetResourceId().GetDeviceId()] = append(allResources[content.GetResourceId().GetDeviceId()], Representation{
			Href:           getHref(content.GetResourceId().GetDeviceId(), content.GetResourceId().GetHref()),
			Representation: rep,
			Status:         content.GetStatus(),
		})

	}
	if len(allResources) == 0 {
		return nil, fmt.Errorf("cannot retrieve resources values: not found")
	}
	return allResources, nil
}

func (rh *RequestHandler) RetrieveDeviceWithLinks(ctx context.Context, w http.ResponseWriter, deviceID string, encoder responseWriterEncoderFunc) (int, error) {
	devices, err := rh.GetDevices(ctx, []string{deviceID})
	if err != nil {
		return kitNetHttp.ErrToStatusWithDef(err, http.StatusForbidden), fmt.Errorf("cannot retrieve device(%v) [base]: %w", deviceID, err)
	}
	resourceLink, err := rh.GetResourceLinks(ctx, []string{deviceID})
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
	devices, err := rh.GetDevices(ctx, []string{deviceID})
	if err != nil {
		return kitNetHttp.ErrToStatusWithDef(err, http.StatusForbidden), fmt.Errorf("cannot retrieve device(%v) [base]: %w", deviceID, err)
	}
	allResources, err := rh.RetrieveResourcesValues(ctx, nil, []string{deviceID})
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
	statusCode, err := rh.retrieveWithCallback(w, r, rh.RetrieveDeviceWithContentQuery)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot retrieve device: %w", err), statusCode, w)
	}
}

package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/go-coap/v3/message"
	pbGRPC "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/plgd-dev/kit/v2/codec/json"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RetrieveDeviceWithLinksResponse struct {
	Device
	Links []schema.ResourceLink `json:"links"`
}

func getHref(deviceID, href string) string {
	return "/" + deviceID + href
}

func (rh *RequestHandler) GetResourceLinks(ctx context.Context, deviceIdFilter []string) (map[string]schema.ResourceLinks, error) {
	client, err := rh.gwClient.GetResourceLinks(ctx, &pbGRPC.GetResourceLinksRequest{
		DeviceIdFilter: deviceIdFilter,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot get resource links: %w", err)
	}
	defer func() {
		if err := client.CloseSend(); err != nil {
			log.Errorf("failed to close client send stream: %w", err)
		}
	}()

	resourceLinks := make(map[string]schema.ResourceLinks)
	for {
		snapshot, err := client.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("cannot get resource links: %w", err)
		}
		_, ok := resourceLinks[snapshot.GetDeviceId()]
		if !ok {
			resourceLinks[snapshot.GetDeviceId()] = make(schema.ResourceLinks, 0, 32)
		}

		links := commands.ResourcesToResourceLinks(snapshot.GetResources())
		for i := range links {
			links[i].Href = getHref(links[i].DeviceID, links[i].Href)
			links[i].ID = ""
			resourceLinks[links[i].DeviceID] = append(resourceLinks[links[i].DeviceID], links[i])
		}
	}
	if len(resourceLinks) == 0 {
		return nil, fmt.Errorf("cannot get resource links: not found")
	}
	return resourceLinks, nil
}

type Representation struct {
	Href           string          `json:"href"`
	Representation interface{}     `json:"rep"`
	Status         commands.Status `json:"-"`
}

type RetrieveDeviceAllResponse struct {
	Device
	Links []Representation `json:"links"`
}

func unmarshalContent(c *commands.Content) (interface{}, error) {
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

func (rh *RequestHandler) RetrieveResources(ctx context.Context, resourceIdFilter []string, deviceIdFilter []string) (map[string][]Representation, error) {
	client, err := rh.gwClient.GetResources(ctx, &pbGRPC.GetResourcesRequest{
		DeviceIdFilter:   deviceIdFilter,
		ResourceIdFilter: resourceIdFilter,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve resources values: %w", err)
	}
	defer func() {
		if err := client.CloseSend(); err != nil {
			log.Errorf("failed to close client send stream: %w", err)
		}
	}()

	allResources := make(map[string][]Representation)
	for {
		content, err := client.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("cannot retrieve resources values: %w", err)
		}
		if content.GetData().GetResourceId().GetHref() == commands.StatusHref {
			continue
		}
		rep, err := unmarshalContent(content.GetData().GetContent())
		if err != nil {
			log.Errorf("cannot retrieve resources values: %v", err)
			continue
		}

		_, ok := allResources[content.GetData().GetResourceId().GetDeviceId()]
		if !ok {
			allResources[content.GetData().GetResourceId().GetDeviceId()] = make([]Representation, 0, 32)
		}
		allResources[content.GetData().GetResourceId().GetDeviceId()] = append(allResources[content.GetData().GetResourceId().GetDeviceId()], Representation{
			Href:           getHref(content.GetData().GetResourceId().GetDeviceId(), content.GetData().GetResourceId().GetHref()),
			Representation: rep,
			Status:         content.GetData().GetStatus(),
		})
	}
	if len(allResources) == 0 {
		return nil, status.Errorf(codes.NotFound, "cannot retrieve resources values: not found")
	}
	return allResources, nil
}

func retrieveDeviceError(deviceID, tag string, err error) error {
	return fmt.Errorf("cannot retrieve device(%v) [%v]: %w", deviceID, tag, err)
}

func (rh *RequestHandler) RetrieveDeviceWithLinks(ctx context.Context, w http.ResponseWriter, deviceID string, encoder responseWriterEncoderFunc) (int, error) {
	devices, err := rh.GetDevices(ctx, []string{deviceID})
	if err != nil {
		return kitNetHttp.ErrToStatusWithDef(err, http.StatusForbidden), retrieveDeviceError(deviceID, "base", err)
	}
	resourceLink, err := rh.GetResourceLinks(ctx, []string{deviceID})
	if err != nil {
		return kitNetHttp.ErrToStatusWithDef(err, http.StatusForbidden), retrieveDeviceError(deviceID, "base", fmt.Errorf("cannot retrieve device links: %w", err))
	}

	resp := RetrieveDeviceWithLinksResponse{
		Device: devices[0],
		Links:  resourceLink[deviceID],
	}

	err = encoder(w, resp, http.StatusOK)
	if err != nil {
		return http.StatusBadRequest, retrieveDeviceError(deviceID, "base", fmt.Errorf("cannot encode response: %w", err))
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
		return kitNetHttp.ErrToStatusWithDef(err, http.StatusForbidden), retrieveDeviceError(deviceID, "base", err)
	}
	allResources, err := rh.RetrieveResources(ctx, nil, []string{deviceID})
	if err != nil {
		return kitNetHttp.ErrToStatusWithDef(err, http.StatusForbidden), retrieveDeviceError(deviceID, "all", fmt.Errorf("cannot retrieve device links: %w", err))
	}

	resp := RetrieveDeviceContentAllResponse{
		Device: devices[0],
		Links:  allResources[deviceID],
	}

	err = encoder(w, resp, http.StatusOK)
	if err != nil {
		return http.StatusBadRequest, retrieveDeviceError(deviceID, "all", fmt.Errorf("cannot encode response: %w", err))
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
	encoder, err := getResponseWriterEncoder(strings.Split(r.Header.Get("Accept"), ","))
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot retrieve device: %w", err), http.StatusBadRequest, w)
		return
	}

	statusCode, err := rh.RetrieveDeviceWithContentQuery(r.Context(), w, mux.Vars(r), getContentQueryValue(r.URL), encoder)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot retrieve device: %w", err), statusCode, w)
	}
}

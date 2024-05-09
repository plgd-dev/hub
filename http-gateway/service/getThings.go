package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	jsoniter "github.com/json-iterator/go"
	bridgeDeviceTD "github.com/plgd-dev/device/v2/bridge/device/thingDescription"
	bridgeResourcesTD "github.com/plgd-dev/device/v2/bridge/resources/thingDescription"
	schemaCloud "github.com/plgd-dev/device/v2/schema/cloud"
	schemaCredential "github.com/plgd-dev/device/v2/schema/credential"
	schemaDevice "github.com/plgd-dev/device/v2/schema/device"
	schemaMaintenance "github.com/plgd-dev/device/v2/schema/maintenance"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/http-gateway/serverMux"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/kit/v2/codec/json"
	wotTD "github.com/web-of-things-open-source/thingdescription-go/thingDescription"
)

type ThingLink struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
}

type GetThingsResponse struct {
	Base  string      `json:"base"`
	Links []ThingLink `json:"links"`
}

const (
	ThingLinkRelationItem       = "item"
	ThingLinkRelationCollection = "collection"
)

func (requestHandler *RequestHandler) getResourceLinks(ctx context.Context, deviceFilter []string, typeFilter []string) ([]*events.ResourceLinksPublished, error) {
	client, err := requestHandler.client.GrpcGatewayClient().GetResourceLinks(ctx, &pb.GetResourceLinksRequest{
		DeviceIdFilter: deviceFilter,
		TypeFilter:     typeFilter,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot get resource links: %w", err)
	}
	links := make([]*events.ResourceLinksPublished, 0, 16)
	for {
		link, err := client.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("cannot receive resource link: %w", err)
		}
		links = append(links, link)
	}
	return links, nil
}

func (requestHandler *RequestHandler) getThings(w http.ResponseWriter, r *http.Request) {
	resLinks, err := requestHandler.getResourceLinks(r.Context(), nil, []string{bridgeResourcesTD.ResourceType})
	if err != nil {
		serverMux.WriteError(w, err)
		return
	}
	links := make([]ThingLink, 0, len(resLinks))
	for _, l := range resLinks {
		links = append(links, ThingLink{
			Href: "/" + l.GetDeviceId(),
			Rel:  ThingLinkRelationItem,
		})
	}
	things := GetThingsResponse{
		Base:  requestHandler.config.UI.WebConfiguration.HTTPGatewayAddress + uri.Things,
		Links: links,
	}
	if err := jsonResponseWriter(w, things); err != nil {
		log.Errorf("failed to write response: %v", err)
	}
}

func patchProperty(pe wotTD.PropertyElement, deviceID, href, contentType string) (wotTD.PropertyElement, error) {
	deviceUUID, err := uuid.Parse(deviceID)
	if err != nil {
		return wotTD.PropertyElement{}, fmt.Errorf("cannot parse deviceID: %w", err)
	}
	propertyBaseURL := "/" + uri.DevicesPathKey + "/" + deviceID + "/" + uri.ResourcesPathKey
	patchFnMap := map[string]func(wotTD.PropertyElement) (wotTD.PropertyElement, error){
		schemaDevice.ResourceURI: func(pe wotTD.PropertyElement) (wotTD.PropertyElement, error) {
			return bridgeResourcesTD.PatchDeviceResourcePropertyElement(pe, deviceUUID, propertyBaseURL, contentType, "")
		},
		schemaMaintenance.ResourceURI: func(pe wotTD.PropertyElement) (wotTD.PropertyElement, error) {
			return bridgeResourcesTD.PatchMaintenanceResourcePropertyElement(pe, deviceUUID, propertyBaseURL, contentType)
		},
		schemaCloud.ResourceURI: func(pe wotTD.PropertyElement) (wotTD.PropertyElement, error) {
			return bridgeResourcesTD.PatchCloudResourcePropertyElement(pe, deviceUUID, propertyBaseURL, contentType)
		},
		schemaCredential.ResourceURI: func(pe wotTD.PropertyElement) (wotTD.PropertyElement, error) {
			return bridgeResourcesTD.PatchCredentialResourcePropertyElement(pe, deviceUUID, propertyBaseURL, contentType)
		},
	}
	patchFn, ok := patchFnMap[href]
	if ok {
		pe, err = patchFn(pe)
		if err != nil {
			return wotTD.PropertyElement{}, err
		}
		return pe, nil
	}

	propOps := bridgeDeviceTD.GetPropertyElementOperations(pe)
	pe, err = bridgeDeviceTD.PatchPropertyElement(pe, nil, true, deviceUUID, propertyBaseURL+href,
		propOps.ToSupportedOperations(), contentType)
	if err != nil {
		return wotTD.PropertyElement{}, err
	}
	return pe, nil
}

func isDeviceLink(le wotTD.IconLinkElement) (string, bool) {
	if le.Href == "" {
		return "", false
	}
	if le.Href[0] != '/' {
		return "", false
	}
	if le.Rel == nil {
		return "", false
	}
	validRefs := map[string]struct{}{
		ThingLinkRelationItem:       {},
		ThingLinkRelationCollection: {},
	}
	if _, ok := validRefs[*le.Rel]; !ok {
		return "", false
	}
	linkedDeviceID := le.Href
	if linkedDeviceID[0] == '/' {
		linkedDeviceID = linkedDeviceID[1:]
	}
	uuidDeviceID, err := uuid.Parse(linkedDeviceID)
	if err != nil {
		return "", false
	}
	if uuidDeviceID == uuid.Nil {
		return "", false
	}
	return linkedDeviceID, true
}

func getLinkedDevices(links []wotTD.IconLinkElement) []string {
	devices := make([]string, 0, len(links))
	for _, l := range links {
		if deviceID, ok := isDeviceLink(l); ok {
			devices = append(devices, deviceID)
		}
	}
	return devices
}

func ThingPatchLink(le wotTD.IconLinkElement, validateDevice map[string]struct{}) (wotTD.IconLinkElement, bool) {
	if le.Href == "" {
		return wotTD.IconLinkElement{}, false
	}
	device, ok := isDeviceLink(le)
	if !ok {
		return le, true
	}
	if len(validateDevice) == 0 {
		return wotTD.IconLinkElement{}, false
	}
	if _, ok := validateDevice[device]; !ok {
		return wotTD.IconLinkElement{}, false
	}
	le.Href = "/" + uri.ThingsPathKey + le.Href
	return le, true
}

func makeDevicePropertiesValidator(deviceID string, links []*events.ResourceLinksPublished) (map[string]struct{}, bool) {
	for _, l := range links {
		if l.GetDeviceId() == deviceID {
			validateProperties := map[string]struct{}{}
			for _, r := range l.GetResources() {
				validateProperties[r.GetHref()] = struct{}{}
			}
			return validateProperties, true
		}
	}
	return nil, false
}

func makeDeviceLinkValidator(links []*events.ResourceLinksPublished) map[string]struct{} {
	validator := make(map[string]struct{})
	for _, l := range links {
		validator[l.GetDeviceId()] = struct{}{}
	}
	return validator
}

func (requestHandler *RequestHandler) thingSetBase(td *wotTD.ThingDescription) error {
	baseURL := requestHandler.config.UI.WebConfiguration.HTTPGatewayAddress + uri.API
	base, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("cannot parse base url: %w", err)
	}
	td.Base = *base
	return nil
}

func (requestHandler *RequestHandler) thingSetProperties(ctx context.Context, deviceID string, td *wotTD.ThingDescription) error {
	deviceLinks, err := requestHandler.getResourceLinks(ctx, []string{deviceID}, nil)
	if err != nil {
		return fmt.Errorf("cannot get resource links: %w", err)
	}
	validateProperties, ok := makeDevicePropertiesValidator(deviceID, deviceLinks)
	if !ok {
		return fmt.Errorf("cannot get resource links for device %v", deviceID)
	}
	for href, prop := range td.Properties {
		if len(prop.Forms) > 0 {
			continue
		}
		_, ok := validateProperties[href]
		if !ok {
			_, ok = validateProperties["/"+href]
		}
		if !ok {
			delete(td.Properties, href)
			continue
		}
		patchedProp, err := patchProperty(prop, deviceID, href, message.AppJSON.String())
		if err != nil {
			return fmt.Errorf("cannot patch device resource property element: %w", err)
		}
		td.Properties[href] = patchedProp
	}
	return nil
}

func (requestHandler *RequestHandler) thingSetLinks(ctx context.Context, td *wotTD.ThingDescription) {
	linkedDevices := getLinkedDevices(td.Links)
	var validLinkedDevices map[string]struct{}
	if len(linkedDevices) > 0 {
		links, err := requestHandler.getResourceLinks(ctx, linkedDevices, []string{bridgeResourcesTD.ResourceType})
		if err == nil {
			validLinkedDevices = makeDeviceLinkValidator(links)
		}
	}
	patchedLinks := make([]wotTD.IconLinkElement, 0, len(td.Links))
	for _, link := range td.Links {
		patchedLink, ok := ThingPatchLink(link, validLinkedDevices)
		if !ok {
			continue
		}
		patchedLinks = append(patchedLinks, patchedLink)
	}
	if len(patchedLinks) == 0 {
		td.Links = nil
	} else {
		td.Links = patchedLinks
	}
}

func (requestHandler *RequestHandler) thingDescriptionResponse(ctx context.Context, w http.ResponseWriter, rec *httptest.ResponseRecorder, writeError func(w http.ResponseWriter, err error), deviceID string) {
	content := jsoniter.Get(rec.Body.Bytes(), streamResponseKey, "data", "content")
	if content.ValueType() != jsoniter.ObjectValue {
		writeError(w, errors.New("cannot decode thingDescription content"))
		return
	}
	td := wotTD.ThingDescription{}
	err := json.Decode([]byte(content.ToString()), &td)
	if err != nil {
		writeError(w, fmt.Errorf("cannot decode thingDescription content: %w", err))
		return
	}

	// .base
	if err = requestHandler.thingSetBase(&td); err != nil {
		writeError(w, fmt.Errorf("cannot set base url: %w", err))
		return
	}

	// .properties.forms
	if err = requestHandler.thingSetProperties(ctx, deviceID, &td); err != nil {
		writeError(w, fmt.Errorf("cannot set properties: %w", err))
	}

	// .links
	requestHandler.thingSetLinks(ctx, &td)

	writeSimpleResponse(w, rec, td, writeError)
}

func (requestHandler *RequestHandler) getThing(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars[uri.DeviceIDKey]
	rec, err := requestHandler.serveResourceRequest(r, deviceID, bridgeResourcesTD.ResourceURI, "", "")
	if err != nil {
		serverMux.WriteError(w, err)
		return
	}
	requestHandler.thingDescriptionResponse(r.Context(), w, rec, serverMux.WriteError, deviceID)
}

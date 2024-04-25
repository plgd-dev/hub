package service

import (
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

const ThingLinkRelationItem = "item"

func (requestHandler *RequestHandler) getThings(w http.ResponseWriter, r *http.Request) {
	client, err := requestHandler.client.GrpcGatewayClient().GetResourceLinks(r.Context(), &pb.GetResourceLinksRequest{
		TypeFilter: []string{bridgeResourcesTD.ResourceType},
	})
	if err != nil {
		serverMux.WriteError(w, fmt.Errorf("cannot get resource links: %w", err))
		return
	}
	links := make([]ThingLink, 0)
	for {
		link, err := client.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			serverMux.WriteError(w, fmt.Errorf("cannot receive resource link: %w", err))
			return
		}
		links = append(links, ThingLink{
			Href: "/" + link.GetDeviceId(),
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
	const propertyBaseURL = ""
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

func (requestHandler *RequestHandler) thingDescriptionResponse(w http.ResponseWriter, rec *httptest.ResponseRecorder, writeError func(w http.ResponseWriter, err error), deviceID string) {
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
	baseURL := requestHandler.config.UI.WebConfiguration.HTTPGatewayAddress + uri.Devices + "/" + deviceID + "/" + uri.ResourcesPathKey
	base, err := url.Parse(baseURL)
	if err != nil {
		writeError(w, fmt.Errorf("cannot parse base url: %w", err))
		return
	}
	td.Base = *base

	// .properties.forms
	for href, prop := range td.Properties {
		if len(prop.Forms) > 0 {
			continue
		}
		patchedProp, err := patchProperty(prop, deviceID, href, message.AppJSON.String())
		if err != nil {
			writeError(w, fmt.Errorf("cannot patch device resource property element: %w", err))
			return
		}
		td.Properties[href] = patchedProp
	}

	// links
	// TODO

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
	requestHandler.thingDescriptionResponse(w, rec, serverMux.WriteError, deviceID)
}

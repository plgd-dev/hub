package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/plgd-dev/hub/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/cloud2cloud-connector/store"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	kitHttp "github.com/plgd-dev/hub/pkg/net/http"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	raEvents "github.com/plgd-dev/hub/resource-aggregate/events"
	raService "github.com/plgd-dev/hub/resource-aggregate/service"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/v2/log"
)

func makeHTTPEndpoint(url, deviceID, href string) string {
	return url + kitHttp.CanonicalHref("devices/"+deviceID+"/"+href)
}

func updateDeviceResource(ctx context.Context, deviceID, href, contentType string, content []byte, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) (string, []byte, commands.Status, error) {
	client := linkedCloud.GetHTTPClient()
	defer client.CloseIdleConnections()
	r, w := io.Pipe()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, makeHTTPEndpoint(linkedCloud.Endpoint.URL, deviceID, href), r)
	if err != nil {
		return "", nil, commands.Status_BAD_REQUEST, fmt.Errorf("cannot create post request: %w", err)
	}
	req.Header.Set(AcceptHeader, events.ContentType_JSON+","+events.ContentType_VNDOCFCBOR)
	req.Header.Set(events.ContentTypeKey, contentType)
	req.Header.Set(AuthorizationHeader, "Bearer "+string(linkedAccount.Data.Target().AccessToken))
	req.Header.Set("Connection", "close")
	req.Close = true

	go func() {
		defer func() {
			if err := w.Close(); err != nil {
				log.Errorf("failed to close write pipe: %v", err)
			}
		}()
		_, err := w.Write(content)
		if err != nil {
			log.Errorf("cannot update content of device %v resource %v: %w", deviceID, href, err)
		}
	}()
	httpResp, err := client.Do(req)
	if err != nil {
		return "", nil, commands.Status_UNAVAILABLE, fmt.Errorf("cannot post: %w", err)
	}
	defer func() {
		if err := httpResp.Body.Close(); err != nil {
			log.Errorf("failed to close response body stream: %v")
		}
	}()
	if httpResp.StatusCode != http.StatusOK {
		status := commands.HTTPStatus2Status(httpResp.StatusCode)
		return "", nil, status, fmt.Errorf("unexpected statusCode %v", httpResp.StatusCode)
	}
	respContentType := httpResp.Header.Get(events.ContentTypeKey)
	respContent := bytes.NewBuffer(make([]byte, 0, 1024))
	_, err = respContent.ReadFrom(httpResp.Body)
	if err != nil {
		return "", nil, commands.Status_UNAVAILABLE, fmt.Errorf("cannot read update response: %w", err)
	}

	return respContentType, respContent.Bytes(), commands.Status_OK, nil
}

func updateResource(ctx context.Context, raClient raService.ResourceAggregateClient, e *raEvents.ResourceUpdatePending, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	deviceID := e.GetResourceId().GetDeviceId()
	href := e.GetResourceId().GetHref()
	contentType, content, status, err := updateDeviceResource(ctx, deviceID, href, e.GetContent().GetContentType(), e.GetContent().GetData(), linkedAccount, linkedCloud)
	if err != nil {
		log.Errorf("cannot update resource %v/%v: %w", deviceID, href, err)
	}
	coapContentFormat := int32(-1)

	switch contentType {
	case message.AppCBOR.String():
		coapContentFormat = int32(message.AppCBOR)
	case message.AppOcfCbor.String():
		coapContentFormat = int32(message.AppOcfCbor)
	case message.AppJSON.String():
		coapContentFormat = int32(message.AppJSON)
	}

	ctx = kitNetGrpc.CtxWithToken(ctx, linkedAccount.Data.Origin().AccessToken.String())
	_, err = raClient.ConfirmResourceUpdate(ctx, &commands.ConfirmResourceUpdateRequest{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		CorrelationId: e.GetAuditContext().GetCorrelationId(),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: linkedAccount.ID,
			Sequence:     uint64(time.Now().UnixNano()),
		},
		Content: &commands.Content{
			Data:              content,
			ContentType:       contentType,
			CoapContentFormat: coapContentFormat,
		},
		Status: status,
	})
	if err != nil {
		return fmt.Errorf("cannot update resource /%v%v: %w", deviceID, href, err)
	}
	return nil
}

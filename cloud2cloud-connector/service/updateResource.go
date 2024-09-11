package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/store"
	"github.com/plgd-dev/hub/v2/internal/math"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttpUri "github.com/plgd-dev/hub/v2/pkg/net/http/uri"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	raEvents "github.com/plgd-dev/hub/v2/resource-aggregate/events"
	raService "github.com/plgd-dev/hub/v2/resource-aggregate/service"
	"go.opentelemetry.io/otel/trace"
)

func makeHTTPEndpoint(url, deviceID, href string) string {
	return url + pkgHttpUri.CanonicalHref("devices/"+deviceID+"/"+href)
}

func updateDeviceResource(ctx context.Context, tracerProvider trace.TracerProvider, deviceID, href, contentType string, content []byte, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) (string, []byte, commands.Status, error) {
	client := linkedCloud.GetHTTPClient(tracerProvider)
	defer client.CloseIdleConnections()
	r, w := io.Pipe()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, makeHTTPEndpoint(linkedCloud.Endpoint.URL, deviceID, href), r)
	if err != nil {
		return "", nil, commands.Status_BAD_REQUEST, fmt.Errorf("cannot create post request: %w", err)
	}
	req.Header.Set(AcceptHeader, events.ContentType_JSON+","+events.ContentType_VNDOCFCBOR)
	req.Header.Set(events.ContentTypeKey, contentType)
	req.Header.Set(AuthorizationHeader, AuthorizationBearerPrefix+string(linkedAccount.Data.Target().AccessToken))
	req.Header.Set("Connection", "close")
	req.Close = true

	go func() {
		defer func() {
			if errC := w.Close(); errC != nil {
				log.Errorf("failed to close write pipe: %w", errC)
			}
		}()
		_, errW := w.Write(content)
		if errW != nil {
			log.Errorf("cannot update content of device %v resource %v: %w", deviceID, href, errW)
		}
	}()
	httpResp, err := client.Do(req)
	if err != nil {
		return "", nil, commands.Status_UNAVAILABLE, fmt.Errorf("cannot post: %w", err)
	}
	defer func() {
		if errC := httpResp.Body.Close(); errC != nil {
			log.Errorf("failed to close response body stream: %w", errC)
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

func updateResource(ctx context.Context, tracerProvider trace.TracerProvider, raClient raService.ResourceAggregateClient, e *raEvents.ResourceUpdatePending, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	deviceID := e.GetResourceId().GetDeviceId()
	href := e.GetResourceId().GetHref()
	contentType, content, status, err := updateDeviceResource(ctx, tracerProvider, deviceID, href, e.GetContent().GetContentType(), e.GetContent().GetData(), linkedAccount, linkedCloud)
	if err != nil {
		log.Errorf("cannot update resource /%v%v: %w", deviceID, href, err)
	}
	coapContentFormat := stringToSupportedMediaType(contentType)
	ctx = kitNetGrpc.CtxWithToken(ctx, linkedAccount.Data.Origin().AccessToken.String())
	_, err = raClient.ConfirmResourceUpdate(ctx, &commands.ConfirmResourceUpdateRequest{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		CorrelationId: e.GetAuditContext().GetCorrelationId(),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: linkedAccount.ID,
			Sequence:     math.CastTo[uint64](time.Now().UnixNano()),
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

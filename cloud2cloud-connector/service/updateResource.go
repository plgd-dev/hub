package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/plgd-dev/go-coap/v2/message"

	"github.com/plgd-dev/cloud/cloud2cloud-connector/events"
	"github.com/plgd-dev/cloud/cloud2cloud-connector/store"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/kit/log"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	kitHttp "github.com/plgd-dev/kit/net/http"

	raCqrs "github.com/plgd-dev/cloud/resource-aggregate/cqrs"
	pbCQRS "github.com/plgd-dev/cloud/resource-aggregate/pb"
	pbRA "github.com/plgd-dev/cloud/resource-aggregate/pb"
)

func makeHTTPEndpoint(url, deviceID, href string) string {
	return url + kitHttp.CanonicalHref("devices/"+deviceID+"/"+href)
}

func updateDeviceResource(ctx context.Context, deviceID, href, contentType string, content []byte, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) (string, []byte, pbRA.Status, error) {
	client := linkedCloud.GetHTTPClient()
	r, w := io.Pipe()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, makeHTTPEndpoint(linkedCloud.Endpoint.URL, deviceID, href), r)
	if err != nil {
		return "", nil, pbRA.Status_BAD_REQUEST, fmt.Errorf("cannot create post request: %w", err)
	}
	req.Header.Set(AcceptHeader, events.ContentType_JSON+","+events.ContentType_VNDOCFCBOR)
	req.Header.Set(events.ContentTypeKey, contentType)
	req.Header.Set(AuthorizationHeader, "Bearer "+string(linkedAccount.TargetCloud.AccessToken))
	req.Header.Set("Connection", "close")
	req.Close = true

	go func() {
		defer w.Close()
		_, err := w.Write(content)
		if err != nil {
			log.Errorf("cannot update content of device %v resource %v: %v", deviceID, href, err)
		}
	}()
	httpResp, err := client.Do(req)
	if err != nil {
		return "", nil, pbRA.Status_UNAVAILABLE, fmt.Errorf("cannot post: %w", err)
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		status := pbRA.HTTPStatus2Status(httpResp.StatusCode)
		return "", nil, status, fmt.Errorf("unexpected statusCode %v", httpResp.StatusCode)
	}
	respContentType := httpResp.Header.Get(events.ContentTypeKey)
	respContent := bytes.NewBuffer(make([]byte, 0, 1024))
	_, err = respContent.ReadFrom(httpResp.Body)
	if err != nil {
		return "", nil, pbRA.Status_UNAVAILABLE, fmt.Errorf("cannot read update response: %w", err)
	}

	return respContentType, respContent.Bytes(), pbRA.Status_OK, nil
}

func updateResource(ctx context.Context, raClient pbRA.ResourceAggregateClient, e *pb.Event_ResourceUpdatePending, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	deviceID := e.GetResourceId().GetDeviceId()
	href := e.GetResourceId().GetHref()
	contentType, content, status, err := updateDeviceResource(ctx, deviceID, href, e.GetContent().GetContentType(), e.GetContent().GetData(), linkedAccount, linkedCloud)
	if err != nil {
		log.Errorf("cannot update resource %v/%v: %v", deviceID, href, err)
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

	_, err = raClient.ConfirmResourceUpdate(kitNetGrpc.CtxWithUserID(ctx, linkedAccount.UserID), &pbRA.ConfirmResourceUpdateRequest{
		ResourceId:    raCqrs.MakeResourceId(deviceID, href),
		CorrelationId: e.GetCorrelationId(),
		CommandMetadata: &pbCQRS.CommandMetadata{
			ConnectionId: linkedAccount.ID,
			Sequence:     uint64(time.Now().UnixNano()),
		},
		Content: &pbRA.Content{
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

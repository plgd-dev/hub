package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/go-ocf/go-coap/v2/message"

	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
	"github.com/go-ocf/cloud/cloud2cloud-connector/store"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"

	raCqrs "github.com/go-ocf/cloud/resource-aggregate/cqrs"
	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
)

func retrieveDeviceResource(ctx context.Context, deviceID, href string, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) (string, []byte, pbRA.Status, error) {
	client := linkedCloud.GetHTTPClient()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, makeHTTPEndpoint(linkedCloud.C2CURL, deviceID, href), nil)
	if err != nil {
		return "", nil, pbRA.Status_BAD_REQUEST, fmt.Errorf("cannot create post request: %v", err)
	}
	req.Header.Set(AcceptHeader, events.ContentType_JSON+","+events.ContentType_VNDOCFCBOR)
	req.Header.Set(AuthorizationHeader, "Bearer "+string(linkedAccount.TargetCloud.AccessToken))
	httpResp, err := client.Do(req)
	if err != nil {
		return "", nil, pbRA.Status_UNAVAILABLE, fmt.Errorf("cannot post: %v", err)
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		status := pbRA.Status_UNKNOWN
		switch status {
		case http.StatusAccepted:
			status = pbRA.Status_ACCEPTED
		case http.StatusOK:
			status = pbRA.Status_OK
		case http.StatusBadRequest:
			status = pbRA.Status_BAD_REQUEST
		case http.StatusNotFound:
			status = pbRA.Status_NOT_FOUND
		case http.StatusNotImplemented:
			status = pbRA.Status_NOT_IMPLEMENTED
		case http.StatusForbidden:
			status = pbRA.Status_FORBIDDEN
		case http.StatusUnauthorized:
			status = pbRA.Status_UNAUTHORIZED
		}
		return "", nil, status, fmt.Errorf("unexpected statusCode %v", httpResp.StatusCode)
	}
	respContentType := httpResp.Header.Get(events.ContentTypeKey)
	respContent := bytes.NewBuffer(make([]byte, 0, 1024))
	_, err = respContent.ReadFrom(httpResp.Body)
	if err != nil {
		return "", nil, pbRA.Status_UNAVAILABLE, fmt.Errorf("cannot read update response: %v", err)
	}

	return respContentType, respContent.Bytes(), pbRA.Status_OK, nil
}

func retrieveResource(ctx context.Context, raClient pbRA.ResourceAggregateClient, e *pb.Event_ResourceRetrievePending, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	deviceID := e.GetResourceId().GetDeviceId()
	href := e.GetResourceId().GetHref()
	contentType, content, status, err := retrieveDeviceResource(ctx, deviceID, href, linkedAccount, linkedCloud)
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

	_, err = raClient.ConfirmResourceRetrieve(kitNetGrpc.CtxWithUserID(ctx, linkedAccount.UserID), &pbRA.ConfirmResourceRetrieveRequest{
		ResourceId:    raCqrs.MakeResourceId(deviceID, href),
		CorrelationId: e.GetCorrelationId(),
		CommandMetadata: &pbCQRS.CommandMetadata{
			ConnectionId: Cloud2cloudConnectorConnectionId,
			//Sequence:     header.SequenceNumber,
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

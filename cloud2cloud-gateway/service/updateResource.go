package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
	pbGRPC "github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/go-ocf/go-coap/v2/message"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/gofrs/uuid"
	"github.com/gorilla/mux"

	cqrsRA "github.com/go-ocf/cloud/resource-aggregate/cqrs"
	raEvents "github.com/go-ocf/cloud/resource-aggregate/cqrs/events"
	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
)

const checkAgain = http.StatusPreconditionRequired

func getCoapContentFormat(contentType string) int32 {
	switch contentType {
	case message.AppJSON.String():
		return int32(message.AppJSON)
	case message.AppCBOR.String():
		return int32(message.AppCBOR)
	case message.AppOcfCbor.String():
		return int32(message.AppOcfCbor)
	}

	return -1
}

var seqNumber uint64

func (rh *RequestHandler) onFirstTimeout(ctx context.Context, w http.ResponseWriter, deviceID, resourceID string) (int, error) {
	if err := rh.resourceProjection.ForceUpdate(ctx, deviceID, resourceID); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("cannot get response for update resource %v.%v: %w", deviceID, resourceID, err)
	}
	return checkAgain, nil
}

func (rh *RequestHandler) onSecondTimeout(ctx context.Context, w http.ResponseWriter, deviceID, resourceID string) (int, error) {
	// timeout again it means update will be processed in future
	w.WriteHeader(http.StatusAccepted)
	return http.StatusAccepted, nil
}

func statusToHttpStatus(status pbGRPC.Status) int {
	switch status {
	case pbGRPC.Status_UNKNOWN:
		return http.StatusBadRequest
	case pbGRPC.Status_OK:
		return http.StatusOK
	case pbGRPC.Status_BAD_REQUEST:
		return http.StatusBadRequest
	case pbGRPC.Status_UNAUTHORIZED:
		return http.StatusUnauthorized
	case pbGRPC.Status_FORBIDDEN:
		return http.StatusForbidden
	case pbGRPC.Status_NOT_FOUND:
		return http.StatusNotFound
	case pbGRPC.Status_UNAVAILABLE:
		return http.StatusServiceUnavailable
	case pbGRPC.Status_NOT_IMPLEMENTED:
		return http.StatusNotImplemented
	case pbGRPC.Status_ACCEPTED:
		return http.StatusAccepted
	case pbGRPC.Status_ERROR:
		return http.StatusInternalServerError
	}
	return http.StatusInternalServerError
}

func clientUpdateSendResponse(w http.ResponseWriter, deviceID, resourceID string, processed raEvents.ResourceUpdated) (int, error) {
	statusCode := statusToHttpStatus(pbGRPC.RAStatus2Status(processed.Status))

	if processed.Content != nil {
		content, err := unmarshalContent(pbGRPC.RAContent2Content(processed.Content))
		if err != nil {
			logAndWriteErrorResponse(fmt.Errorf("cannot make action on resource content changed: %w", err), statusCode, w)
			return statusCode, nil
		}
		switch v := content.(type) {
		case string:
			w.WriteHeader(statusCode)
			w.Write([]byte(v))
			return statusCode, nil
		case []byte:
			w.WriteHeader(statusCode)
			w.Write(v)
			return statusCode, nil
		default:
			err = jsonResponseWriterEncoder(w, content, statusCode)
			if err != nil {
				logAndWriteErrorResponse(fmt.Errorf("cannot make action on resource content changed: %w", err), statusCode, w)
				return statusCode, nil
			}
			return statusCode, nil
		}
	}
	return statusCode, nil
}

func (rh *RequestHandler) waitForUpdateContentResponse(ctx context.Context, w http.ResponseWriter, deviceID, resourceID string, notify <-chan raEvents.ResourceUpdated, onTimeout func(ctx context.Context, w http.ResponseWriter, destDeviceId, resourceId string) (int, error)) (int, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, rh.timeoutForRequests)
	defer cancel()
	select {
	case processed := <-notify:
		return clientUpdateSendResponse(w, deviceID, resourceID, processed)
	case <-timeoutCtx.Done():
		return onTimeout(ctx, w, deviceID, resourceID)
	}
}

func (rh *RequestHandler) updateResourceContent(w http.ResponseWriter, r *http.Request) (int, error) {
	token, err := getAccessToken(r)
	if err != nil {
		return http.StatusUnauthorized, fmt.Errorf("cannot get access token: %w", err)
	}

	contentType := r.Header.Get(events.ContentTypeKey)
	coapContentFormat := getCoapContentFormat(contentType)

	routeVars := mux.Vars(r)
	deviceID := routeVars[deviceIDKey]
	resourceID := cqrsRA.MakeResourceId(deviceID, routeVars[resourceLinkHrefKey])
	correlationIdUUID, err := uuid.NewV4()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("cannot create correlationId for update resource %v.%v: %w", deviceID, resourceID, err)
	}
	correlationId := correlationIdUUID.String()

	buffer := bytes.NewBuffer(make([]byte, 0, 1024))
	_, err = buffer.ReadFrom(r.Body)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot read body: %w", err)
	}

	notify := rh.updateNotificationContainer.Add(correlationId)
	defer rh.updateNotificationContainer.Remove(correlationId)

	ctx := context.Background()

	_, err = rh.resourceProjection.Register(ctx, deviceID)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("DeviceId %v: cannot regiter device to projection for update resource: %w", deviceID, err)
	}
	defer rh.resourceProjection.Unregister(deviceID)

	_, err = rh.raClient.UpdateResource(kitNetGrpc.CtxWithToken(r.Context(), token), &pbRA.UpdateResourceRequest{
		ResourceId: resourceID,
		Content: &pbRA.Content{
			CoapContentFormat: coapContentFormat,
			ContentType:       contentType,
			Data:              buffer.Bytes(),
		},
		CommandMetadata: &pbCQRS.CommandMetadata{
			ConnectionId: r.RemoteAddr,
			Sequence:     atomic.AddUint64(&seqNumber, 1),
		},
		CorrelationId: correlationId,
	})
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot update resource content: %w", err)
	}

	statusCode, err := rh.waitForUpdateContentResponse(ctx, w, deviceID, resourceID, notify, rh.onFirstTimeout)
	if statusCode == checkAgain && err == nil {
		statusCode, err = rh.waitForUpdateContentResponse(ctx, w, deviceID, resourceID, notify, rh.onSecondTimeout)
	}
	return statusCode, err
}

func (rh *RequestHandler) UpdateResource(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.updateResourceContent(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot update resource: %w", err), statusCode, w)
	}
}

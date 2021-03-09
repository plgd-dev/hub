package service

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/gorilla/mux"
	"github.com/plgd-dev/cloud/cloud2cloud-connector/events"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/operations"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
)

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
	case pbGRPC.Status_METHOD_NOT_ALLOWED:
		return http.StatusMethodNotAllowed
	case pbGRPC.Status_CREATED:
		return http.StatusCreated
	}
	return http.StatusInternalServerError
}

func sendResponse(w http.ResponseWriter, processed *pbGRPC.UpdateResourceResponse) (int, error) {
	statusCode := statusToHttpStatus(processed.GetStatus())
	if processed.Content != nil {
		content, err := unmarshalContent(processed.GetContent())
		if err != nil {
			logAndWriteErrorResponse(fmt.Errorf("cannot unmarshal content: %w", err), statusCode, w)
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
		case nil:
			w.WriteHeader(statusCode)
			return statusCode, nil
		default:
			err = jsonResponseWriterEncoder(w, content, statusCode)
			if err != nil {
				logAndWriteErrorResponse(fmt.Errorf("cannot write response: %w", err), statusCode, w)
				return statusCode, nil
			}
			return statusCode, nil
		}
	}
	w.WriteHeader(statusCode)
	return statusCode, nil
}

func (rh *RequestHandler) updateResourceContent(w http.ResponseWriter, r *http.Request) (int, error) {
	_, userID, err := parseAuth(r.Header.Get("Authorization"))
	if err != nil {
		return http.StatusUnauthorized, fmt.Errorf("cannot get access token: %w", err)
	}
	correlationUUID, err := uuid.NewV4()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("cannot create correlationID: %w", err)
	}

	contentType := r.Header.Get(events.ContentTypeKey)

	routeVars := mux.Vars(r)
	deviceID := routeVars[deviceIDKey]
	href := routeVars[HrefKey]

	buffer := bytes.NewBuffer(make([]byte, 0, 1024))
	_, err = buffer.ReadFrom(r.Body)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot read body: %w", err)
	}

	updateCommand := &commands.UpdateResourceRequest{
		ResourceId:    commands.NewResourceID(deviceID, href),
		CorrelationId: correlationUUID.String(),
		Content: &commands.Content{
			Data:              buffer.Bytes(),
			ContentType:       contentType,
			CoapContentFormat: -1,
		},
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: r.RemoteAddr,
		},
	}

	operator := operations.New(rh.resourceSubscriber, rh.raClient)
	updatedEvent, err := operator.UpdateResource(kitNetGrpc.CtxWithUserID(r.Context(), userID), updateCommand)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot update resource content: %w", err)
	}
	resp, err := pb.RAResourceUpdatedEventToResponse(updatedEvent)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot update resource content: %w", err)
	}
	return sendResponse(w, resp)
}

func (rh *RequestHandler) UpdateResource(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.updateResourceContent(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot update resource: %w", err), statusCode, w)
	}
}

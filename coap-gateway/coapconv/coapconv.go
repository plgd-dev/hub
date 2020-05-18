package coapconv

import (
	"bytes"
	"fmt"

	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/go-coap/v2/message/codes"
)

func StatusToCoapCode(status pbRA.Status, cmdCode codes.Code) codes.Code {
	switch status {
	case pbRA.Status_OK:
		switch cmdCode {
		case codes.POST:
			return codes.Changed
		case codes.GET:
			return codes.Content
		case codes.PUT:
			return codes.Created
		case codes.DELETE:
			return codes.Deleted
		}
	case pbRA.Status_ACCEPTED:
		return codes.Valid
	case pbRA.Status_BAD_REQUEST:
		return codes.BadRequest
	case pbRA.Status_UNAUTHORIZED:
		return codes.Unauthorized
	case pbRA.Status_FORBIDDEN:
		return codes.Forbidden
	case pbRA.Status_NOT_FOUND:
		return codes.NotFound
	case pbRA.Status_UNAVAILABLE:
		return codes.ServiceUnavailable
	case pbRA.Status_NOT_IMPLEMENTED:
		return codes.NotImplemented
	}
	return codes.BadRequest
}

func CoapCodeToStatus(code codes.Code) pbRA.Status {
	switch code {
	case codes.Changed, codes.Content:
		return pbRA.Status_OK
	case codes.Valid:
		return pbRA.Status_ACCEPTED
	case codes.BadRequest:
		return pbRA.Status_BAD_REQUEST
	case codes.Unauthorized:
		return pbRA.Status_UNAUTHORIZED
	case codes.Forbidden:
		return pbRA.Status_FORBIDDEN
	case codes.NotFound:
		return pbRA.Status_NOT_FOUND
	case codes.ServiceUnavailable:
		return pbRA.Status_UNAVAILABLE
	case codes.NotImplemented:
		return pbRA.Status_NOT_IMPLEMENTED
	default:
		return pbRA.Status_ERROR
	}
}

func MakeMediaType(coapContentFormat int32, contentType string) (message.MediaType, error) {
	if coapContentFormat >= 0 {
		return message.MediaType(coapContentFormat), nil
	}
	switch contentType {
	case message.TextPlain.String():
		return message.TextPlain, nil
	case message.AppJSON.String():
		return message.AppJSON, nil
	case message.AppCBOR.String():
		return message.AppCBOR, nil
	case message.AppOcfCbor.String():
		return message.AppOcfCbor, nil
	default:
		return message.TextPlain, fmt.Errorf("unknown content type coapContentFormat(%v), contentType(%v)", coapContentFormat, contentType)
	}
}

func NewCoapResourceUpdateRequest(client *message.ClientConn, href string, reqContentUpdate *pbRA.ResourceUpdatePending) (message.Message, error) {
	if reqContentUpdate.Content == nil {
		return nil, fmt.Errorf("invalid content for update content")
	}
	mediaType, err := MakeMediaType(reqContentUpdate.Content.CoapContentFormat, reqContentUpdate.Content.ContentType)
	if err != nil {
		return nil, fmt.Errorf("invalid content type for update content: %v", err)
	}
	req, err := client.NewPostRequest(href, mediaType, bytes.NewBuffer(reqContentUpdate.Content.Data))
	if err != nil {
		return nil, fmt.Errorf("cannot create update content request: %v", err)
	}
	if reqContentUpdate.GetResourceInterface() != "" {
		req.AddOption(message.URIQuery, "if="+reqContentUpdate.GetResourceInterface())
	}
	return req, nil
}

func NewCoapResourceRetrieveRequest(client *message.ClientConn, href string, resRetrieve *pbRA.ResourceRetrievePending) (message.Message, error) {
	req, err := client.NewGetRequest(href)
	if err != nil {
		return nil, fmt.Errorf("cannot create retrieve content request: %v", err)
	}
	if resRetrieve.GetResourceInterface() != "" {
		req.AddOption(message.URIQuery, "if="+resRetrieve.GetResourceInterface())
	}
	return req, nil
}

func MakeContent(resp message.Message) pbRA.Content {
	contentTypeString := ""
	coapContentFormat := int32(-1)
	if contentType, ok := resp.Option(message.ContentFormat).(message.MediaType); ok {
		contentTypeString = contentType.String()
		coapContentFormat = int32(contentType)
	}
	return pbRA.Content{
		ContentType:       contentTypeString,
		CoapContentFormat: coapContentFormat,
		Data:              resp.Payload(),
	}
}

func MakeCommandMetadata(req *message.Message) pbCQRS.CommandMetadata {
	return pbCQRS.CommandMetadata{
		Sequence:     req.Sequence,
		ConnectionId: req.Client.RemoteAddr().String(),
	}
}

func MakeConfirmResourceRetrieveRequest(resourceId, correlationId string, authCtx pbCQRS.AuthorizationContext, req *message.Message) pbRA.ConfirmResourceRetrieveRequest {
	content := MakeContent(req.Msg)
	metadata := MakeCommandMetadata(req)

	return pbRA.ConfirmResourceRetrieveRequest{
		AuthorizationContext: &authCtx,
		ResourceId:           resourceId,
		CorrelationId:        correlationId,
		Status:               CoapCodeToStatus(req.Msg.Code()),
		Content:              &content,
		CommandMetadata:      &metadata,
	}
}

func MakeConfirmResourceUpdateRequest(resourceId, correlationId string, authCtx pbCQRS.AuthorizationContext, req *message.Message) pbRA.ConfirmResourceUpdateRequest {
	content := MakeContent(req.Msg)
	metadata := MakeCommandMetadata(req)

	return pbRA.ConfirmResourceUpdateRequest{
		AuthorizationContext: &authCtx,
		ResourceId:           resourceId,
		CorrelationId:        correlationId,
		Status:               CoapCodeToStatus(req.Msg.Code()),
		Content:              &content,
		CommandMetadata:      &metadata,
	}
}

func MakeNotifyResourceChangedRequest(resourceId string, authCtx pbCQRS.AuthorizationContext, req *message.Message) pbRA.NotifyResourceChangedRequest {
	content := MakeContent(req.Msg)
	metadata := MakeCommandMetadata(req)

	return pbRA.NotifyResourceChangedRequest{
		AuthorizationContext: &authCtx,
		ResourceId:           resourceId,
		Content:              &content,
		CommandMetadata:      &metadata,
		Status:               CoapCodeToStatus(req.Msg.Code()),
	}
}

func MakeUpdateResourceRequest(resourceId, correlationId string, authCtx pbCQRS.AuthorizationContext, req *message.Message) pbRA.UpdateResourceRequest {
	content := MakeContent(req.Msg)
	metadata := MakeCommandMetadata(req)

	return pbRA.UpdateResourceRequest{
		AuthorizationContext: &authCtx,
		ResourceId:           resourceId,
		CorrelationId:        correlationId,
		Content:              &content,
		CommandMetadata:      &metadata,
	}
}

func MakeRetrieveResourceRequest(resourceId, resourceInterface, correlationId string, authCtx pbCQRS.AuthorizationContext, req *message.Message) pbRA.RetrieveResourceRequest {
	metadata := MakeCommandMetadata(req)

	return pbRA.RetrieveResourceRequest{
		AuthorizationContext: &authCtx,
		ResourceId:           resourceId,
		CorrelationId:        correlationId,
		ResourceInterface:    resourceInterface,
		CommandMetadata:      &metadata,
	}
}

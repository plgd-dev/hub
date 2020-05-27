package coapconv

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/go-ocf/go-coap/v2/tcp"

	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/mux"
	"github.com/go-ocf/go-coap/v2/tcp/message/pool"
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

func NewCoapResourceUpdateRequest(ctx context.Context, href string, reqContentUpdate *pbRA.ResourceUpdatePending) (*pool.Message, error) {
	mediaType, err := MakeMediaType(reqContentUpdate.Content.CoapContentFormat, reqContentUpdate.Content.ContentType)
	if err != nil {
		return nil, fmt.Errorf("invalid content type for update content: %v", err)
	}
	if reqContentUpdate.Content == nil {
		return nil, fmt.Errorf("invalid content for update content")
	}

	req, err := tcp.NewPostRequest(ctx, href, mediaType, bytes.NewReader(reqContentUpdate.GetContent().GetData()))
	if err != nil {
		return nil, err
	}
	if reqContentUpdate.GetResourceInterface() != "" {
		req.AddOptionString(message.URIQuery, "if="+reqContentUpdate.GetResourceInterface())
	}

	return req, nil
}

func NewCoapResourceRetrieveRequest(ctx context.Context, href string, resRetrieve *pbRA.ResourceRetrievePending) (*pool.Message, error) {
	req, err := tcp.NewGetRequest(ctx, href)
	if err != nil {
		return nil, err
	}
	if resRetrieve.GetResourceInterface() != "" {
		req.AddOptionString(message.URIQuery, "if="+resRetrieve.GetResourceInterface())
	}

	return req, nil
}

func MakeContent(opts message.Options, body io.Reader) pbRA.Content {
	contentTypeString := ""
	coapContentFormat := int32(-1)
	mt, err := opts.ContentFormat()
	if err == nil {
		contentTypeString = mt.String()
		coapContentFormat = int32(mt)
	}
	var data []byte
	if body != nil {
		data, _ = ioutil.ReadAll(body)
	}
	return pbRA.Content{
		ContentType:       contentTypeString,
		CoapContentFormat: coapContentFormat,
		Data:              data,
	}
}

func MakeCommandMetadata(sequenceNumber uint64, connectionID string) pbCQRS.CommandMetadata {
	return pbCQRS.CommandMetadata{
		Sequence:     sequenceNumber,
		ConnectionId: connectionID,
	}
}

func MakeConfirmResourceRetrieveRequest(resourceId, correlationId string, authCtx pbCQRS.AuthorizationContext, connectionID string, req *pool.Message) pbRA.ConfirmResourceRetrieveRequest {
	content := MakeContent(req.Options(), req.Body())
	metadata := MakeCommandMetadata(req.Sequence(), connectionID)

	return pbRA.ConfirmResourceRetrieveRequest{
		AuthorizationContext: &authCtx,
		ResourceId:           resourceId,
		CorrelationId:        correlationId,
		Status:               CoapCodeToStatus(req.Code()),
		Content:              &content,
		CommandMetadata:      &metadata,
	}
}

func MakeConfirmResourceUpdateRequest(resourceId, correlationId string, authCtx pbCQRS.AuthorizationContext, connectionID string, req *pool.Message) pbRA.ConfirmResourceUpdateRequest {
	content := MakeContent(req.Options(), req.Body())
	metadata := MakeCommandMetadata(req.Sequence(), connectionID)

	return pbRA.ConfirmResourceUpdateRequest{
		AuthorizationContext: &authCtx,
		ResourceId:           resourceId,
		CorrelationId:        correlationId,
		Status:               CoapCodeToStatus(req.Code()),
		Content:              &content,
		CommandMetadata:      &metadata,
	}
}

func MakeNotifyResourceChangedRequest(resourceId string, authCtx pbCQRS.AuthorizationContext, connectionID string, req *pool.Message) pbRA.NotifyResourceChangedRequest {
	content := MakeContent(req.Options(), req.Body())
	metadata := MakeCommandMetadata(req.Sequence(), connectionID)

	return pbRA.NotifyResourceChangedRequest{
		AuthorizationContext: &authCtx,
		ResourceId:           resourceId,
		Content:              &content,
		CommandMetadata:      &metadata,
		Status:               CoapCodeToStatus(req.Code()),
	}
}

func MakeUpdateResourceRequest(resourceId, correlationId string, authCtx pbCQRS.AuthorizationContext, connectionID string, req *mux.Message) pbRA.UpdateResourceRequest {
	content := MakeContent(req.Options, req.Body)
	metadata := MakeCommandMetadata(req.SequenceNumber, connectionID)

	return pbRA.UpdateResourceRequest{
		AuthorizationContext: &authCtx,
		ResourceId:           resourceId,
		CorrelationId:        correlationId,
		Content:              &content,
		CommandMetadata:      &metadata,
	}
}

func MakeRetrieveResourceRequest(resourceId, resourceInterface, correlationId string, authCtx pbCQRS.AuthorizationContext, connectionID string, req *mux.Message) pbRA.RetrieveResourceRequest {
	metadata := MakeCommandMetadata(req.SequenceNumber, connectionID)

	return pbRA.RetrieveResourceRequest{
		AuthorizationContext: &authCtx,
		ResourceId:           resourceId,
		CorrelationId:        correlationId,
		ResourceInterface:    resourceInterface,
		CommandMetadata:      &metadata,
	}
}

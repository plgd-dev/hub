package coapconv

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/plgd-dev/go-coap/v2/tcp"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
)

func StatusToCoapCode(status pbGRPC.Status, cmdCode codes.Code) codes.Code {
	switch status {
	case pbGRPC.Status_OK:
		switch cmdCode {
		case codes.POST:
			return codes.Changed
		case codes.GET:
			return codes.Content
		case codes.DELETE:
			return codes.Deleted
		case codes.PUT:
			return codes.Created
		}
	case pbGRPC.Status_CREATED:
		return codes.Created
	case pbGRPC.Status_ACCEPTED:
		return codes.Valid
	case pbGRPC.Status_BAD_REQUEST:
		return codes.BadRequest
	case pbGRPC.Status_UNAUTHORIZED:
		return codes.Unauthorized
	case pbGRPC.Status_FORBIDDEN:
		return codes.Forbidden
	case pbGRPC.Status_NOT_FOUND:
		return codes.NotFound
	case pbGRPC.Status_UNAVAILABLE:
		return codes.ServiceUnavailable
	case pbGRPC.Status_NOT_IMPLEMENTED:
		return codes.NotImplemented
	}
	return codes.BadRequest
}

func CoapCodeToStatus(code codes.Code) commands.Status {
	switch code {
	case codes.Changed, codes.Content, codes.Deleted:
		return commands.Status_OK
	case codes.Valid:
		return commands.Status_ACCEPTED
	case codes.BadRequest:
		return commands.Status_BAD_REQUEST
	case codes.Unauthorized:
		return commands.Status_UNAUTHORIZED
	case codes.Forbidden:
		return commands.Status_FORBIDDEN
	case codes.NotFound:
		return commands.Status_NOT_FOUND
	case codes.ServiceUnavailable:
		return commands.Status_UNAVAILABLE
	case codes.NotImplemented:
		return commands.Status_NOT_IMPLEMENTED
	case codes.MethodNotAllowed:
		return commands.Status_METHOD_NOT_ALLOWED
	case codes.Created:
		return commands.Status_CREATED
	default:
		return commands.Status_ERROR
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

func NewCoapResourceUpdateRequest(ctx context.Context, event *pb.Event_ResourceUpdatePending) (*pool.Message, error) {
	mediaType, err := MakeMediaType(-1, event.GetContent().GetContentType())
	if err != nil {
		return nil, fmt.Errorf("invalid content type for update content: %w", err)
	}
	if event.Content == nil {
		return nil, fmt.Errorf("invalid content for update content")
	}

	req, err := tcp.NewPostRequest(ctx, event.GetResourceId().GetHref(), mediaType, bytes.NewReader(event.GetContent().GetData()))
	if err != nil {
		return nil, err
	}
	if event.GetResourceInterface() != "" {
		req.AddOptionString(message.URIQuery, "if="+event.GetResourceInterface())
	}

	return req, nil
}

func NewCoapResourceRetrieveRequest(ctx context.Context, event *pb.Event_ResourceRetrievePending) (*pool.Message, error) {
	req, err := tcp.NewGetRequest(ctx, event.GetResourceId().GetHref())
	if err != nil {
		return nil, err
	}
	if event.GetResourceInterface() != "" {
		req.AddOptionString(message.URIQuery, "if="+event.GetResourceInterface())
	}

	return req, nil
}

func NewCoapResourceDeleteRequest(ctx context.Context, event *pb.Event_ResourceDeletePending) (*pool.Message, error) {
	req, err := tcp.NewDeleteRequest(ctx, event.GetResourceId().GetHref())
	if err != nil {
		return nil, err
	}

	return req, nil
}

func NewContent(opts message.Options, body io.Reader) *commands.Content {
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
	return &commands.Content{
		ContentType:       contentTypeString,
		CoapContentFormat: coapContentFormat,
		Data:              data,
	}
}

func NewCommandMetadata(sequenceNumber uint64, connectionID string) *commands.CommandMetadata {
	return &commands.CommandMetadata{
		Sequence:     sequenceNumber,
		ConnectionId: connectionID,
	}
}

func NewConfirmResourceRetrieveRequest(resourceID *commands.ResourceId, correlationId string, connectionID string, req *pool.Message) *commands.ConfirmResourceRetrieveRequest {
	content := NewContent(req.Options(), req.Body())
	metadata := NewCommandMetadata(req.Sequence(), connectionID)

	return &commands.ConfirmResourceRetrieveRequest{
		ResourceId:      resourceID,
		CorrelationId:   correlationId,
		Status:          CoapCodeToStatus(req.Code()),
		Content:         content,
		CommandMetadata: metadata,
	}
}

func NewConfirmResourceUpdateRequest(resourceID *commands.ResourceId, correlationId string, connectionID string, req *pool.Message) *commands.ConfirmResourceUpdateRequest {
	content := NewContent(req.Options(), req.Body())
	metadata := NewCommandMetadata(req.Sequence(), connectionID)

	return &commands.ConfirmResourceUpdateRequest{
		ResourceId:      resourceID,
		CorrelationId:   correlationId,
		Status:          CoapCodeToStatus(req.Code()),
		Content:         content,
		CommandMetadata: metadata,
	}
}

func NewDeleteResourceRequest(resourceID *commands.ResourceId, req *mux.Message, connectionID string) (*commands.DeleteResourceRequest, error) {
	correlationUUID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("cannot create correlationID: %w", err)
	}
	metadata := NewCommandMetadata(req.SequenceNumber, connectionID)
	return &commands.DeleteResourceRequest{
		ResourceId:      resourceID,
		CorrelationId:   correlationUUID.String(),
		CommandMetadata: metadata,
	}, nil
}

func NewConfirmResourceDeleteRequest(resourceID *commands.ResourceId, correlationId string, connectionID string, req *pool.Message) *commands.ConfirmResourceDeleteRequest {
	content := NewContent(req.Options(), req.Body())
	metadata := NewCommandMetadata(req.Sequence(), connectionID)

	return &commands.ConfirmResourceDeleteRequest{
		ResourceId:      resourceID,
		CorrelationId:   correlationId,
		Status:          CoapCodeToStatus(req.Code()),
		Content:         content,
		CommandMetadata: metadata,
	}
}

func NewNotifyResourceChangedRequest(resourceID *commands.ResourceId, connectionID string, req *pool.Message) *commands.NotifyResourceChangedRequest {
	content := NewContent(req.Options(), req.Body())
	metadata := NewCommandMetadata(req.Sequence(), connectionID)

	return &commands.NotifyResourceChangedRequest{
		ResourceId:      resourceID,
		Content:         content,
		CommandMetadata: metadata,
		Status:          CoapCodeToStatus(req.Code()),
	}
}

func NewUpdateResourceRequest(resourceID *commands.ResourceId, req *mux.Message, connectionID string) (*commands.UpdateResourceRequest, error) {
	correlationUUID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("cannot create correlationID: %w", err)
	}

	content := NewContent(req.Options, req.Body)
	metadata := NewCommandMetadata(req.SequenceNumber, connectionID)
	var resourceInterface string
	qs, err := req.Options.Queries()
	if err == nil {
		for _, q := range qs {
			if strings.HasPrefix(q, "if=") {
				resourceInterface = strings.TrimPrefix(q, "if=")
				break
			}
		}
	}

	return &commands.UpdateResourceRequest{
		ResourceId: resourceID,
		Content: &commands.Content{
			Data:        content.Data,
			ContentType: content.ContentType,
		},
		ResourceInterface: resourceInterface,
		CommandMetadata:   metadata,
		CorrelationId:     correlationUUID.String(),
	}, nil
}

func NewRetrieveResourceRequest(resourceID *commands.ResourceId, req *mux.Message, connectionID string) (*commands.RetrieveResourceRequest, error) {
	correlationUUID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("cannot create correlationID: %w", err)
	}
	metadata := NewCommandMetadata(req.SequenceNumber, connectionID)
	var resourceInterface string
	qs, err := req.Options.Queries()
	if err == nil {
		for _, q := range qs {
			if strings.HasPrefix(q, "if=") {
				resourceInterface = strings.TrimPrefix(q, "if=")
				break
			}
		}
	}
	return &commands.RetrieveResourceRequest{
		ResourceId:        resourceID,
		CorrelationId:     correlationUUID.String(),
		ResourceInterface: resourceInterface,
		CommandMetadata:   metadata,
	}, nil
}

func NewCreateResourceRequest(resourceID *commands.ResourceId, req *mux.Message, connectionID string) (*commands.CreateResourceRequest, error) {
	correlationUUID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("cannot create correlationID: %w", err)
	}

	content := NewContent(req.Options, req.Body)
	metadata := NewCommandMetadata(req.SequenceNumber, connectionID)

	return &commands.CreateResourceRequest{
		ResourceId:      resourceID,
		CorrelationId:   correlationUUID.String(),
		Content:         content,
		CommandMetadata: metadata,
	}, nil
}

func NewConfirmResourceCreateRequest(resourceID *commands.ResourceId, correlationId string, connectionID string, req *pool.Message) *commands.ConfirmResourceCreateRequest {
	content := NewContent(req.Options(), req.Body())
	metadata := NewCommandMetadata(req.Sequence(), connectionID)

	return &commands.ConfirmResourceCreateRequest{
		ResourceId:      resourceID,
		CorrelationId:   correlationId,
		Status:          CoapCodeToStatus(req.Code()),
		Content:         content,
		CommandMetadata: metadata,
	}
}

const OCFCreateInterface = "oic.if.create"

func NewCoapResourceCreateRequest(ctx context.Context, event *pb.Event_ResourceCreatePending) (*pool.Message, error) {
	mediaType, err := MakeMediaType(-1, event.GetContent().GetContentType())
	if err != nil {
		return nil, fmt.Errorf("invalid content type for create content: %w", err)
	}
	if event.Content == nil {
		return nil, fmt.Errorf("invalid content for create content")
	}

	req, err := tcp.NewPostRequest(ctx, event.GetResourceId().GetHref(), mediaType, bytes.NewReader(event.GetContent().GetData()))
	if err != nil {
		return nil, err
	}
	req.AddOptionString(message.URIQuery, "if="+OCFCreateInterface)

	return req, nil
}

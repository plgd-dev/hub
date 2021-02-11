package coapconv

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/plgd-dev/go-coap/v2/tcp"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	pbCQRS "github.com/plgd-dev/cloud/resource-aggregate/pb"
	pbRA "github.com/plgd-dev/cloud/resource-aggregate/pb"
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
		case codes.PUT:
			return codes.Created
		case codes.DELETE:
			return codes.Deleted
		}
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

func CoapCodeToStatus(code codes.Code) pbRA.Status {
	switch code {
	case codes.Changed, codes.Content, codes.Deleted:
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
	case codes.MethodNotAllowed:
		return pbRA.Status_METHOD_NOT_ALLOWED
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

func MakeConfirmResourceRetrieveRequest(deviceID, href, correlationId string, authCtx *pbCQRS.AuthorizationContext, connectionID string, req *pool.Message) pbRA.ConfirmResourceRetrieveRequest {
	content := MakeContent(req.Options(), req.Body())
	metadata := MakeCommandMetadata(req.Sequence(), connectionID)

	return pbRA.ConfirmResourceRetrieveRequest{
		AuthorizationContext: authCtx,
		ResourceId: &pbRA.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		CorrelationId:   correlationId,
		Status:          CoapCodeToStatus(req.Code()),
		Content:         &content,
		CommandMetadata: &metadata,
	}
}

func MakeConfirmResourceUpdateRequest(deviceID, href, correlationId string, authCtx *pbCQRS.AuthorizationContext, connectionID string, req *pool.Message) pbRA.ConfirmResourceUpdateRequest {
	content := MakeContent(req.Options(), req.Body())
	metadata := MakeCommandMetadata(req.Sequence(), connectionID)

	return pbRA.ConfirmResourceUpdateRequest{
		AuthorizationContext: authCtx,
		ResourceId: &pbRA.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		CorrelationId:   correlationId,
		Status:          CoapCodeToStatus(req.Code()),
		Content:         &content,
		CommandMetadata: &metadata,
	}
}

func MakeConfirmResourceDeleteRequest(deviceID, href string, correlationId string, authCtx *pbCQRS.AuthorizationContext, connectionID string, req *pool.Message) pbRA.ConfirmResourceDeleteRequest {
	content := MakeContent(req.Options(), req.Body())
	metadata := MakeCommandMetadata(req.Sequence(), connectionID)

	return pbRA.ConfirmResourceDeleteRequest{
		AuthorizationContext: authCtx,
		ResourceId: &pbRA.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		CorrelationId:   correlationId,
		Status:          CoapCodeToStatus(req.Code()),
		Content:         &content,
		CommandMetadata: &metadata,
	}
}

func MakeNotifyResourceChangedRequest(deviceID, href string, authCtx *pbCQRS.AuthorizationContext, connectionID string, req *pool.Message) pbRA.NotifyResourceChangedRequest {
	content := MakeContent(req.Options(), req.Body())
	metadata := MakeCommandMetadata(req.Sequence(), connectionID)

	return pbRA.NotifyResourceChangedRequest{
		AuthorizationContext: authCtx,
		ResourceId: &pbRA.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		Content:         &content,
		CommandMetadata: &metadata,
		Status:          CoapCodeToStatus(req.Code()),
	}
}

func MakeUpdateResourceRequest(deviceID, href string, req *mux.Message) *pbGRPC.UpdateResourceValuesRequest {
	content := MakeContent(req.Options, req.Body)
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

	return &pbGRPC.UpdateResourceValuesRequest{
		ResourceId: &pbGRPC.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		Content: &pbGRPC.Content{
			Data:        content.Data,
			ContentType: content.ContentType,
		},
		ResourceInterface: resourceInterface,
	}
}

func MakeRetrieveResourceRequest(deviceID, href string, resourceInterface, correlationId string, authCtx pbCQRS.AuthorizationContext, connectionID string, req *mux.Message) pbRA.RetrieveResourceRequest {
	metadata := MakeCommandMetadata(req.SequenceNumber, connectionID)

	return pbRA.RetrieveResourceRequest{
		AuthorizationContext: &authCtx,
		ResourceId: &pbRA.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		CorrelationId:     correlationId,
		ResourceInterface: resourceInterface,
		CommandMetadata:   &metadata,
	}
}

func MakeConfirmResourceCreateRequest(deviceID, href string, correlationId string, authCtx *commands.AuthorizationContext, connectionID string, req *pool.Message) commands.ConfirmResourceCreateRequest {
	content := MakeContent(req.Options(), req.Body())
	metadata := MakeCommandMetadata(req.Sequence(), connectionID)

	return commands.ConfirmResourceCreateRequest{
		AuthorizationContext: authCtx,
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		CorrelationId:   correlationId,
		Status:          CoapCodeToStatus(req.Code()),
		Content:         &content,
		CommandMetadata: &metadata,
	}
}

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
	req.AddOptionString(message.URIQuery, "if=oic.if.create")

	return req, nil
}

package coapconv

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	"github.com/plgd-dev/device/v2/pkg/codec/json"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/hub/v2/coap-gateway/uri"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
)

func StatusToCoapCode(status commands.Status, operation Operation) codes.Code {
	switch status {
	case commands.Status_OK:
		switch operation {
		case Update:
			return codes.Changed
		case Retrieve:
			return codes.Content
		case Delete:
			return codes.Deleted
		case Create:
			return codes.Created
		}
	case commands.Status_CREATED:
		return codes.Created
	case commands.Status_NOT_MODIFIED:
		return codes.Valid
	case commands.Status_ACCEPTED:
		return codes.Valid
	case commands.Status_BAD_REQUEST:
		return codes.BadRequest
	case commands.Status_UNAUTHORIZED:
		return codes.Unauthorized
	case commands.Status_FORBIDDEN:
		return codes.Forbidden
	case commands.Status_NOT_FOUND:
		return codes.NotFound
	case commands.Status_UNAVAILABLE:
		return codes.ServiceUnavailable
	case commands.Status_NOT_IMPLEMENTED:
		return codes.NotImplemented
	}
	return codes.BadRequest
}

func CoapCodeToStatus(code codes.Code, operation Operation) commands.Status {
	switch code {
	case codes.Changed, codes.Content, codes.Deleted:
		return commands.Status_OK
	case codes.Valid:
		if operation == Retrieve {
			return commands.Status_NOT_MODIFIED
		}
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
		return message.MediaType(coapContentFormat), nil //nolint:gosec
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

func newCoapResourceCreateOrUpdateRequest(ctx context.Context, messagePool *pool.Pool, resourceID *commands.ResourceId, content *commands.Content, resourceInterface string) (*pool.Message, error) {
	mediaType, err := MakeMediaType(-1, content.GetContentType())
	if err != nil {
		return nil, fmt.Errorf("invalid content type for request: %w", err)
	}
	if content == nil {
		return nil, errors.New("invalid content for request")
	}
	token, err := message.GetToken()
	if err != nil {
		return nil, err
	}
	req := messagePool.AcquireMessage(ctx)
	err = req.SetupPost(resourceID.GetHref(), token, mediaType,
		bytes.NewReader(content.GetData()))
	if err != nil {
		return nil, err
	}
	if resourceInterface != "" {
		req.AddOptionString(message.URIQuery, uri.InterfaceQueryKeyPrefix+resourceInterface)
	}
	return req, nil
}

func NewCoapResourceUpdateRequest(ctx context.Context, messagePool *pool.Pool, event *events.ResourceUpdatePending) (*pool.Message, error) {
	return newCoapResourceCreateOrUpdateRequest(ctx, messagePool, event.GetResourceId(), event.GetContent(), event.GetResourceInterface())
}

func NewCoapResourceRetrieveRequest(ctx context.Context, messagePool *pool.Pool, event *events.ResourceRetrievePending) (*pool.Message, error) {
	token, err := message.GetToken()
	if err != nil {
		return nil, err
	}
	req := messagePool.AcquireMessage(ctx)
	err = req.SetupGet(event.GetResourceId().GetHref(), token)
	if err != nil {
		return nil, err
	}
	if event.GetResourceInterface() != "" {
		req.AddOptionString(message.URIQuery, uri.InterfaceQueryKeyPrefix+event.GetResourceInterface())
	}
	for _, etag := range event.GetEtag() {
		if err := req.AddETag(etag); err != nil {
			return nil, err
		}
	}
	return req, nil
}

func NewCoapResourceDeleteRequest(ctx context.Context, messagePool *pool.Pool, event *events.ResourceDeletePending) (*pool.Message, error) {
	token, err := message.GetToken()
	if err != nil {
		return nil, err
	}
	req := messagePool.AcquireMessage(ctx)
	err = req.SetupDelete(event.GetResourceId().GetHref(), token)
	if err != nil {
		return nil, err
	}
	if event.GetResourceInterface() != "" {
		req.AddOptionString(message.URIQuery, uri.InterfaceQueryKeyPrefix+event.GetResourceInterface())
	}
	return req, nil
}

func NewContent(opts message.Options, body io.Reader) *commands.Content {
	data, coapContentFormat := GetContentData(opts, body)

	return &commands.Content{
		ContentType:       GetContentFormatString(coapContentFormat),
		CoapContentFormat: coapContentFormat,
		Data:              data,
	}
}

func GetContentData(opts message.Options, body io.Reader) (data []byte, contentFormat int32) {
	contentFormat = int32(-1)
	mt, err := opts.ContentFormat()
	if err == nil {
		contentFormat = int32(mt)
	}
	if body != nil {
		data, _ = io.ReadAll(body)
	}
	return data, contentFormat
}

func GetContentFormatString(coapContentFormat int32) string {
	if coapContentFormat != -1 {
		return message.MediaType(coapContentFormat).String() //nolint:gosec
	}
	return ""
}

func NewCommandMetadata(sequenceNumber uint64, connectionID string) *commands.CommandMetadata {
	return &commands.CommandMetadata{
		Sequence:     sequenceNumber,
		ConnectionId: connectionID,
	}
}

func getETagFromMessage(msg interface{ ETag() ([]byte, error) }) []byte {
	etag, err := msg.ETag()
	if err != nil {
		etag = nil
	}
	return etag
}

func getETagsFromMessage(msg interface{ ETags(b [][]byte) (int, error) }) [][]byte {
	if msg == nil {
		return nil
	}
	etags := make([][]byte, 32)
	for {
		n, err := msg.ETags(etags)
		if errors.Is(err, message.ErrTooSmall) {
			etags = make([][]byte, len(etags)*2)
			continue
		}
		if err != nil {
			return nil
		}
		etags = etags[:n]
		break
	}
	return etags
}

func NewConfirmResourceRetrieveRequest(resourceID *commands.ResourceId, correlationID, connectionID string, req *pool.Message) *commands.ConfirmResourceRetrieveRequest {
	content := NewContent(req.Options(), req.Body())
	metadata := NewCommandMetadata(req.Sequence(), connectionID)

	return &commands.ConfirmResourceRetrieveRequest{
		ResourceId:      resourceID,
		CorrelationId:   correlationID,
		Status:          CoapCodeToStatus(req.Code(), Retrieve),
		Content:         content,
		CommandMetadata: metadata,
		Etag:            getETagFromMessage(req),
	}
}

func NewConfirmResourceUpdateRequest(resourceID *commands.ResourceId, correlationID, connectionID string, req *pool.Message) *commands.ConfirmResourceUpdateRequest {
	content := NewContent(req.Options(), req.Body())
	metadata := NewCommandMetadata(req.Sequence(), connectionID)

	return &commands.ConfirmResourceUpdateRequest{
		ResourceId:      resourceID,
		CorrelationId:   correlationID,
		Status:          CoapCodeToStatus(req.Code(), Update),
		Content:         content,
		CommandMetadata: metadata,
	}
}

func createCorrelationID() (string, error) {
	correlationUUID, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("cannot create correlationID: %w", err)
	}
	return correlationUUID.String(), nil
}

func getResourceInterface(req *mux.Message) string {
	qs, err := req.Options().Queries()
	if err == nil {
		for _, q := range qs {
			if strings.HasPrefix(q, uri.InterfaceQueryKeyPrefix) {
				return strings.TrimPrefix(q, uri.InterfaceQueryKeyPrefix)
			}
		}
	}
	return ""
}

func NewDeleteResourceRequest(resourceID *commands.ResourceId, req *mux.Message, connectionID string) (*commands.DeleteResourceRequest, error) {
	correlationID, err := createCorrelationID()
	if err != nil {
		return nil, err
	}
	resourceInterface := getResourceInterface(req)

	metadata := NewCommandMetadata(req.Sequence(), connectionID)
	return &commands.DeleteResourceRequest{
		ResourceId:        resourceID,
		CorrelationId:     correlationID,
		CommandMetadata:   metadata,
		ResourceInterface: resourceInterface,
	}, nil
}

func NewConfirmResourceDeleteRequest(resourceID *commands.ResourceId, correlationID, connectionID string, req *pool.Message) *commands.ConfirmResourceDeleteRequest {
	content := NewContent(req.Options(), req.Body())
	metadata := NewCommandMetadata(req.Sequence(), connectionID)

	return &commands.ConfirmResourceDeleteRequest{
		ResourceId:      resourceID,
		CorrelationId:   correlationID,
		Status:          CoapCodeToStatus(req.Code(), Delete),
		Content:         content,
		CommandMetadata: metadata,
	}
}

func NewNotifyResourceChangedRequest(resourceID *commands.ResourceId, resourceTypes []string, connectionID string, req *pool.Message) *commands.NotifyResourceChangedRequest {
	content := NewContent(req.Options(), req.Body())
	metadata := NewCommandMetadata(req.Sequence(), connectionID)

	rtFromBody := tryToGetResourceTypesFromContent(content.GetCoapContentFormat(), content.GetData())
	if len(rtFromBody) > 0 {
		resourceTypes = rtFromBody
	}

	return &commands.NotifyResourceChangedRequest{
		ResourceId:      resourceID,
		Content:         content,
		CommandMetadata: metadata,
		ResourceTypes:   resourceTypes,
		Status:          CoapCodeToStatus(req.Code(), Update),
		Etag:            getETagFromMessage(req),
	}
}

var filterOutEmptyResources = []string{
	"/oc/wk/introspection",
	"/oic/sec/",
}

// inaccessible oic/sec resources have empty content and should be skipped
func filterOutEmptyResource(resource resources.BatchRepresentation) (isEmpty bool, filterOut bool) {
	if len(resource.Content) == 2 {
		v := make(map[interface{}]interface{}, 128)
		if err := cbor.Decode(resource.Content, &v); err == nil && len(v) == 0 {
			isEmpty = true
			for _, f := range filterOutEmptyResources {
				if strings.HasPrefix(resource.Href(), f) {
					return isEmpty, true
				}
			}
		}
	}
	return isEmpty, false
}

type ct struct {
	ResourceTypes []string `json:"rt"`
}

func tryToGetResourceTypesFromContent(contentFormat int32, content []byte) []string {
	if len(content) == 0 {
		return nil
	}
	decode := func([]byte, interface{}) error {
		return errors.New("unsupported")
	}
	switch contentFormat {
	case int32(message.AppOcfCbor), int32(message.AppCBOR):
		decode = cbor.Decode
	case int32(message.AppJSON):
		decode = json.Decode
	}
	var c ct
	if err := decode(content, &c); err == nil {
		return c.ResourceTypes
	}
	return nil
}

func NewNotifyResourceChangedRequestsFromBatchResourceDiscovery(deviceID, connectionID string, req *pool.Message) ([]*commands.NotifyResourceChangedRequest, error) {
	data, contentFormat := GetContentData(req.Options(), req.Body())
	metadata := NewCommandMetadata(req.Sequence(), connectionID)

	discoveryError := func(err error) error {
		return fmt.Errorf("failed to parse discovery resource: %w", err)
	}
	var rs resources.BatchResourceDiscovery
	switch contentFormat {
	case int32(message.AppOcfCbor), int32(message.AppCBOR):
		if err := cbor.Decode(data, &rs); err != nil {
			return nil, discoveryError(err)
		}
	default:
		return nil, discoveryError(fmt.Errorf("invalid format(%v)", contentFormat))
	}

	requests := make([]*commands.NotifyResourceChangedRequest, 0, len(rs))
	etag := getETagFromMessage(req)
	var latestETagResource *commands.NotifyResourceChangedRequest
	for _, r := range rs {
		isEmpty, filterOut := filterOutEmptyResource(r)
		if filterOut {
			continue
		}
		ct := contentFormat
		data := r.Content
		code := CoapCodeToStatus(req.Code(), Retrieve)
		if isEmpty {
			// if we gets empty content we consider it as not found. Empty message is send when resource is deleted/acls don't allows as to access.
			ct = -1
			data = nil
			code = commands.Status_NOT_FOUND
		}
		resourceTypes := r.ResourceTypes
		if len(resourceTypes) == 0 {
			resourceTypes = tryToGetResourceTypesFromContent(contentFormat, r.Content)
		}
		resourceChangedReq := &commands.NotifyResourceChangedRequest{
			ResourceId: commands.NewResourceID(deviceID, r.Href()),
			Content: &commands.Content{
				ContentType:       GetContentFormatString(ct),
				CoapContentFormat: ct,
				Data:              data,
			},
			CommandMetadata: metadata,
			Status:          code,
			Etag:            r.ETag,
			ResourceTypes:   resourceTypes,
		}
		if len(etag) > 0 && bytes.Equal(etag, r.ETag) {
			latestETagResource = resourceChangedReq
			continue
		}
		requests = append(requests, resourceChangedReq)
	}
	// send latestETagResource need to be send as last because the resource are applied in order in resource aggregate
	// so latestETagResource is the last resource in the batch
	if latestETagResource != nil {
		requests = append(requests, latestETagResource)
	}
	return requests, nil
}

func NewUpdateResourceRequest(resourceID *commands.ResourceId, req *mux.Message, connectionID string) (*commands.UpdateResourceRequest, error) {
	correlationID, err := createCorrelationID()
	if err != nil {
		return nil, err
	}

	content := NewContent(req.Options(), req.Body())
	metadata := NewCommandMetadata(req.Sequence(), connectionID)
	resourceInterface := getResourceInterface(req)

	return &commands.UpdateResourceRequest{
		ResourceId: resourceID,
		Content: &commands.Content{
			Data:        content.GetData(),
			ContentType: content.GetContentType(),
		},
		ResourceInterface: resourceInterface,
		CommandMetadata:   metadata,
		CorrelationId:     correlationID,
	}, nil
}

func NewRetrieveResourceRequest(resourceID *commands.ResourceId, req *mux.Message, connectionID string) (*commands.RetrieveResourceRequest, error) {
	correlationID, err := createCorrelationID()
	if err != nil {
		return nil, err
	}
	metadata := NewCommandMetadata(req.Sequence(), connectionID)
	resourceInterface := getResourceInterface(req)
	return &commands.RetrieveResourceRequest{
		ResourceId:        resourceID,
		CorrelationId:     correlationID,
		ResourceInterface: resourceInterface,
		CommandMetadata:   metadata,
		Etag:              getETagsFromMessage(req),
	}, nil
}

func NewCreateResourceRequest(resourceID *commands.ResourceId, req *mux.Message, connectionID string) (*commands.CreateResourceRequest, error) {
	correlationID, err := createCorrelationID()
	if err != nil {
		return nil, err
	}

	content := NewContent(req.Options(), req.Body())
	metadata := NewCommandMetadata(req.Sequence(), connectionID)

	return &commands.CreateResourceRequest{
		ResourceId:      resourceID,
		CorrelationId:   correlationID,
		Content:         content,
		CommandMetadata: metadata,
	}, nil
}

func NewConfirmResourceCreateRequest(resourceID *commands.ResourceId, correlationID, connectionID string, req *pool.Message) *commands.ConfirmResourceCreateRequest {
	content := NewContent(req.Options(), req.Body())
	metadata := NewCommandMetadata(req.Sequence(), connectionID)

	return &commands.ConfirmResourceCreateRequest{
		ResourceId:      resourceID,
		CorrelationId:   correlationID,
		Status:          CoapCodeToStatus(req.Code(), Create),
		Content:         content,
		CommandMetadata: metadata,
	}
}

func NewCoapResourceCreateRequest(ctx context.Context, messagePool *pool.Pool, event *events.ResourceCreatePending) (*pool.Message, error) {
	return newCoapResourceCreateOrUpdateRequest(ctx, messagePool, event.GetResourceId(), event.GetContent(), interfaces.OC_IF_CREATE)
}

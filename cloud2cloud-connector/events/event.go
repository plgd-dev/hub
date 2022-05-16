package events

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/plgd-dev/kit/v2/codec/json"
)

// https://github.com/openconnectivityfoundation/cloud-services/blob/master/swagger2.0/oic.r.cloudapiforcloudservices.swagger.json
const (
	CorrelationIDKey  = "Correlation-ID"
	SubscriptionIDKey = "Subscription-ID"
	ContentTypeKey    = "Content-Type"
	AcceptKey         = "Accept"
	EventTypeKey      = "Event-Type"
	SequenceNumberKey = "Sequence-Number"
	EventTimestampKey = "Event-Timestamp"
	EventSignatureKey = "Event-Signature"
	AcceptEncodingKey = "Accept-Encoding"
)
const ContentEncodingKey = "Content-Encoding"

var ContentType_JSON = message.AppJSON.String()
var ContentType_VNDOCFCBOR = message.AppOcfCbor.String()

type EventHeader struct {
	CorrelationID   string
	ID              string
	ContentType     string
	EventType       EventType
	SequenceNumber  uint64
	EventTimestamp  time.Time
	EventSignature  string
	AcceptEncoding  []string
	ContentEncoding string
}

func invalidKey(key string) string {
	return "invalid " + key
}

func invalidKeyError(key string) error {
	return fmt.Errorf(invalidKey(key))
}

func invalidKeyValueError(key string, value interface{}, err error) error {
	if err == nil {
		return fmt.Errorf(invalidKey(key)+"(%v)", value)
	}
	return fmt.Errorf(invalidKey(key)+"(%v): %w", value, err)
}

func ParseEventHeader(r *http.Request) (h EventHeader, _ error) {
	correlationID := r.Header.Get(CorrelationIDKey)
	subscriptionID := r.Header.Get(SubscriptionIDKey)
	if subscriptionID == "" {
		return h, invalidKeyError(SubscriptionIDKey)
	}
	eventType := EventType(r.Header.Get(EventTypeKey))
	switch eventType {
	case EventType_ResourceChanged,
		EventType_ResourcesPublished, EventType_ResourcesUnpublished,
		EventType_DevicesOnline, EventType_DevicesOffline, EventType_DevicesRegistered, EventType_DevicesUnregistered,
		EventType_SubscriptionCanceled:
	default:
		return h, invalidKeyValueError(EventTypeKey, eventType, nil)
	}

	contentType := r.Header.Get(ContentTypeKey)
	switch contentType {
	case "":
		switch eventType {
		case EventType_SubscriptionCanceled:
		default:
			return h, invalidKeyError(ContentTypeKey)
		}
	case ContentType_JSON:
	case ContentType_VNDOCFCBOR:
	default:
		return h, invalidKeyValueError(ContentTypeKey, contentType, nil)
	}

	seqNum := r.Header.Get(SequenceNumberKey)
	if seqNum == "" {
		return h, invalidKeyError(SequenceNumberKey)
	}
	sequenceNumber, err := strconv.ParseUint(seqNum, 10, 64)
	if err != nil {
		return h, invalidKeyValueError(SequenceNumberKey, seqNum, err)
	}

	evTimestamp := r.Header.Get(EventTimestampKey)
	if evTimestamp == "" {
		return h, invalidKeyError(EventTimestampKey)
	}
	eventTimestamp, err := strconv.ParseInt(evTimestamp, 10, 64)
	if err != nil {
		return h, invalidKeyValueError(EventTimestampKey, evTimestamp, err)
	}
	eventSignature := r.Header.Get(EventSignatureKey)
	if eventSignature == "" {
		return h, invalidKeyError(EventSignatureKey)
	}

	contentEncoding := r.Header.Get(ContentEncodingKey)

	var acceptEncoding []string
	v := r.Header.Get(AcceptEncodingKey)
	if r.Method == "POST" && v != "" {
		acceptEncoding = strings.Split(v, ",")
		if len(acceptEncoding) != 1 {
			return h, invalidKeyValueError(AcceptEncodingKey, acceptEncoding, fmt.Errorf("more than 1"))
		}
	}

	return EventHeader{
		CorrelationID:   correlationID,
		ID:              subscriptionID,
		ContentType:     contentType,
		EventType:       eventType,
		SequenceNumber:  sequenceNumber,
		EventTimestamp:  time.Unix(eventTimestamp, 0),
		EventSignature:  eventSignature,
		ContentEncoding: contentEncoding,
		AcceptEncoding:  acceptEncoding,
	}, nil
}

func (h EventHeader) GetContentDecoder() (func(w []byte, v interface{}) error, error) {
	var decoder func(w []byte, v interface{}) error
	switch h.ContentType {
	case ContentType_JSON:
		decoder = json.Decode
	case ContentType_VNDOCFCBOR:
		decoder = cbor.Decode
	}
	if decoder == nil {
		return nil, fmt.Errorf("%v decoder not found", h.ContentType)
	}

	return decoder, nil
}

func CalculateEventSignature(secret, contentType string, eventType EventType, subscriptionID string, seqNum uint64, timeStamp time.Time, body []byte) string {
	hash := hmac.New(sha256.New, []byte(secret))
	hash.Write([]byte(contentType))
	hash.Write([]byte(":"))
	hash.Write([]byte(eventType))
	hash.Write([]byte(":"))
	hash.Write([]byte(subscriptionID))
	hash.Write([]byte(":"))
	hash.Write([]byte(strconv.FormatUint(seqNum, 10)))
	hash.Write([]byte(":"))
	hash.Write([]byte(strconv.FormatInt(timeStamp.Unix(), 10)))
	hash.Write([]byte(":"))
	hash.Write(body)
	return hex.EncodeToString(hash.Sum(nil))
}

package otelcoap

import (
	"context"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	tcpCoder "github.com/plgd-dev/go-coap/v3/tcp/coder"
	udpCoder "github.com/plgd-dev/go-coap/v3/udp/coder"
	pkgMessage "github.com/plgd-dev/hub/v2/coap-gateway/service/message"
	"github.com/plgd-dev/hub/v2/pkg/opentelemetry"
	"github.com/plgd-dev/kit/v2/codec/json"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	semconv1_30 "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	messageSent       = MessageType(semconv1_30.RPCMessageTypeSent)
	messageReceived   = MessageType(semconv1_30.RPCMessageTypeReceived)
	COAPStatusCodeKey = attribute.Key("coap.status_code")
	COAPMethodKey     = attribute.Key("coap.method")
	COAPPathKey       = attribute.Key("coap.path")
	COAPRequest       = attribute.Key("coap.request")
)

type MessageType attribute.KeyValue

type Message struct {
	Size int
}

func MakeMessage(msg *pool.Message) Message {
	tcpMsg := message.Message{
		Code:    msg.Code(),
		Token:   msg.Token(),
		Options: msg.Options(),
	}
	var coder interface {
		Size(message.Message) (int, error)
	}
	if msg.Type() == message.Unset && msg.MessageID() < 0 {
		coder = tcpCoder.DefaultCoder
	} else {
		coder = udpCoder.DefaultCoder
	}
	size, err := coder.Size(tcpMsg)
	if err != nil {
		size = 0
	}

	return Message{
		Size: size,
	}
}

// Event adds an event of the messageType to the span associated with the
// passed context with id and size (if message is a proto message).
func (m MessageType) Event(ctx context.Context, msg Message) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent("message", trace.WithAttributes(
		attribute.KeyValue(m),
		semconv.MessageUncompressedSizeKey.Int(msg.Size),
	))
}

func SetRequest(ctx context.Context, message *pool.Message) {
	span := trace.SpanFromContext(ctx)
	msg := pkgMessage.ToJson(message, true, false)
	if msg.Body != nil {
		request := ""
		if body, ok := msg.Body.(string); ok {
			request = body
		} else {
			v, err := json.Encode(msg.Body)
			if err == nil {
				request = string(v)
			}
		}
		span.SetAttributes(COAPRequest.String(request))
	}
}

func DefaultTransportFormatter(path string) string {
	return "COAP " + path
}

func StatusCodeAttr(c codes.Code) attribute.KeyValue {
	return COAPStatusCodeKey.Int64(int64(c))
}

func MessageReceivedEvent(ctx context.Context, message Message) {
	messageReceived.Event(ctx, message)
}

func MessageSentEvent(ctx context.Context, message Message) {
	messageSent.Event(ctx, message)
}

func Start(ctx context.Context, path, method string, opts ...Option) (context.Context, trace.Span) {
	cfg := newConfig(opts...)

	tracer := cfg.TracerProvider.Tracer(
		InstrumentationName,
		trace.WithInstrumentationVersion(opentelemetry.SemVersion()),
	)

	attrs := []attribute.KeyValue{
		COAPMethodKey.String(method),
		COAPPathKey.String(path),
	}
	spanOpts := []trace.SpanStartOption{trace.WithAttributes(attrs...)}
	if len(cfg.SpanStartOptions) > 0 {
		spanOpts = append(spanOpts, cfg.SpanStartOptions...)
	}

	return tracer.Start(ctx, DefaultTransportFormatter(path), spanOpts...) //nolint:spancheck
}

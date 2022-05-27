package otelcoap

import (
	"context"

	"github.com/plgd-dev/go-coap/v2/message/codes"
	tcpMessage "github.com/plgd-dev/go-coap/v2/tcp/message"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	"github.com/plgd-dev/hub/v2/pkg/opentelemetry"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	messageSent       = MessageType(otelgrpc.RPCMessageTypeSent)
	messageReceived   = MessageType(otelgrpc.RPCMessageTypeReceived)
	COAPStatusCodeKey = attribute.Key("coap.status_code")
	COAPMethodKey     = attribute.Key("coap.method")
	COAPPathKey       = attribute.Key("coap.path")
)

type MessageType attribute.KeyValue

// Event adds an event of the messageType to the span associated with the
// passed context with id and size (if message is a proto message).
func (m MessageType) Event(ctx context.Context, message *pool.Message) {
	span := trace.SpanFromContext(ctx)
	tcpMsg := tcpMessage.Message{
		Code:    message.Code(),
		Token:   message.Token(),
		Options: message.Options(),
	}
	size, err := tcpMsg.Size()
	if err != nil {
		size = 0
	}
	if bodySize, err := message.BodySize(); err != nil {
		size += int(bodySize)
	}
	span.AddEvent("message", trace.WithAttributes(
		attribute.KeyValue(m),
		semconv.MessageUncompressedSizeKey.Int(size),
	))
}

func DefaultTransportFormatter(path string) string {
	return "COAP " + path
}

func StatusCodeAttr(c codes.Code) attribute.KeyValue {
	return COAPStatusCodeKey.Int64(int64(c))
}

func MessageReceivedEvent(ctx context.Context, message *pool.Message) {
	messageReceived.Event(ctx, message)
}

func MessageSentEvent(ctx context.Context, message *pool.Message) {
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

	return tracer.Start(ctx, DefaultTransportFormatter(path), spanOpts...)
}

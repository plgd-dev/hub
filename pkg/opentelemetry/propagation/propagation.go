package propagation

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
)

// TraceFromCtx set cross-cutting concerns from the Context into the carrier.
func TraceFromCtx(ctx context.Context) propagation.MapCarrier {
	propgator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	// Serialize the context into carrier
	carrier := propagation.MapCarrier{}
	propgator.Inject(ctx, carrier)
	if len(carrier) == 0 {
		return nil
	}
	return carrier
}

// CtxWithTrace reads cross-cutting concerns from the carrier into a Context.
func CtxWithTrace(ctx context.Context, carrier propagation.MapCarrier) context.Context {
	if len(carrier) == 0 {
		return ctx
	}
	propgator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	return propgator.Extract(ctx, carrier)
}

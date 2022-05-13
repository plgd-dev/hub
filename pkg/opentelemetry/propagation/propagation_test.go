package propagation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"
)

func TestPropagation(t *testing.T) {
	type args struct {
		carrier propagation.MapCarrier
	}
	tests := []struct {
		name string
		args args
		want propagation.MapCarrier
	}{
		{
			name: "valid carrier",
			args: args{
				carrier: map[string]string{
					"traceparent": "00-525bfa3fb9a36c20c52b70b0c971e8a2-7add4edefbf0082a-01",
				},
			},
			want: map[string]string{
				"traceparent": "00-525bfa3fb9a36c20c52b70b0c971e8a2-7add4edefbf0082a-01",
			},
		},
		{
			name: "nil carrier",
			args: args{
				carrier: nil,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := CtxWithTrace(context.Background(), tt.args.carrier)
			got := TraceFromCtx(ctx)
			require.Equal(t, tt.want, got)
		})
	}
}

package events

import (
	"testing"
	"time"
)

func TestCalculateEventSignature(t *testing.T) {
	type args struct {
		secret         string
		contentType    string
		eventType      EventType
		subscriptionID string
		seqNum         uint64
		timeStamp      time.Time
		body           []byte
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "nil-body",
			args: args{
				secret:         "a",
				contentType:    "b",
				eventType:      EventType_DevicesOnline,
				subscriptionID: "c",
				seqNum:         0,
				timeStamp:      time.Unix(1, -1),
				body:           nil,
			},
			want: "72150f5f9795e728fa594ece9fa6aa2f0e8877e8d36be89246782cfed00216c3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateEventSignature(tt.args.secret, tt.args.contentType, tt.args.eventType, tt.args.subscriptionID, tt.args.seqNum, tt.args.timeStamp, tt.args.body); got != tt.want {
				t.Errorf("CalculateEventSignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

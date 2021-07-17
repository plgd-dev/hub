package events_test

import (
	"testing"

	commands "github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestResourceLinksPublished_CopyData(t *testing.T) {
	evt := events.ResourceLinksPublished{
		Resources: []*commands.Resource{
			{
				Href:                  "/res1",
				DeviceId:              "dev1",
				ResourceTypes:         []string{"type1", "type2"},
				Interfaces:            []string{"if1", "if2"},
				Anchor:                "anchor1",
				Title:                 "Resource1",
				SupportedContentTypes: []string{"stype1", "stype2"},
				ValidUntil:            123,
				Policies: &commands.Policies{
					BitFlags: 42,
				},
				EndpointInformations: []*commands.EndpointInformation{
					{
						Endpoint: "ep1",
						Priority: 1,
					},
				},
			},
		},
		DeviceId: "dev1",
		AuditContext: &commands.AuditContext{
			UserId:        "501",
			CorrelationId: "1",
		},
		EventMetadata: &events.EventMetadata{
			Version:      42,
			Timestamp:    12345,
			ConnectionId: "con1",
			Sequence:     1,
		},
	}
	type args struct {
		event *events.ResourceLinksPublished
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Identity",
			args: args{
				event: &evt,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e events.ResourceLinksPublished
			e.CopyData(tt.args.event)
			require.True(t, proto.Equal(tt.args.event, &e))
		})
	}
}

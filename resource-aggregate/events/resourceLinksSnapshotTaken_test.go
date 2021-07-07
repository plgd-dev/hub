package events_test

import (
	"testing"

	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestResourceLinksSnapshotTaken_GetNewPublishedLinks(t *testing.T) {
	type fields struct {
		Resources map[string]*commands.Resource
	}
	type args struct {
		pub *events.ResourceLinksPublished
	}
	res := commands.Resource{
		Href:     "/res1",
		DeviceId: "id1",
	}

	resHref := commands.Resource{
		Href:     "/res2",
		DeviceId: "id2",
	}

	res1Upd := commands.Resource{
		Href:     res.Href,
		DeviceId: res.Href + "-upd",
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   []*commands.Resource
	}{
		{
			name: "Nil published",
			fields: fields{
				Resources: nil,
			},
			args: args{
				pub: nil,
			},
			want: nil,
		},
		{
			name: "Nil published resources",
			fields: fields{
				Resources: nil,
			},
			args: args{
				pub: &events.ResourceLinksPublished{
					Resources: nil,
				},
			},
			want: nil,
		},
		{
			name: "Identical",
			fields: fields{
				Resources: map[string]*commands.Resource{
					res.Href: &res,
				},
			},
			args: args{
				pub: &events.ResourceLinksPublished{
					Resources: []*commands.Resource{&res},
				},
			},
			want: make([]*commands.Resource, 0),
		},
		{
			name: "New published resource (1)",
			fields: fields{
				Resources: nil,
			},
			args: args{
				pub: &events.ResourceLinksPublished{
					Resources: []*commands.Resource{&res},
				},
			},
			want: []*commands.Resource{&res},
		},
		{
			name: "New published resource (2)",
			fields: fields{
				Resources: map[string]*commands.Resource{
					res.Href: &res,
				},
			},
			args: args{
				pub: &events.ResourceLinksPublished{
					Resources: []*commands.Resource{&res, &resHref},
				},
			},
			want: []*commands.Resource{&resHref},
		},
		{
			name: "Updated resource",
			fields: fields{
				Resources: map[string]*commands.Resource{
					res.Href:     &res,
					resHref.Href: &resHref,
				},
			},
			args: args{
				pub: &events.ResourceLinksPublished{
					Resources: []*commands.Resource{&res1Upd},
				},
			},
			want: []*commands.Resource{&res1Upd},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &events.ResourceLinksSnapshotTaken{
				Resources: tt.fields.Resources,
			}
			got := e.GetNewPublishedLinks(tt.args.pub)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResourceLinksSnapshotTaken_CopyData(t *testing.T) {
	evt := events.ResourceLinksSnapshotTaken{
		Resources: map[string]*commands.Resource{
			"res1": {
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
		EventMetadata: &events.EventMetadata{
			Version:      42,
			Timestamp:    12345,
			ConnectionId: "con1",
			Sequence:     1,
		},
	}
	type args struct {
		event *events.ResourceLinksSnapshotTaken
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Idetity",
			args: args{
				event: &evt,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e events.ResourceLinksSnapshotTaken
			e.CopyData(tt.args.event)
			require.True(t, proto.Equal(tt.args.event, &e))
		})
	}
}

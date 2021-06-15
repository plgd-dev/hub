package events_test

import (
	"testing"

	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/stretchr/testify/assert"
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

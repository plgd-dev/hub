package events_test

import (
	"testing"

	commands "github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/plgd-dev/hub/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEqualStringSlice(t *testing.T) {
	type args struct {
		x []string
		y []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Nil",
			args: args{},
			want: true,
		},
		{
			name: "Nil and empty",
			args: args{x: nil, y: []string{}},
			want: true,
		},
		{
			name: "Nil and not-nil",
			args: args{x: nil, y: []string{"a"}},
			want: false,
		},
		{
			name: "Identical (equal length)",
			args: args{x: []string{"a", "b"}, y: []string{"a", "b"}},
			want: true,
		},
		{
			name: "Different (equal length)",
			args: args{x: []string{"a", "b"}, y: []string{"b", "a"}},
			want: false,
		},
		{
			name: "Different",
			args: args{x: []string{"b"}, y: []string{"b", "a"}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := events.EqualStringSlice(tt.args.x, tt.args.y)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEqualResource(t *testing.T) {
	type args struct {
		res1 *commands.Resource
		res2 *commands.Resource
	}

	createResource := func() commands.Resource {
		return commands.Resource{
			Href:                  "/res1",
			DeviceId:              "id1",
			ResourceTypes:         []string{"resType1", "resType2"},
			Interfaces:            []string{"if1", "if2", "if3"},
			Anchor:                "anchor",
			Title:                 "title",
			SupportedContentTypes: []string{"contentType1"},
			ValidUntil:            42,
			Policy:                &commands.Policy{BitFlags: 0x42},
		}
	}

	res := createResource()

	resId := createResource()
	resId.DeviceId = "id2"

	resTypesNil := createResource()
	resTypesNil.ResourceTypes = nil

	resTypes2 := createResource()
	resTypes2.ResourceTypes = []string{"resType2"}

	resInterfacesNil := createResource()
	resInterfacesNil.Interfaces = nil

	resInterfaces2 := createResource()
	resInterfaces2.Interfaces = make([]string, 1)
	copy(resInterfaces2.Interfaces, res.Interfaces)

	resAnchor := createResource()
	resAnchor.Anchor = "Anchor2"

	resTitle := createResource()
	resTitle.Title = "Title2"

	resSupportedTypesNil := createResource()
	resSupportedTypesNil.SupportedContentTypes = nil

	resSupportedTypes2 := createResource()
	resSupportedTypes2.SupportedContentTypes = []string{"contentType2"}

	resTimeToLive := createResource()
	resTimeToLive.ValidUntil = 0

	resPolicyNil := createResource()
	resPolicyNil.Policy = nil

	resPolicy2 := createResource()
	resPolicy2.Policy = &commands.Policy{BitFlags: 0}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Identical",
			args: args{
				res1: &res,
				res2: &res,
			},
			want: true,
		},
		{
			name: "Different Id",
			args: args{
				res1: &res,
				res2: &resId,
			},
			want: false,
		},
		{
			name: "Different Resource types (1)",
			args: args{
				res1: &res,
				res2: &resTypesNil,
			},
			want: false,
		},
		{
			name: "Different Resource types (2)",
			args: args{
				res1: &res,
				res2: &resTypes2,
			},
			want: false,
		},
		{
			name: "Different Interfaces (1)",
			args: args{
				res1: &res,
				res2: &resInterfacesNil,
			},
			want: false,
		},
		{
			name: "Different Interfaces (2)",
			args: args{
				res1: &res,
				res2: &resInterfaces2,
			},
			want: false,
		},
		{
			name: "Different Anchor",
			args: args{
				res1: &res,
				res2: &resAnchor,
			},
			want: false,
		},
		{
			name: "Different Title",
			args: args{
				res1: &res,
				res2: &resTitle,
			},
			want: false,
		},
		{
			name: "Different Supported content types (1)",
			args: args{
				res1: &res,
				res2: &resSupportedTypesNil,
			},
			want: false,
		},
		{
			name: "Different Supported content types (2)",
			args: args{
				res1: &res,
				res2: &resSupportedTypes2,
			},
			want: false,
		},
		{
			name: "Different Time To Live",
			args: args{
				res1: &res,
				res2: &resTimeToLive,
			},
			want: false,
		},
		{
			name: "Nil Policy",
			args: args{
				res1: &resPolicyNil,
				res2: &resPolicyNil,
			},
			want: true,
		},
		{
			name: "Different Policy (1)",
			args: args{
				res1: &res,
				res2: &resPolicyNil,
			},
			want: false,
		},
		{
			name: "Different Policy (2)",
			args: args{
				res1: &res,
				res2: &resPolicy2,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := events.EqualResource(tt.args.res1, tt.args.res2)
			test.CheckProtobufs(t, tt.want, got, test.RequireToCheckFunc(require.Equal))
		})
	}
}

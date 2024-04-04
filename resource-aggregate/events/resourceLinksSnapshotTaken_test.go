package events_test

import (
	"sort"
	"testing"

	"github.com/plgd-dev/hub/v2/coap-gateway/resource"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceLinksSnapshotTakenGetNewPublishedLinks(t *testing.T) {
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
		Href:     res.GetHref(),
		DeviceId: res.GetHref() + "-upd",
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
					res.GetHref(): &res,
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
					res.GetHref(): &res,
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
					res.GetHref():     &res,
					resHref.GetHref(): &resHref,
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

var testEventResourceLinksSnapshotTakenDevice = dev1

var (
	testEventResourceLinksSnapshotTakenRes1 = &commands.Resource{
		Href:                  "/res1",
		DeviceId:              testEventResourceLinksSnapshotTakenDevice,
		ResourceTypes:         []string{"type1", "type2"},
		Interfaces:            []string{"if1", "if2"},
		Anchor:                "anchor1",
		Title:                 "Resource1",
		SupportedContentTypes: []string{"stype1", "stype2"},
		ValidUntil:            123,
		Policy: &commands.Policy{
			BitFlags: 42,
		},
		EndpointInformations: []*commands.EndpointInformation{
			{
				Endpoint: "ep1",
				Priority: 1,
			},
		},
	}

	testEventResourceLinksSnapshotTakenRes2 = &commands.Resource{
		Href:                  "/res2",
		DeviceId:              testEventResourceLinksSnapshotTakenDevice,
		ResourceTypes:         []string{"type3"},
		Interfaces:            []string{"if1", "if3"},
		Anchor:                "anchor2",
		Title:                 "Resource2",
		SupportedContentTypes: []string{"stype3"},
		ValidUntil:            456,
		Policy: &commands.Policy{
			BitFlags: 14,
		},
		EndpointInformations: []*commands.EndpointInformation{
			{
				Endpoint: "ep2",
				Priority: 2,
			},
		},
	}

	testEventResourceLinksSnapshotTakenRes3 = &commands.Resource{
		Href:                  "/res3",
		DeviceId:              testEventResourceLinksSnapshotTakenDevice,
		ResourceTypes:         []string{"type4"},
		Interfaces:            []string{"if4"},
		Anchor:                "anchor3",
		Title:                 "Resource3",
		SupportedContentTypes: []string{"stype1", "stype3"},
		ValidUntil:            789,
		Policy: &commands.Policy{
			BitFlags: 77,
		},
		EndpointInformations: []*commands.EndpointInformation{
			{
				Endpoint: "ep3",
				Priority: 3,
			},
		},
	}

	testEventResourceLinksSnapshotTaken events.ResourceLinksSnapshotTaken = events.ResourceLinksSnapshotTaken{
		Resources: map[string]*commands.Resource{
			testEventResourceLinksSnapshotTakenRes1.GetHref(): testEventResourceLinksSnapshotTakenRes1,
			testEventResourceLinksSnapshotTakenRes2.GetHref(): testEventResourceLinksSnapshotTakenRes2,
			testEventResourceLinksSnapshotTakenRes3.GetHref(): testEventResourceLinksSnapshotTakenRes3,
		},
		DeviceId: testEventResourceLinksSnapshotTakenDevice,
		EventMetadata: &events.EventMetadata{
			Version:      42,
			Timestamp:    12345,
			ConnectionId: "con1",
			Sequence:     1,
		},
		AuditContext: commands.NewAuditContext("userID", "", "userID"),
	}
)

func TestResourceLinksSnapshotTakenCopyData(t *testing.T) {
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
				event: &testEventResourceLinksSnapshotTaken,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e events.ResourceLinksSnapshotTaken
			e.CopyData(tt.args.event)
			test.CheckProtobufs(t, tt.args.event, &e, test.RequireToCheckFunc(require.Equal))
		})
	}
}

func TestResourceLinksSnapshotTakenCloneData(t *testing.T) {
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
				event: &testEventResourceLinksSnapshotTaken,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e events.ResourceLinksSnapshotTaken
			e.CloneData(tt.args.event)
			test.CheckProtobufs(t, tt.args.event, &e, test.RequireToCheckFunc(require.Equal))
		})
	}
}

func TestResourceLinksSnapshotTakenCheckInitialized(t *testing.T) {
	type args struct {
		event *events.ResourceLinksSnapshotTaken
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Uninitialized",
			args: args{
				event: &events.ResourceLinksSnapshotTaken{},
			},
			want: false,
		},
		{
			name: "Initialized",
			args: args{
				event: &testEventResourceLinksSnapshotTaken,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.args.event.CheckInitialized())
		})
	}
}

func TestResourceLinksSnapshotTakenHandleEventResourceLinksUnpublished(t *testing.T) {
	type args struct {
		instanceIDs []int64
		hrefs       []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "invalid resource href",
			args: args{
				hrefs: []string{"invalidHref"},
			},
			want: []string{},
		},
		{
			name: "invalid resource instanceID",
			args: args{
				instanceIDs: []int64{123},
			},
			want: []string{},
		},
		{
			name: "unpublish by hrefs",
			args: args{
				hrefs: []string{testEventResourceLinksSnapshotTakenRes1.GetHref(), testEventResourceLinksSnapshotTakenRes2.GetHref()},
			},
			want: []string{testEventResourceLinksSnapshotTakenRes1.GetHref(), testEventResourceLinksSnapshotTakenRes2.GetHref()},
		},
		{
			name: "unpublish by instanceIDs",
			args: args{
				instanceIDs: []int64{
					resource.GetInstanceID(testEventResourceLinksSnapshotTakenRes1.GetHref()),
					resource.GetInstanceID(testEventResourceLinksSnapshotTakenRes3.GetHref()),
				},
			},
			want: []string{testEventResourceLinksSnapshotTakenRes1.GetHref(), testEventResourceLinksSnapshotTakenRes3.GetHref()},
		},
		{
			name: "unpublish all",
			args: args{},
			want: []string{
				testEventResourceLinksSnapshotTakenRes1.GetHref(),
				testEventResourceLinksSnapshotTakenRes2.GetHref(),
				testEventResourceLinksSnapshotTakenRes3.GetHref(),
			},
		},
		{
			name: "unpublish by repeated hrefs",
			args: args{
				hrefs: []string{
					testEventResourceLinksSnapshotTakenRes2.GetHref(),
					testEventResourceLinksSnapshotTakenRes2.GetHref(),
					testEventResourceLinksSnapshotTakenRes2.GetHref(),
				},
			},
			want: []string{testEventResourceLinksSnapshotTakenRes2.GetHref()},
		},
		{
			name: "unpublish by repeated instanceIDs",
			args: args{
				instanceIDs: []int64{
					resource.GetInstanceID(testEventResourceLinksSnapshotTakenRes2.GetHref()),
					resource.GetInstanceID(testEventResourceLinksSnapshotTakenRes2.GetHref()),
					resource.GetInstanceID(testEventResourceLinksSnapshotTakenRes2.GetHref()),
				},
			},
			want: []string{testEventResourceLinksSnapshotTakenRes2.GetHref()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e events.ResourceLinksSnapshotTaken
			e.CloneData(&testEventResourceLinksSnapshotTaken)
			got := e.HandleEventResourceLinksUnpublished(tt.args.instanceIDs, &events.ResourceLinksUnpublished{
				Hrefs: tt.args.hrefs,
			})
			if len(tt.want) == 0 {
				require.Empty(t, got)
				return
			}
			sort.Strings(tt.want)
			sort.Strings(got)
			require.Equal(t, tt.want, got)
		})
	}
}

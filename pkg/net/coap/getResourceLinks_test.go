package coap_test

import (
	"bytes"
	"context"
	"net"
	"strings"
	"testing"

	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/hub/v2/coap-gateway/uri"
	"github.com/plgd-dev/hub/v2/pkg/net/coap"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/stretchr/testify/require"
)

type testCoapConn struct {
	devID string
	links schema.ResourceLinks
	t     *testing.T
}

func (t *testCoapConn) Get(ctx context.Context, path string, opts ...message.Option) (*pool.Message, error) {
	var rtFilter []string
	for _, opt := range opts {
		if opt.ID == message.URIQuery {
			vals := strings.SplitN(string(opt.Value), "=", 2)
			if len(vals) != 2 {
				continue
			}
			if vals[0] == uri.ResourceTypeQueryKey {
				rtFilter = append(rtFilter, vals[1])
			}
		}
	}
	resp := pool.NewMessage(ctx)
	switch path {
	case device.ResourceURI:
		resp.SetCode(codes.Content)
		resp.SetContentFormat(message.AppOcfCbor)
		resp.SetBody(bytes.NewReader(test.EncodeToCbor(t.t, device.Device{
			ID: t.devID,
		})))
		return resp, nil
	case resources.ResourceURI:
		links := t.links
		if len(rtFilter) > 0 {
			links = links.GetResourceLinks(rtFilter...)
		}
		if len(links) == 0 {
			resp.SetCode(codes.BadRequest)
			return resp, nil
		}
		resp.SetCode(codes.Content)
		resp.SetContentFormat(message.AppOcfCbor)
		resp.SetBody(bytes.NewReader(test.EncodeToCbor(t.t, links)))
		return resp, nil
	}
	resp.SetCode(codes.NotFound)
	return resp, nil
}

func (t *testCoapConn) ReleaseMessage(*pool.Message) {
}

func (t *testCoapConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{}
}

func TestGetEndpointsFromDeviceResource(t *testing.T) {
	type args struct {
		ctx      context.Context
		coapConn coap.ClientConn
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				ctx: context.Background(),
				coapConn: &testCoapConn{
					t:     t,
					devID: "dev1",
					links: schema.ResourceLinks{
						{
							DeviceID:      "00000000-0000-0000-0000-000000000001",
							Href:          device.ResourceURI,
							ResourceTypes: []string{device.ResourceType},
							Endpoints: []schema.Endpoint{
								{
									URI: "coap://localhost:5683",
								},
								{
									URI: "coaps://localhost:5684",
								},
							},
						},
					},
				},
			},
			want: []string{"coap://localhost:5683", "coaps://localhost:5684"},
		},
		{
			name: "rt not found",
			args: args{
				ctx: context.Background(),
				coapConn: &testCoapConn{
					t:     t,
					devID: "dev1",
					links: schema.ResourceLinks{
						{
							DeviceID:      "00000000-0000-0000-0000-000000000001",
							Href:          resources.ResourceURI,
							ResourceTypes: []string{resources.ResourceType},
							Endpoints: []schema.Endpoint{
								{
									URI: "coap://localhost:5683",
								},
								{
									URI: "coaps://localhost:5684",
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := coap.GetEndpointsFromDeviceResource(tt.args.ctx, tt.args.coapConn)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestGetResourceLinksWithLinkInterface(t *testing.T) {
	type args struct {
		ctx      context.Context
		coapConn coap.ClientConn
		href     string
	}
	tests := []struct {
		name    string
		args    args
		want    schema.ResourceLinks
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				ctx:  context.Background(),
				href: resources.ResourceURI,
				coapConn: &testCoapConn{
					t:     t,
					devID: "dev1",
					links: schema.ResourceLinks{
						{
							DeviceID:      "00000000-0000-0000-0000-000000000001",
							Href:          device.ResourceURI,
							ResourceTypes: []string{device.ResourceType},
							Endpoints: []schema.Endpoint{
								{
									URI: "coap://localhost:5683",
								},
								{
									URI: "coaps://localhost:5684",
								},
							},
						},
					},
				},
			},
			want: schema.ResourceLinks{
				{
					DeviceID:      "00000000-0000-0000-0000-000000000001",
					Href:          device.ResourceURI,
					ResourceTypes: []string{device.ResourceType},
					Endpoints: []schema.Endpoint{
						{
							URI: "coap://localhost:5683",
						},
						{
							URI: "coaps://localhost:5684",
						},
					},
				},
			},
		},
		{
			name: "rt not found",
			args: args{
				ctx:  context.Background(),
				href: device.ResourceURI,
				coapConn: &testCoapConn{
					t:     t,
					devID: "dev1",
					links: schema.ResourceLinks{
						{
							DeviceID:      "00000000-0000-0000-0000-000000000001",
							Href:          resources.ResourceURI,
							ResourceTypes: []string{resources.ResourceType},
							Endpoints: []schema.Endpoint{
								{
									URI: "coap://localhost:5683",
								},
								{
									URI: "coaps://localhost:5684",
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := coap.GetResourceLinksWithLinkInterface(tt.args.ctx, tt.args.coapConn, tt.args.href)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

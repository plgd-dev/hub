package client_test

import (
	"testing"

	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestConfigPublisher(t *testing.T) {
	basicConfigYAML := `
url: "test"
flusherTimeout: 1s
pendingLimits:
  msgLimit: 1
  bytesLimit: 1
tls:
  certFile: "test"
  keyFile: "test"
  caPool: "test"
`
	var basicConfig client.ConfigPublisher
	err := yaml.Unmarshal([]byte(basicConfigYAML), &basicConfig)
	require.NoError(t, err)
	err = basicConfig.Validate()
	require.NoError(t, err)

	type args struct {
		data string
	}
	tests := []struct {
		name    string
		args    args
		want    client.ConfigPublisher
		wantErr bool
	}{
		{
			name: "invalid - lead resource type filter with bad string filter",
			args: args{
				data: basicConfigYAML + `
leadResourceType:
  enabled: true
  filter: invalid`,
			},
			wantErr: true,
		},
		{
			name: "invalid - lead resource type filter with bad regex filter",
			args: args{
				data: basicConfigYAML + `
leadResourceType:
  enabled: true
  regexFilter:
    - "("`,
			},
			wantErr: true,
		},
		{
			name: "valid with lead resource type",
			args: args{
				data: basicConfigYAML + `
leadResourceType:
  enabled: true
  regexFilter:
    - "^a"
    - "b$"
  filter: "first"
  useUUID: true`,
			},
			want: func() client.ConfigPublisher {
				c := basicConfig
				c.LeadResourceType = &client.LeadResourceTypeConfig{
					Enabled: true,
					RegexFilter: []string{
						"^a",
						"b$",
					},
					Filter:  client.LeadResourceTypeFilter_First,
					UseUUID: true,
				}
				err := c.Validate()
				require.NoError(t, err)
				return c
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got client.ConfigPublisher
			err := yaml.Unmarshal([]byte(tt.args.data), &got)
			require.NoError(t, err)
			err = got.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

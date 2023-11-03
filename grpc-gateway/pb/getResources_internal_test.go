package pb

import (
	"encoding/base64"
	"testing"

	commands "github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/stretchr/testify/require"
)

func decodeBase64(t *testing.T, s string) []byte {
	b, err := base64.StdEncoding.DecodeString(s)
	require.NoError(t, err)
	return b
}

func TestResourceIdFilterFromString(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    *ResourceIdFilter
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				s: "di/1/2/3",
			},
			want: &ResourceIdFilter{
				ResourceId: &commands.ResourceId{
					DeviceId: "di",
					Href:     "/1/2/3",
				},
			},
		},
		{
			name: "valid-etag",
			args: args{
				s: "afa5e64b-98de-4d7d-597f-91cb50a13f9c/light/1?etag=eE+rCz0JBgA=",
			},
			want: &ResourceIdFilter{
				ResourceId: &commands.ResourceId{
					DeviceId: "afa5e64b-98de-4d7d-597f-91cb50a13f9c",
					Href:     "/light/1",
				},
				Etag: [][]byte{decodeBase64(t, "eE+rCz0JBgA=")},
			},
		},
		{
			name: "valid-etags",
			args: args{
				s: "be7f4bf7-1717-4262-5d56-56939fe1f137/light/1?etag=TVRJeg==&etag=TkRVMg==&etag=VPClrD4JBgA=",
			},
			want: &ResourceIdFilter{
				ResourceId: &commands.ResourceId{
					DeviceId: "be7f4bf7-1717-4262-5d56-56939fe1f137",
					Href:     "/light/1",
				},
				Etag: [][]byte{decodeBase64(t, "TVRJeg=="), decodeBase64(t, "TkRVMg=="), decodeBase64(t, "VPClrD4JBgA=")},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resourceIdFilterFromString(tt.args.s)
			require.Equal(t, tt.want, got)
		})
	}
}

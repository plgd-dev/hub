package file_test

import (
	"testing"

	"github.com/plgd-dev/cloud/pkg/file"
	"github.com/plgd-dev/cloud/test/config"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	type args struct {
		path string
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "ok - whole file",
			args: args{
				path: config.CA_POOL,
			},
		},
		{
			name: "ok - part of file",
			args: args{
				path: config.CA_POOL,
				data: make([]byte, 1),
			},
		},
		{
			name: "not exist",
			args: args{
				path: "not/exist/file",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := file.Load(tt.args.path, tt.args.data)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, got)
		})
	}
}

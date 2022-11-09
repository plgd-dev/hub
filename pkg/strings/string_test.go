package strings

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToStringArray(t *testing.T) {
	type args struct {
		v interface{}
	}
	tests := []struct {
		name   string
		args   args
		want   []string
		wantOk bool
	}{
		{
			name: "nil",
			args: args{
				v: nil,
			},
			want:   nil,
			wantOk: true,
		},
		{
			name: "string",
			args: args{
				v: "test",
			},
			want:   []string{"test"},
			wantOk: true,
		},
		{
			name: "string array",
			args: args{
				v: []string{"test1", "test2"},
			},
			want:   []string{"test1", "test2"},
			wantOk: true,
		},
		{
			name: "interface array",
			args: args{
				v: []interface{}{"test1", "test2"},
			},
			want:   []string{"test1", "test2"},
			wantOk: true,
		},
		{
			name: "interface array with not same type",
			args: args{
				v: []interface{}{"test1", 1},
			},
			want:   nil,
			wantOk: false,
		},
		{
			name: "invalid value",
			args: args{
				v: 1,
			},
			want:   nil,
			wantOk: false,
		},
		{
			name: "invalid array",
			args: args{
				v: []interface{}{0, 1},
			},
			want:   nil,
			wantOk: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ToStringArray(tt.args.v)
			require.Equal(t, tt.wantOk, ok)
			if !ok {
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}

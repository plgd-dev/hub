package mongodb_test

import (
	"testing"

	"github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertStringValueToInt64(t *testing.T) {
	type args struct {
		data               interface{}
		paths              []string
		permitMissingPaths bool
	}

	tests := []struct {
		name      string
		args      args
		want      interface{}
		wantErr   bool
		ignoreErr bool
	}{
		{
			name: "emptyPath",
			args: args{
				data:  map[string]interface{}{},
				paths: []string{""},
			},
			wantErr: true,
		},
		{
			name: "invalidPath",
			args: args{
				data:  map[string]interface{}{},
				paths: []string{"foo"},
			},
			wantErr: true,
		},
		{
			name: "directValue",
			args: args{
				data:  "123",
				paths: []string{"."},
			},
			want: int64(123),
		},
		{
			name: "arrayValue",
			args: args{
				data: []interface{}{
					"123",
					"456",
					"789",
				},
				paths: []string{".[0]", ".[2]"},
			},
			want: []interface{}{int64(123), "456", int64(789)},
		},
		{
			name: "mapValue",
			args: args{
				data: map[string]interface{}{
					"foo": "123",
				},
				paths: []string{".foo"},
			},
			want: map[string]interface{}{
				"foo": int64(123),
			},
		},
		{
			name: "mapArrayValue",
			args: args{
				data: map[string]interface{}{
					"foo": []interface{}{
						"123",
						"456",
						"789",
					},
				},
				paths: []string{".foo[0]", ".foo[2]"},
			},
			want: map[string]interface{}{
				"foo": []interface{}{int64(123), "456", int64(789)},
			},
		},
		{
			name: "nestedMapValue",
			args: args{
				data: map[string]interface{}{
					"foo": map[string]interface{}{
						"bar": "123",
					},
				},
				paths: []string{".foo.bar"},
			},
			want: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": int64(123),
				},
			},
		},
		{
			name: "nestedArrayMapValue",
			args: args{
				data: map[string]interface{}{
					"foo": []interface{}{
						map[string]interface{}{
							"bar": "123",
						},
						map[string]interface{}{
							"bar": "456",
						},
					},
				},
				paths: []string{".foo[0].bar", ".foo[1].bar"},
			},
			want: map[string]interface{}{
				"foo": []interface{}{
					map[string]interface{}{
						"bar": int64(123),
					},
					map[string]interface{}{
						"bar": int64(456),
					},
				},
			},
		},
		{
			name: "nestedArrayMapAllValues",
			args: args{
				data: map[string]interface{}{
					"foo": []interface{}{
						map[string]interface{}{
							"bar": "123",
						},
						map[string]interface{}{
							"bar": "456",
						},
					},
				},
				paths: []string{".foo[*].bar"},
			},
			want: map[string]interface{}{
				"foo": []interface{}{
					map[string]interface{}{
						"bar": int64(123),
					},
					map[string]interface{}{
						"bar": int64(456),
					},
				},
			},
		},
		{
			name: "nestedArrayMapWithMissingPathsAllValues",
			args: args{
				data: map[string]interface{}{
					"foo": []interface{}{
						map[string]interface{}{
							"bar": "123",
						},
						map[string]interface{}{
							"efg": "456",
						},
						map[string]interface{}{
							"bar": "789",
						},
					},
				},
				paths:              []string{".foo[*].bar"},
				permitMissingPaths: true,
			},
			want: map[string]interface{}{
				"foo": []interface{}{
					map[string]interface{}{
						"bar": int64(123),
					},
					map[string]interface{}{
						"efg": "456",
					},
					map[string]interface{}{
						"bar": int64(789),
					},
				},
			},
		},
		{
			name: "mapArrayAllValues",
			args: args{
				data: map[string]interface{}{
					"foo": []interface{}{
						"123",
						"456",
						"789",
					},
				},
				paths: []string{".foo[*]"},
			},
			want: map[string]interface{}{
				"foo": []interface{}{int64(123), int64(456), int64(789)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mongodb.ConvertStringValueToInt64(tt.args.data, tt.args.permitMissingPaths, tt.args.paths...)
			if tt.wantErr {
				require.Error(t, err)
				if !tt.ignoreErr {
					return
				}
			}
			if !tt.ignoreErr {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

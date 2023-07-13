package observation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestObservedResourceEncodeETagsForIncrementChanged(t *testing.T) {
	type fields struct {
		etags [][]byte
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "empty",
			fields: fields{
				etags: nil,
			},
		},
		{
			name: "only-latest-etag",
			fields: fields{
				etags: [][]byte{
					[]byte("0"),
				},
			},
		},
		{
			name: "one-etag",
			fields: fields{
				etags: [][]byte{
					[]byte("0"),
					[]byte("01234567"),
				},
			},
			want: []string{
				prefixQueryIncChanged + "3031323334353637",
			},
		},
		{
			name: "two-etags",
			fields: fields{
				etags: [][]byte{
					[]byte("0"),
					[]byte("1"),
					[]byte("2"),
				},
			},
			want: []string{
				prefixQueryIncChanged + "31,32",
			},
		},
		{
			name: "two-etags-invalid-etag",
			fields: fields{
				etags: [][]byte{
					[]byte("0"),
					[]byte("1"),
					[]byte("2"),
					[]byte("invalid-etag-is-ignored"),
				},
			},
			want: []string{
				prefixQueryIncChanged + "31,32",
			},
		},
		{
			name: "multiple-etags",
			fields: fields{
				etags: [][]byte{
					[]byte("01234567"),
					[]byte("01234567"),
					[]byte("01234567"),
					[]byte("01234567"),
					[]byte("01234567"),
					[]byte("01234567"),
					[]byte("01234567"),
					[]byte("01234567"),
					[]byte("01234567"),
					[]byte("01234567"),
					[]byte("01234567"),
					[]byte("01234567"),
					[]byte("01234567"),
					[]byte("01234567"),
					[]byte("01234567"), // 15
					[]byte("01234567"),
				},
			},
			want: []string{
				prefixQueryIncChanged + "3031323334353637,3031323334353637,3031323334353637,3031323334353637,3031323334353637,3031323334353637,3031323334353637,3031323334353637,3031323334353637,3031323334353637,3031323334353637,3031323334353637,3031323334353637,3031323334353637",
				prefixQueryIncChanged + "3031323334353637",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &observedResource{
				etags: tt.fields.etags,
			}
			got := r.EncodeETagsForIncrementChanged()
			for _, g := range got {
				assert.Less(t, len(g), 255) // RFC 7641 - Uri-query length
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

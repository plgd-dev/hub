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
				prefixQueryIncChanges + "MDEyMzQ1Njc",
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
				prefixQueryIncChanges + "MQ,Mg",
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
				prefixQueryIncChanges + "MQ,Mg",
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
					[]byte("01234567"),
					[]byte("01234567"),
					[]byte("01234567"),
					[]byte("01234567"),
					[]byte("01234567"),
					[]byte("01234567"),
					[]byte("01234567"),
					[]byte("01234567"), // 22
				},
			},
			want: []string{
				prefixQueryIncChanges + "MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc",
				prefixQueryIncChanges + "MDEyMzQ1Njc",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &observedResource{
				etags: tt.fields.etags,
			}
			got := r.EncodeETagsForIncrementChanges()
			for _, g := range got {
				assert.Less(t, len(g), 255) // RFC 7641 - Uri-query length
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

package observation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestObservedResourceEncodeETagsForIncrementChanged(t *testing.T) {
	tests := []struct {
		name  string
		etags [][]byte
		want  []string
	}{
		{
			name:  "empty",
			etags: nil,
		},
		{
			name:  "not-nil",
			etags: [][]byte{},
		},
		{
			name: "one-etag",
			etags: [][]byte{
				[]byte("01234567"),
			},
			want: []string{
				prefixQueryIncChanges + "MDEyMzQ1Njc",
			},
		},
		{
			name: "two-etags",
			etags: [][]byte{
				[]byte("1"),
				[]byte("2"),
			},
			want: []string{
				prefixQueryIncChanges + "MQ,Mg",
			},
		},
		{
			name: "two-etags-invalid-etag",
			etags: [][]byte{
				[]byte("1"),
				[]byte("2"),
				[]byte("invalid-etag-is-ignored"),
			},
			want: []string{
				prefixQueryIncChanges + "MQ,Mg",
			},
		},
		{
			name: "multiple-etags",
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
				[]byte("01234567"), // 21
			},
			want: []string{
				prefixQueryIncChanges + "MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc",
				prefixQueryIncChanges + "MDEyMzQ1Njc",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeETagsForIncrementChanges(tt.etags)
			for _, g := range got {
				assert.Less(t, len(g), 255) // RFC 7641 - Uri-query length
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

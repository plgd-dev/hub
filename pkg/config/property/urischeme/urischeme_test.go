package urischeme_test

import (
	"testing"

	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/stretchr/testify/require"
)

func TestURISchemeReadData(t *testing.T) {
	tests := []struct {
		name    string
		scheme  urischeme.URIScheme
		want    []byte
		wantErr bool
	}{
		{
			name:    "valid base64 data url",
			scheme:  "data:text/plain;base64,SGVsbG8sIFdvcmxkIQ==",
			want:    []byte("Hello, World!"),
			wantErr: false,
		},
		{
			name:    "valid base64 data url",
			scheme:  "data:;base64,SGVsbG8sIFdvcmxkIQ==",
			want:    []byte("Hello, World!"),
			wantErr: false,
		},
		{
			name:    "valid ascii data url",
			scheme:  urischeme.URIScheme("data:,HelloWorld!"),
			want:    []byte("HelloWorld!"),
			wantErr: false,
		},
		{
			name:    "invalid data url",
			scheme:  "invalid",
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.scheme.Read()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestURISchemeFilePath(t *testing.T) {
	tests := []struct {
		name     string
		scheme   urischeme.URIScheme
		expected string
	}{
		{
			name:     "data scheme",
			scheme:   "data:text/plain;base64,SGVsbG8sIFdvcmxkIQ==",
			expected: "",
		},
		{
			name:     "file scheme",
			scheme:   "file:///path/to/file",
			expected: "/path/to/file",
		},
		{
			name:     "unsupported file scheme",
			scheme:   "file://{host}/path/to/file",
			expected: "",
		},
		{
			name:     "other scheme",
			scheme:   "http://example.com",
			expected: "",
		},
		// backward compatibility
		{
			name:     "empty scheme",
			scheme:   "/path/to/file",
			expected: "/path/to/file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.scheme.FilePath()
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestToFilePathArray(t *testing.T) {
	tests := []struct {
		name string
		v    []urischeme.URIScheme
		want []string
	}{
		{
			name: "empty slice",
			v:    []urischeme.URIScheme{},
			want: []string{},
		},
		{
			name: "slice with one valid scheme",
			v:    []urischeme.URIScheme{"file:///path/to/file"},
			want: []string{"/path/to/file"},
		},
		{
			name: "slice with one path",
			v:    []urischeme.URIScheme{"/path/to/file"},
			want: []string{"/path/to/file"},
		},
		{
			name: "slice with multiple schemes",
			v:    []urischeme.URIScheme{"file:///path/to/file", "http://example.com", "path/to/file", "data:text/plain;base64,SGVsbG8sIFdvcmxkIQ=="},
			want: []string{"/path/to/file", "path/to/file"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := urischeme.ToFilePathArray(tt.v)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestToURISchemeArray(t *testing.T) {
	tests := []struct {
		name string
		v    []string
		want []urischeme.URIScheme
	}{
		{
			name: "empty slice",
			v:    []string{},
			want: nil,
		},
		{
			name: "single element slice",
			v:    []string{"http"},
			want: []urischeme.URIScheme{"http"},
		},
		{
			name: "multiple element slice",
			v:    []string{"http", "https", "ftp"},
			want: []urischeme.URIScheme{"http", "https", "ftp"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := urischeme.ToURISchemeArray(tt.v)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestURISchemeIsFile(t *testing.T) {
	tests := []struct {
		name   string
		scheme urischeme.URIScheme
		want   bool
	}{
		{
			name:   "file scheme",
			scheme: "file:///path/to/file",
			want:   true,
		},
		{
			name:   "data scheme",
			scheme: "data:text/plain;base64,SGVsbG8sIFdvcmxkIQ==",
			want:   false,
		},
		{
			name:   "http scheme",
			scheme: "http://example.com",
			want:   false,
		},
		{
			name:   "empty scheme",
			scheme: "",
			want:   false,
		},
		{
			name:   "backward compatibility",
			scheme: "/a/b/c",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.scheme.IsFile()
			require.Equal(t, tt.want, got)
		})
	}
}

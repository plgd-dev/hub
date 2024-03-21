package strings_test

import (
	"testing"

	"github.com/plgd-dev/hub/v2/pkg/strings"
	"github.com/stretchr/testify/require"
)

func TestUnescape(t *testing.T) {
	tbl := []struct {
		name         string
		input        string
		mode         strings.UnescapingMode
		multisegment bool
		expected     string
		expectedErr  error
	}{
		{
			name:         "No escaping required",
			input:        "hello world",
			mode:         strings.UnescapingModeAllCharacters,
			multisegment: true,
			expected:     "hello world",
			expectedErr:  nil,
		},
		{
			name:         "Single character escaping",
			input:        "/%20",
			mode:         strings.UnescapingModeAllCharacters,
			multisegment: true,
			expected:     "/ ",
			expectedErr:  nil,
		},
		{
			name:         "Multiple character escaping",
			input:        "hello%20world",
			mode:         strings.UnescapingModeAllCharacters,
			multisegment: true,
			expected:     "hello world",
			expectedErr:  nil,
		},
		{
			name:         "Invalid escape sequence",
			input:        "%2",
			mode:         strings.UnescapingModeAllCharacters,
			multisegment: true,
			expected:     "",
			expectedErr:  strings.MalformedSequenceError("%2"),
		},
		{
			name:         "Escaping except slash with multisegment=false",
			input:        "/%2F%23%20",
			mode:         strings.UnescapingModeAllExceptSlash,
			multisegment: true,
			expected:     "/%2F# ",
			expectedErr:  nil,
		},
		{
			name:         "Escaping except reserved with multisegment=true",
			input:        "/%2F%23%20",
			mode:         strings.UnescapingModeAllExceptReserved,
			multisegment: true,
			expected:     "/%2F%23 ",
			expectedErr:  nil,
		},
		{
			name:         "Escaping all characters with multisegment=true",
			input:        "/%2F%23%20",
			mode:         strings.UnescapingModeAllCharacters,
			multisegment: true,
			expected:     "//# ",
			expectedErr:  nil,
		},
	}

	for _, test := range tbl {
		t.Run(test.name, func(t *testing.T) {
			result, err := strings.Unescape(test.input, test.mode, test.multisegment)
			require.Equal(t, test.expected, result)
			require.Equal(t, test.expectedErr, err)
		})
	}
}

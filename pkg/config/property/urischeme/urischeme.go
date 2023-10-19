package urischeme

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/vincent-petithory/dataurl"
)

// URIScheme represents a string type that defines a URI scheme.
// Supported URI schemes are:
// - /path/to/file (backward compatibility)
// - file:///path/to/file (RFC 8089)
// - data:;base64,SGVsbG8sIFdvcmxkIQ== (RFC 2397)
// - data:,plgd (RFC 2397)
type URIScheme string

var anyScheme = regexp.MustCompile(`^[a-zA-Z]+:+//`)

// ToURISchemeArray converts a slice of strings to a slice of URIScheme.
func ToURISchemeArray(v []string) []URIScheme {
	result := make([]URIScheme, 0, len(v))
	for _, s := range v {
		result = append(result, URIScheme(s))
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// ToFilePathArray returns an array of file paths from an array of URIScheme.
// If a URIScheme does not have a file path, it is not included in the result.
func ToFilePathArray(v []URIScheme) []string {
	result := make([]string, 0, len(v))
	for _, s := range v {
		f := s.FilePath()
		if f != "" {
			result = append(result, s.FilePath())
		}
	}
	return result
}

// IsFile returns true if the URIScheme is a file scheme.
func (p URIScheme) IsFile() bool {
	if len(p) == 0 {
		return false
	}
	if p.IsData() {
		return false
	}
	if strings.HasPrefix(string(p), "file:///") {
		return true
	}
	if anyScheme.MatchString(string(p)) {
		return false
	}
	// backward compatibility
	return true
}

// FilePath returns the file path of the URI scheme if it is a file URI scheme.
// If the URI scheme is a data URI scheme or any other supported URI scheme, it returns an empty string.
func (p URIScheme) FilePath() string {
	if len(p) == 0 {
		return ""
	}
	if p.IsData() {
		return ""
	}
	if strings.HasPrefix(string(p), "file:///") {
		return string(p)[7:]
	}
	if anyScheme.MatchString(string(p)) {
		return ""
	}
	return string(p)
}

// IsData returns true if the URIScheme is a data URI scheme.
func (p URIScheme) IsData() bool {
	return strings.HasPrefix(string(p), "data:")
}

func (p URIScheme) readFile() ([]byte, error) {
	return os.ReadFile(string(p))
}

func (p URIScheme) readData() (data []byte, err error) {
	defer func() {
		if err1 := recover(); err1 != nil {
			err = fmt.Errorf("cannot load data: %v", err1)
		}
	}()
	dataURL, err := dataurl.DecodeString(string(p))
	if err != nil {
		return nil, err
	}
	return dataURL.Data, nil
}

// Read reads the value of the URIScheme property.
// If the property is a data URI, it reads the data from the URI.
// Otherwise, it reads the data from the file specified by the URI.
func (p URIScheme) Read() ([]byte, error) {
	if p.IsData() {
		return p.readData()
	}
	return p.readFile()
}

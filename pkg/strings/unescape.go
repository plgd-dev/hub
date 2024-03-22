package strings

import (
	"strconv"
	"strings"
)

type MalformedSequenceError string

func (e MalformedSequenceError) Error() string {
	return "malformed path escape " + strconv.Quote(string(e))
}

// UnescapingMode defines the behavior of ServeMux when unescaping path parameters.
type UnescapingMode int

const (
	// UnescapingModeAllExceptReserved unescapes all path parameters except RFC 6570
	// reserved characters.
	UnescapingModeAllExceptReserved UnescapingMode = 1

	// UnescapingModeAllExceptSlash unescapes URL path parameters except path
	// separators, which will be left as "%2F".
	UnescapingModeAllExceptSlash UnescapingMode = 2

	// UnescapingModeAllCharacters unescapes all URL path parameters.
	UnescapingModeAllCharacters UnescapingMode = 3
)

/*
 * The following code is adopted and modified from Go's standard library
 * and carries the attached license.
 *
 *     Copyright 2009 The Go Authors. All rights reserved.
 *     Use of this source code is governed by a BSD-style
 *     license that can be found in the LICENSE file.
 */

// isHex returns whether or not the given byte is a valid hex character
func isHex(c byte) bool {
	switch {
	case '0' <= c && c <= '9':
		return true
	case 'a' <= c && c <= 'f':
		return true
	case 'A' <= c && c <= 'F':
		return true
	}
	return false
}

func isRFC6570Reserved(c byte) bool {
	switch c {
	case '!', '#', '$', '&', '\'', '(', ')', '*',
		'+', ',', '/', ':', ';', '=', '?', '@', '[', ']':
		return true
	default:
		return false
	}
}

// unHex converts a hex point to the bit representation
func unHex(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

// shouldUnescapeWithMode returns true if the character is escapable with the
// given mode
func shouldUnescapeWithMode(c byte, mode UnescapingMode) bool {
	switch mode {
	case UnescapingModeAllExceptReserved:
		if isRFC6570Reserved(c) {
			return false
		}
	case UnescapingModeAllExceptSlash:
		if c == '/' {
			return false
		}
	case UnescapingModeAllCharacters:
		return true
	}
	return true
}

// checkWellFormed checks if the given string is well-formed
func checkWellFormed(s string) (bool, error) {
	n := 0
	for i := 0; i < len(s); {
		if s[i] != '%' {
			i++
			continue
		}
		n++
		if i+2 >= len(s) || !isHex(s[i+1]) || !isHex(s[i+2]) {
			s = s[i:]
			if len(s) > 3 {
				s = s[:3]
			}
			return false, MalformedSequenceError(s)
		}
		i += 3
	}
	return n != 0, nil
}

// Unescape unescapes a path string using the provided mode
func Unescape(s string, mode UnescapingMode, multisegment bool) (string, error) {
	if !multisegment {
		mode = UnescapingModeAllCharacters
	}

	// Count %, check that they're well-formed.
	needEscape, err := checkWellFormed(s)
	if err != nil {
		return "", err
	}
	if !needEscape {
		return s, nil
	}

	var t strings.Builder
	t.Grow(len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '%':
			c := unHex(s[i+1])<<4 | unHex(s[i+2])
			if shouldUnescapeWithMode(c, mode) {
				t.WriteByte(c)
				i += 2
				continue
			}
			fallthrough
		default:
			t.WriteByte(s[i])
		}
	}

	return t.String(), nil
}

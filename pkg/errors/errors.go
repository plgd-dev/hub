package errors

import (
	"errors"
	"fmt"
)

type wrappedError struct {
	msg  string
	errs []error
}

// Error returns the stored error string
func (we wrappedError) Error() string {
	return we.msg
}

func (we wrappedError) Is(err error) bool {
	for _, e := range we.errs {
		if errors.Is(e, err) {
			return true
		}
	}
	return false
}

// Simple error wrapper that appends a message to the error string and chains errors with a "%s(: %s)*" format
func NewErrorWithMessage(message string, err error, more ...error) error {
	var format string
	var args []interface{}
	if message != "" {
		format = "%s: %s"
		args = []interface{}{err, message}
	} else {
		format = "%s"
		args = []interface{}{err}
	}
	errs := []error{err}

	for _, e := range more {
		format += ": %s"
		args = append(args, e)
		errs = append(errs, e)
	}

	err = &wrappedError{
		msg:  fmt.Sprintf(format, args...),
		errs: errs,
	}
	return err
}

// Simple error wrapper that chains errors and prints them with a "%s(: %s)*" format
func NewError(err error, more ...error) error {
	return NewErrorWithMessage("", err, more...)
}

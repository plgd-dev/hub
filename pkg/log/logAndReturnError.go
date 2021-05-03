package log

import (
	"context"
	"errors"
	"io"
)

func LogAndReturnError(err error) error {
	if err == nil {
		return err
	}
	if errors.Is(err, io.EOF) {
		Debugf("%v", err)
		return err
	}
	if errors.Is(err, context.Canceled) {
		Debugf("%v", err)
		return err
	}
	Errorf("%v", err)
	return err
}

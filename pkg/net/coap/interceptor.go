package coap

import (
	"context"

	"github.com/plgd-dev/go-coap/v2/message/codes"
)

type Interceptor = func(ctx context.Context, code codes.Code, path string) (context.Context, error)

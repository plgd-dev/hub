package http

import (
	fmt "fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestReadErrorResponse(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "empty",
			args: args{
				err: nil,
			},
			wantErr: ErrInternalServerError,
		},
		{
			name: "error",
			args: args{
				err: fmt.Errorf("a"),
			},
			wantErr: fmt.Errorf("a"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := fasthttp.AcquireResponse()
			defer fasthttp.ReleaseResponse(resp)

			WriteErrorResponse(tt.args.err, resp)
			err := ReadErrorResponse(resp)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/plgd-dev/go-coap/v2/net/blockwise"
	"github.com/plgd-dev/go-coap/v2/net/monitor/inactivity"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/stretchr/testify/require"
)

func TestBlockWiseTransferSZXFromString(t *testing.T) {
	type args struct {
		szx string
	}
	tests := []struct {
		name    string
		args    args
		want    blockwise.SZX
		wantErr bool
	}{
		{name: "16", args: args{szx: "16"}, want: blockwise.SZX16},
		{name: "32", args: args{szx: "32"}, want: blockwise.SZX32},
		{name: "64", args: args{szx: "64"}, want: blockwise.SZX64},
		{name: "128", args: args{szx: "128"}, want: blockwise.SZX128},
		{name: "256", args: args{szx: "256"}, want: blockwise.SZX256},
		{name: "512", args: args{szx: "512"}, want: blockwise.SZX512},
		{name: "1024", args: args{szx: "1024"}, want: blockwise.SZX1024},
		{name: "bert", args: args{szx: "bert"}, want: blockwise.SZXBERT},
		{name: "invalid", args: args{szx: "invalid"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := blockWiseTransferSZXFromString(tt.args.szx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

type testClientConn struct {
	ctx context.Context
	err error
}

func (t *testClientConn) Context() context.Context {
	return t.ctx
}

func (t *testClientConn) Close() error {
	return t.err
}

func TestGetOnInactivityFn(t *testing.T) {
	type args struct {
		cc inactivity.ClientConn
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "conn",
			args: args{
				cc: &testClientConn{
					ctx: context.Background(),
					err: nil,
				},
			},
		},
		{
			name: "connWithErr",
			args: args{
				cc: &testClientConn{
					ctx: context.Background(),
					err: fmt.Errorf("err"),
				},
			},
		},
	}
	onInactivity := getOnInactivityFn(log.Get())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			onInactivity(tt.args.cc)
		})
	}
}

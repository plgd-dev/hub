//go:build test
// +build test

package service

import (
	"fmt"
	"strings"
	"testing"

	"github.com/plgd-dev/go-coap/v3/net/blockwise"
	"github.com/stretchr/testify/require"
)

func blockWiseTransferSZXFromString(s string) (blockwise.SZX, error) {
	switch strings.ToLower(s) {
	case "16":
		return blockwise.SZX16, nil
	case "32":
		return blockwise.SZX32, nil
	case "64":
		return blockwise.SZX64, nil
	case "128":
		return blockwise.SZX128, nil
	case "256":
		return blockwise.SZX256, nil
	case "512":
		return blockwise.SZX512, nil
	case "1024":
		return blockwise.SZX1024, nil
	case "bert":
		return blockwise.SZXBERT, nil
	}
	return blockwise.SZX(0), fmt.Errorf("invalid value %v", s)
}

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

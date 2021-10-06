package test

import (
	"testing"

	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/stretchr/testify/require"
)

func DecodeCbor(t *testing.T, data []byte) interface{} {
	var v interface{}
	err := cbor.Decode(data, &v)
	require.NoError(t, err)
	return v
}

func EncodeToCbor(t *testing.T, v interface{}) []byte {
	d, err := cbor.Encode(v)
	require.NoError(t, err)
	return d
}

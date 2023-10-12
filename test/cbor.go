package test

import (
	"testing"

	"github.com/fxamacker/cbor/v2"
	pkgCbor "github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/stretchr/testify/require"
)

func DecodeCbor(t *testing.T, data []byte) interface{} {
	var v interface{}
	err := pkgCbor.Decode(data, &v)
	require.NoError(t, err)
	return v
}

func EncodeToSortedCbor(v interface{}) ([]byte, error) {
	enc, err := cbor.EncOptions{Sort: cbor.SortCanonical}.EncMode()
	if err != nil {
		return nil, err
	}
	return enc.Marshal(v)
}

func EncodeToCbor(t *testing.T, v interface{}) []byte {
	d, err := EncodeToSortedCbor(v)
	require.NoError(t, err)
	return d
}

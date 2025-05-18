package cqldb

import (
	"fmt"
	"strings"
)

const (
	DefaultPort = 9142

	NullValue = "null"

	UUIDType   = "UUID"
	Int64Type  = "bigint"
	BytesType  = "blob"
	StringType = "text"

	WhereClause = "WHERE"
	FromClause  = "FROM"

	SelectCommand = "SELECT"
)

func EncodeToBlob(data []byte, buffer *strings.Builder) {
	if len(data) == 0 {
		buffer.WriteString(NullValue)
		return
	}
	buffer.Grow(buffer.Len() + 2 + len(data)*2)
	buffer.WriteString("0x")
	for _, v := range data {
		fmt.Fprintf(buffer, "%02x", v)
	}
}

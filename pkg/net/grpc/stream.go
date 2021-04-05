package grpc

import (
	"io"

	"google.golang.org/grpc"
)

func NewIterator(stream grpc.ClientStream, err error) *Iterator {
	if err != nil {
		return &Iterator{Err: err}
	}
	return &Iterator{stream: stream}
}

type Iterator struct {
	stream grpc.ClientStream
	Err    error
}

func (it *Iterator) Next(v interface{}) bool {
	if it.stream == nil {
		return false
	}
	err := it.stream.RecvMsg(v)
	if err == io.EOF {
		it.Close()
		return false
	}
	if err != nil {
		it.Close()
		it.Err = err
		return false
	}
	return true
}

func (it *Iterator) Close() {
	if it.stream == nil {
		return
	}
	it.Err = it.stream.CloseSend()
	it.stream = nil
}

package codes

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	codes "google.golang.org/grpc/codes"
)

type Code codes.Code

const (
	// Accepted device accepts request and action will be proceed in future.
	Accepted Code = iota + 4096
	// MethodNotAllowed device refuse call the method.
	MethodNotAllowed
	// Created success status response code indicates that the request has succeeded and has led to the creation of a resource.
	Created

	// InvalidCode cannot determines result from device code.
	InvalidCode Code = iota + (2 * 4096)
)

var code2string = map[Code]string{
	Created:          "created",
	MethodNotAllowed: "methodNotAllowed",
	Accepted:         "accepted",
}

var code2httpCode = map[Code]int{
	Created:          http.StatusCreated,
	MethodNotAllowed: http.StatusMethodNotAllowed,
	Accepted:         http.StatusAccepted,
}

func (c Code) ToHTTPCode() int {
	v, ok := code2httpCode[c]
	if ok {
		return v
	}
	return runtime.HTTPStatusFromCode(codes.Code(c))
}

func (c Code) String() string {
	v, ok := code2string[c]
	if ok {
		return v
	}
	return codes.Code(c).String()
}

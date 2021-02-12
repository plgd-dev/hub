package pb

import codes "google.golang.org/grpc/codes"

const (
	// Accepted device accepts request and action will be proceed in future.
	Accepted codes.Code = iota + 4096
	// MethodNotAllowed device refuse call the method.
	MethodNotAllowed
	// Created success status response code indicates that the request has succeeded and has led to the creation of a resource.
	Created

	// InvalidCode cannot determines result from device code.
	InvalidCode codes.Code = iota + (2 * 4096)
)

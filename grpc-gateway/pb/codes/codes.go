package pb

import codes "google.golang.org/grpc/codes"

const (
	// Accepted device accepts request and action will be proceed in future.
	Accepted codes.Code = iota + 4096
	// InvalidCode cannot determines result from device code.
	InvalidCode
)

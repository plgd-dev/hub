package http

const (
	API string = "/api/v1"

	// (GRPC + HTTP) GET /api/v1/conditions -> rpc GetConditions
	// (GRPC + HTTP) DELETE /api/v1/conditions -> rpc DeleteConditions
	// (GRPC + HTTP) POST /api/v1/conditions -> rpc CreateCondition
	Conditions = API + "/conditions"

	// (GRPC + HTTP) GET /api/v1/configurations -> rpc GetConfigurations
	// (GRPC + HTTP) DELETE /api/v1/configurations -> rpc DeleteConfigurations
	// (GRPC + HTTP) POST /api/v1/configurations -> rpc CreateConfiguration
	Configurations = API + "/configurations"
)

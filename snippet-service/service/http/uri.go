package http

const (
	IDKey              = "id"
	ConfigurationIDKey = "configurationId"

	VersionQueryKey = "version"

	API string = "/snippet-service/api/v1"

	// GET /api/v1/conditions -> rpc GetConditions
	// DELETE /api/v1/conditions -> rpc DeleteConditions
	// POST /api/v1/conditions -> rpc CreateCondition
	Conditions = API + "/conditions"

	// GET /api/v1/configurations -> rpc GetConfigurations
	// DELETE /api/v1/configurations -> rpc DeleteConfigurations
	// POST /api/v1/configurations -> rpc CreateConfiguration
	Configurations = API + "/configurations"

	// PUT /api/v1/configurations/{id} -> rpc Update Configuration
	// GET /api/v1/configuration/{id}?version=latest -> rpc GetConfigurations + IDFilter{IDFilter_Latest}
	// GET /api/v1/configuration/{id}?version=all -> rpc GetConfigurations + IDFilter{IDFilter_All}
	// GET /api/v1/configuration/{id}?version={version} -> rpc GetConfigurations + IDFilter{IDFilter_Version{version}}
	AliasConfigurations = Configurations + "/{" + IDKey + "}"
)

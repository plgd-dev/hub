package http

const (
	IDKey              = "id"
	ConfigurationIDKey = "configurationId"

	VersionQueryKey = "version"

	API string = "/snippet-service/api/v1"

	// GET /snippet-service/api/v1/conditions -> rpc GetConditions
	// DELETE /snippet-service/api/v1/conditions -> rpc DeleteConditions
	// POST /snippet-service/api/v1/conditions -> rpc CreateCondition
	Conditions = API + "/conditions"

	// PUT /snippet-service/api/v1/conditions/{id} -> rpc Update Condition
	AliasConditions = Conditions + "/{" + IDKey + "}"

	// GET /snippet-service/api/v1/configurations -> rpc GetConfigurations
	// DELETE /snippet-service/api/v1/configurations -> rpc DeleteConfigurations
	// POST /snippet-service/api/v1/configurations -> rpc CreateConfiguration
	Configurations = API + "/configurations"

	// PUT /snippet-service/api/v1/configurations/{id} -> rpc Update Configuration
	// GET /snippet-service/api/v1/configuration/{id}?version=latest -> rpc GetConfigurations + IDFilter{IDFilter_Latest}
	// GET /snippet-service/api/v1/configuration/{id}?version=all -> rpc GetConfigurations + IDFilter{IDFilter_All}
	// GET /snippet-service/api/v1/configuration/{id}?version={version} -> rpc GetConfigurations + IDFilter{IDFilter_Version{version}}
	AliasConfigurations = Configurations + "/{" + IDKey + "}"
)

package pb

const (
	RecordIDKey        = "_id"
	IDKey              = "id"              // must match with Id field tag
	DeviceIDKey        = "deviceId"        // must match with DeviceId field tag
	OwnerKey           = "owner"           // must match with Owner field tag
	LatestKey          = "latest"          // must match with Latest field tag
	VersionKey         = "version"         // must match with Version field tag
	VersionsKey        = "versions"        // must match with Versions field tag
	ResourcesKey       = "resources"       // must match with Resources field tag
	ConfigurationIDKey = "configurationId" // must match with ConfigurationId field tag
	ConditionIDKey     = "conditionId"     // must match with ConditionId field tag
	EnabledKey         = "enabled"         // must match with Enabled field tag
	TimestampKey       = "timestamp"       // must match with Timestamp field tag
	StatusKey          = "status"          // must match with Status field tag
	HrefKey            = "href"            // must match with Href field tag
	ValidUntil         = "validUntil"      // must match with ValidUntil field tag

	DeviceIDFilterKey     = "deviceIdFilter"     // must match with Condition.DeviceIdFilter tag
	ResourceHrefFilterKey = "resourceHrefFilter" // must match with Condition.ResourceHrefFilter tag
	ResourceTypeFilterKey = "resourceTypeFilter" // must match with Condition.ResourceTypeFilter tag

	ConfigurationLinkIDKey      = ConfigurationIDKey + ".id"            // configurationId.id
	ConfigurationLinkVersionKey = ConfigurationIDKey + "." + VersionKey // configurationId.version
	ConditionLinkIDKey          = ConditionIDKey + ".id"                // conditionId.id
	ConditionLinkVersionKey     = ConditionIDKey + "." + VersionKey     // conditionId.version
)

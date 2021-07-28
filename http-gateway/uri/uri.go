package uri

import "strings"

const (
	DeviceIDKey     = "deviceId"
	ResourceHrefKey = "resourceHref"

	ResourceInterfaceQueryKey = "resourceInterface"
	ShadowQueryKey            = "shadow"
	CommandFilterQueryKey     = "commandFilter"
	TypeFilterQueryKey        = "typeFilter"
	StatusFilterQueryKey      = "statusFilter"
	DeviceIdFilterQueryKey    = "deviceIdFilter"
	ResourceIdFilterQueryKey  = "resourceIdFilter"
	AcceptQueryKey            = "accept" // for websocket
	CorrelationIDQueryKey     = "correlationId"
	TimestampFilterQueryKey   = "timestampFilter"

	AliasInterfaceQueryKey        = "interface"
	AliasCommandFilterQueryKey    = "command"
	AliasDeviceIdFilterQueryKey   = "deviceId"
	AliasResourceIdFilterQueryKey = "resourceId"
	AliasTypeFilterQueryKey       = "type"
	AliasStatusFilterQueryKey     = "status"

	CorrelationIDHeaderKey = "Correlation-Id"
	ContentTypeHeaderKey   = "Content-Type"

	ResourcesPathKey       = "resources"
	ResourceLinksPathKey   = "resource-links"
	PendingCommandsPathKey = "pending-commands"
	EventsPathKey          = "events"

	ApplicationProtoJsonContentType = "application/protojson"

	API   string = "/api/v1"
	APIWS string = API + "/ws"

	// ocfcloud configuration
	ClientConfiguration = "/.well-known/ocfcloud-configuration"

	// oauth configuration for ui
	OAuthConfiguration = "/auth_config.json"

	//certificate-authority
	CertificaAuthority     = API + "/certificate-authority"
	CertificaAuthoritySign = CertificaAuthority + "/sign"

	// (GRPC + HTTP) GET /api/v1/devices -> rpc GetDevices
	Devices = API + "/devices"
	//(HTTP ALIAS) GET /api/v1/devices/{deviceId} -> rpc GetDevices + deviceIdFilter
	AliasDevice = Devices + "/{" + DeviceIDKey + "}"

	//(GRPC + HTTP) GET /api/v1/resource-links -> rpc GetResourceLinks
	ResourceLinks = API + "/" + ResourceLinksPathKey
	// (HTTP ALIAS) GET /api/v1/devices/{deviceId}/resource-links
	AliasDeviceResourceLinks = AliasDevice + "/" + ResourceLinksPathKey

	Resources = API + "/" + ResourcesPathKey

	// (GRPC + HTTP) GET /api/v1/devices/devices-metadata
	DevicesMetadata = API + "/devices-metadata"

	// (GRPC + HTTP) GET /api/v1/devices/devices-metadata
	DeviceMetadata = AliasDevice + "/metadata"

	// (HTTP ALIAS) GET /api/v1/devices/{deviceId}/resources/{resourceHref}?shadow=false -> rpc RetrieveResourceFromDevice
	// (GRPC + HTTP) PUT /api/v1/devices/{deviceId}/resources/{resourceHref} -> rpc Update Resource
	AliasDeviceResource = AliasDeviceResources + "/{" + ResourceHrefKey + "}"

	// (GRPC + HTTP) DELETE /api/v1/devices/{deviceId}/resource-links/{resourceHref} -> rpc DeleteResource
	// (GRPC + HTTP) CREATE /api/v1/devices/{deviceId}/resource-links/{resourceHref} -> rpc CreateResource
	DeviceResourceLink = AliasDeviceResourceLinks + "/{" + ResourceHrefKey + "}"

	// (HTTP ALIAS) GET /api/v1/devices/{deviceId}/resources/ -> rpc RetrieveResources + deviceIdFilter
	AliasDeviceResources = AliasDevice + "/" + ResourcesPathKey

	// (GRPC + HTTP) GET /api/v1/pending-commands -> rpc RetrievePendingCommands
	PendingCommands = API + "/" + PendingCommandsPathKey

	// (HTTP ALIAS) GET /api/v1/devices/{deviceId}/pending-commands == rpc RetrievePendingCommands + deviceIdFilter
	AliasDevicePendingCommands = AliasDevice + "/" + PendingCommandsPathKey

	// (HTTP ALIAS) GET /api/v1/devices/{deviceId}/resources/{resourceHref}/pending-commands == rpc RetrievePendingCommands + resourceIdFilter
	AliasResourcePendingCommands = AliasDeviceResource + "/" + PendingCommandsPathKey

	// (GRPC + HTTP) GET /api/v1/events -> rpc GetEvents
	// (GRPC + HTTP) GET /api/v1/events?timestampFilter={timestamp} -> rpc GetEvents + timestampFilter
	Events = API + "/" + EventsPathKey

	// (HTTP ALIAS) GET /api/v1/devices/{deviceId}/events == rpc GetEvents + deviceIdFilter
	// (HTTP ALIAS) GET /api/v1/devices/{deviceId}/events?timestampFilter={timestamp} == rpc GetEvents + deviceIdFilter + timestampFilter
	AliasDeviceEvents = AliasDevice + "/" + EventsPathKey

	// (HTTP ALIAS) GET /api/v1/devices/{deviceId}/resources/{resourceHref}/events == rpc GetEvents + resourceIdFilter
	// (HTTP ALIAS) GET /api/v1/devices/{deviceId}/resources/{resourceHref}/events?timestampFilter={timestamp} == rpc GetEvents + resourceIdFilter + timestampFilter
	AliasResourceEvents = AliasDeviceResource + "/" + EventsPathKey
)

var QueryCaseInsensitive = map[string]string{
	strings.ToLower(AliasInterfaceQueryKey):        ResourceInterfaceQueryKey,
	strings.ToLower(CommandFilterQueryKey):         CommandFilterQueryKey,
	strings.ToLower(DeviceIdFilterQueryKey):        DeviceIdFilterQueryKey,
	strings.ToLower(ResourceIdFilterQueryKey):      ResourceIdFilterQueryKey,
	strings.ToLower(ResourceInterfaceQueryKey):     ResourceInterfaceQueryKey,
	strings.ToLower(ShadowQueryKey):                ShadowQueryKey,
	strings.ToLower(TypeFilterQueryKey):            TypeFilterQueryKey,
	strings.ToLower(AliasCommandFilterQueryKey):    CommandFilterQueryKey,
	strings.ToLower(AliasDeviceIdFilterQueryKey):   DeviceIdFilterQueryKey,
	strings.ToLower(AliasResourceIdFilterQueryKey): ResourceIdFilterQueryKey,
	strings.ToLower(AliasTypeFilterQueryKey):       TypeFilterQueryKey,
	strings.ToLower(AcceptQueryKey):                AcceptQueryKey,
	strings.ToLower(StatusFilterQueryKey):          StatusFilterQueryKey,
	strings.ToLower(AliasStatusFilterQueryKey):     StatusFilterQueryKey,
	strings.ToLower(CorrelationIDQueryKey):         CorrelationIDQueryKey,
	strings.ToLower(TimestampFilterQueryKey):       TimestampFilterQueryKey,
}

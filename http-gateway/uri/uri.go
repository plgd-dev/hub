package uri

import "strings"

const (
	DeviceIDKey     = "deviceId"
	ResourceHrefKey = "resourceHref"

	ResourceInterfaceQueryKey = "resourceInterface"
	ShadowQueryKey            = "shadow"
	CommandsFilterQueryKey    = "commandsFilter"
	TypeFilterQueryKey        = "typeFilter"
	StatusFilterQueryKey      = "statusFilter"
	DeviceIdsFilterQueryKey   = "deviceIdsFilter"
	ResourceIdsFilterQueryKey = "resourceIdsFilter"
	AcceptQueryKey            = "accept" // for websocket

	AliasInterfaceQueryKey         = "interface"
	AliasCommandsFilterQueryKey    = "commands"
	AliasDeviceIdsFilterQueryKey   = "deviceIds"
	AliasResourceIdsFilterQueryKey = "resourceIds"
	AliasTypeFilterQueryKey        = "type"
	AliasStatusFilterQueryKey      = "status"

	CorrelationIDHeader = "CorrelationID"

	ResourcesPathKey       = "resources"
	PendingCommandsPathKey = "pending-commands"

	ApplicationJsonPBContentType = "application/jsonpb"

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
	ResourceLinks = API + "/resource-links"
	// (HTTP ALIAS) GET /api/v1/devices/{deviceId}/resource-links
	AliasDeviceResourceLinks = AliasDevice + "/resource-links"

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
	DeviceResourceLink = AliasDevice + "/resource-links/{" + ResourceHrefKey + "}"

	// (HTTP ALIAS) GET /api/v1/devices/{deviceId}/resources/ -> rpc RetrieveResources + deviceIdFilter
	AliasDeviceResources = AliasDevice + "/" + ResourcesPathKey

	// (GRPC + HTTP) GET /api/v1/pending-commands -> rpc RetrievePendingCommands
	PendingCommands = API + "/" + PendingCommandsPathKey

	// (HTTP ALIAS) GET /api/v1/devices/{deviceId}/pending-commands == rpc RetrievePendingCommands + deviceIdFilter
	AliasDevicePendingCommands = AliasDevice + "/" + PendingCommandsPathKey

	// (HTTP ALIAS) GET /api/v1/devices/{deviceId}/resources/{resourceHref}/pending-commands == rpc RetrievePendingCommands + resourceIdFilter
	AliasResourcePendingCommands = AliasDeviceResource + "/" + PendingCommandsPathKey
)

var QueryCaseInsensitive = map[string]string{
	strings.ToLower(AliasInterfaceQueryKey):         ResourceInterfaceQueryKey,
	strings.ToLower(CommandsFilterQueryKey):         CommandsFilterQueryKey,
	strings.ToLower(DeviceIdsFilterQueryKey):        DeviceIdsFilterQueryKey,
	strings.ToLower(ResourceIdsFilterQueryKey):      ResourceIdsFilterQueryKey,
	strings.ToLower(ResourceInterfaceQueryKey):      ResourceInterfaceQueryKey,
	strings.ToLower(ShadowQueryKey):                 ShadowQueryKey,
	strings.ToLower(TypeFilterQueryKey):             TypeFilterQueryKey,
	strings.ToLower(AliasCommandsFilterQueryKey):    CommandsFilterQueryKey,
	strings.ToLower(AliasDeviceIdsFilterQueryKey):   DeviceIdsFilterQueryKey,
	strings.ToLower(AliasResourceIdsFilterQueryKey): ResourceIdsFilterQueryKey,
	strings.ToLower(AliasTypeFilterQueryKey):        TypeFilterQueryKey,
	strings.ToLower(AcceptQueryKey):                 AcceptQueryKey,
	strings.ToLower(StatusFilterQueryKey):           StatusFilterQueryKey,
	strings.ToLower(AliasStatusFilterQueryKey):      StatusFilterQueryKey,
}

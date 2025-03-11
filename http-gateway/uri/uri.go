package uri

import "strings"

const (
	DeviceIDKey      = "deviceId"
	ResourceHrefKey  = "resourceHref"
	CorrelationIDKey = "correlationId"

	ResourceInterfaceQueryKey      = "resourceInterface"
	TwinQueryKey                   = "twin"
	CommandFilterQueryKey          = "commandFilter"
	TypeFilterQueryKey             = "typeFilter"
	StatusFilterQueryKey           = "statusFilter"
	DeviceIdFilterQueryKey         = "deviceIdFilter"
	TimeToLiveQueryKey             = "timeToLive"
	ResourceIdFilterQueryKey       = "resourceIdFilter"
	HttpResourceIdFilterQueryKey   = "httpResourceIdFilter"
	AcceptQueryKey                 = "accept" // for websocket
	CorrelationIDQueryKey          = "correlationId"
	TimestampFilterQueryKey        = "timestampFilter"
	CorrelationIdFilterQueryKey    = "correlationIdFilter"
	ETagQueryKey                   = "etag"
	OnlyContentQueryKey            = "onlyContent"
	IncludeHiddenResourcesQueryKey = "includeHiddenResources"
	ForceQueryKey                  = "force"
	IssuerIDKey                    = "issuerId"

	AliasInterfaceQueryKey        = "interface"
	AliasCommandFilterQueryKey    = "command"
	AliasDeviceIdFilterQueryKey   = "deviceId"
	AliasResourceIdFilterQueryKey = "resourceId"
	AliasTypeFilterQueryKey       = "type"
	AliasStatusFilterQueryKey     = "status"

	DevicesPathKey                = "devices"
	ResourcesPathKey              = "resources"
	ResourceLinksPathKey          = "resource-links"
	PendingCommandsPathKey        = "pending-commands"
	PendingMetadataUpdatesPathKey = "pending-metadata-updates"
	EventsPathKey                 = "events"
	ThingsPathKey                 = "things"

	API               string = "/api/v1"
	APIWS             string = API + "/ws"
	SubscribeToEvents string = APIWS + "/" + EventsPathKey

	// hub configuration
	HubConfiguration = "/.well-known/hub-configuration"
	Configuration    = "/.well-known/configuration"

	// web configuration for ui
	WebConfiguration = "/web_configuration.json"

	// list devices with thing descriptions
	Things = API + "/" + ThingsPathKey
	// (HTTP ALIAS) GET /api/v1/things/{deviceId}
	AliasDeviceThing = Things + "/{" + DeviceIDKey + "}"

	// (GRPC + HTTP) GET /api/v1/devices -> rpc GetDevices
	// (GRPC + HTTP) DELETE /api/v1/devices -> rpc DeleteDevices
	Devices = API + "/devices"
	// (HTTP ALIAS) GET /api/v1/devices/{deviceId} -> rpc GetDevices + deviceIdFilter
	// (HTTP ALIAS) DELETE /api/v1/devices/{deviceId} -> rpc DeleteDevices + deviceIdFilter
	AliasDevice = Devices + "/{" + DeviceIDKey + "}"

	// (GRPC + HTTP) GET /api/v1/resource-links -> rpc GetResourceLinks
	ResourceLinks = API + "/" + ResourceLinksPathKey
	// (HTTP ALIAS) GET /api/v1/devices/{deviceId}/resource-links
	AliasDeviceResourceLinks = AliasDevice + "/" + ResourceLinksPathKey

	Resources = API + "/" + ResourcesPathKey

	// (GRPC + HTTP) GET /api/v1/devices/devices-metadata
	DevicesMetadata = API + "/devices-metadata"

	// (GRPC + HTTP) GET /api/v1/devices/devices-metadata
	DeviceMetadata = AliasDevice + "/metadata"

	// (HTTP ALIAS) GET /api/v1/devices/{deviceId}/resources/{resourceHref}?twin=false -> rpc RetrieveResourceFromDevice
	// (GRPC + HTTP) PUT /api/v1/devices/{deviceId}/resources/{resourceHref} -> rpc Update Resource
	AliasDeviceResource = AliasDeviceResources + "/{" + ResourceHrefKey + "}"

	// (GRPC + HTTP) DELETE /api/v1/devices/{deviceId}/resource-links/{resourceHref} -> rpc DeleteResource
	// (GRPC + HTTP) CREATE /api/v1/devices/{deviceId}/resource-links/{resourceHref} -> rpc CreateResource
	DeviceResourceLink = AliasDeviceResourceLinks + "/{" + ResourceHrefKey + "}"

	// (HTTP ALIAS) GET /api/v1/devices/{deviceId}/resources/ -> rpc RetrieveResources + deviceIdFilter
	AliasDeviceResources = AliasDevice + "/" + ResourcesPathKey

	// (GRPC + HTTP) GET /api/v1/pending-commands -> rpc RetrievePendingCommands
	// (GRPC + HTTP) DELETTE /api/v1/pending-commands -> rpc CancelPendingCommands
	PendingCommands = API + "/" + PendingCommandsPathKey

	// (HTTP ALIAS) GET /api/v1/devices/{deviceId}/pending-commands == rpc RetrievePendingCommands + deviceIdFilter
	AliasDevicePendingCommands = AliasDevice + "/" + PendingCommandsPathKey

	// (GRPC + HTTP) GET /api/v1/devices/{device_id}/pending-metadata-updates == rpc CancelPendingMetadataUpdates
	// (GRPC + HTTP) DELETE /api/v1/devices/{deviceId}/pending-metadata-updates == rpc CancelPendingMetadataUpdates
	AliasDevicePendingMetadataUpdates = AliasDevice + "/" + PendingMetadataUpdatesPathKey
	// (GRPC + HTTP) DELETE /api/v1/devices/{deviceId}/pending-metadata-updates/{correlationId} == rpc CancelPendingMetadataUpdates + correlationIdFilter
	AliasDevicePendingMetadataUpdate = AliasDevice + "/" + PendingMetadataUpdatesPathKey + "/" + "{" + CorrelationIDKey + "}"

	// (HTTP ALIAS) GET /api/v1/devices/{deviceId}/resources/{resourceHref}/pending-commands == rpc RetrievePendingCommands + resourceIdFilter
	// (HTTP ALIAS) DELETE /api/v1/devices/{deviceId}/resources/{resourceHref}/pending-commands == rpc CancelPendingCommands + deviceIdFilter
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
	strings.ToLower(AliasInterfaceQueryKey):         ResourceInterfaceQueryKey,
	strings.ToLower(CommandFilterQueryKey):          CommandFilterQueryKey,
	strings.ToLower(DeviceIdFilterQueryKey):         DeviceIdFilterQueryKey,
	strings.ToLower(ResourceIdFilterQueryKey):       HttpResourceIdFilterQueryKey,
	strings.ToLower(ResourceInterfaceQueryKey):      ResourceInterfaceQueryKey,
	strings.ToLower(TwinQueryKey):                   TwinQueryKey,
	strings.ToLower(TypeFilterQueryKey):             TypeFilterQueryKey,
	strings.ToLower(AliasCommandFilterQueryKey):     CommandFilterQueryKey,
	strings.ToLower(AliasDeviceIdFilterQueryKey):    DeviceIdFilterQueryKey,
	strings.ToLower(AliasResourceIdFilterQueryKey):  HttpResourceIdFilterQueryKey,
	strings.ToLower(AliasTypeFilterQueryKey):        TypeFilterQueryKey,
	strings.ToLower(AcceptQueryKey):                 AcceptQueryKey,
	strings.ToLower(StatusFilterQueryKey):           StatusFilterQueryKey,
	strings.ToLower(AliasStatusFilterQueryKey):      StatusFilterQueryKey,
	strings.ToLower(CorrelationIDQueryKey):          CorrelationIDQueryKey,
	strings.ToLower(TimestampFilterQueryKey):        TimestampFilterQueryKey,
	strings.ToLower(TimeToLiveQueryKey):             TimeToLiveQueryKey,
	strings.ToLower(CorrelationIdFilterQueryKey):    CorrelationIdFilterQueryKey,
	strings.ToLower(OnlyContentQueryKey):            OnlyContentQueryKey,
	strings.ToLower(IncludeHiddenResourcesQueryKey): IncludeHiddenResourcesQueryKey,
	strings.ToLower(ForceQueryKey):                  ForceQueryKey,
}

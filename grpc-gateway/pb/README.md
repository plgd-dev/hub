# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [grpc-gateway/pb/cancelCommands.proto](#grpc-gateway_pb_cancelCommands-proto)
    - [CancelPendingCommandsRequest](#grpcgateway-pb-CancelPendingCommandsRequest)
    - [CancelPendingCommandsResponse](#grpcgateway-pb-CancelPendingCommandsResponse)
    - [CancelPendingMetadataUpdatesRequest](#grpcgateway-pb-CancelPendingMetadataUpdatesRequest)
  
- [grpc-gateway/pb/devices.proto](#grpc-gateway_pb_devices-proto)
    - [Content](#grpcgateway-pb-Content)
    - [CreateResourceRequest](#grpcgateway-pb-CreateResourceRequest)
    - [CreateResourceResponse](#grpcgateway-pb-CreateResourceResponse)
    - [DeleteDevicesRequest](#grpcgateway-pb-DeleteDevicesRequest)
    - [DeleteDevicesResponse](#grpcgateway-pb-DeleteDevicesResponse)
    - [DeleteResourceRequest](#grpcgateway-pb-DeleteResourceRequest)
    - [DeleteResourceResponse](#grpcgateway-pb-DeleteResourceResponse)
    - [Device](#grpcgateway-pb-Device)
    - [Device.Metadata](#grpcgateway-pb-Device-Metadata)
    - [Event](#grpcgateway-pb-Event)
    - [Event.DeviceRegistered](#grpcgateway-pb-Event-DeviceRegistered)
    - [Event.DeviceRegistered.OpenTelemetryCarrierEntry](#grpcgateway-pb-Event-DeviceRegistered-OpenTelemetryCarrierEntry)
    - [Event.DeviceUnregistered](#grpcgateway-pb-Event-DeviceUnregistered)
    - [Event.DeviceUnregistered.OpenTelemetryCarrierEntry](#grpcgateway-pb-Event-DeviceUnregistered-OpenTelemetryCarrierEntry)
    - [Event.OperationProcessed](#grpcgateway-pb-Event-OperationProcessed)
    - [Event.OperationProcessed.ErrorStatus](#grpcgateway-pb-Event-OperationProcessed-ErrorStatus)
    - [Event.SubscriptionCanceled](#grpcgateway-pb-Event-SubscriptionCanceled)
    - [GetDevicesRequest](#grpcgateway-pb-GetDevicesRequest)
    - [GetResourceFromDeviceRequest](#grpcgateway-pb-GetResourceFromDeviceRequest)
    - [GetResourceFromDeviceResponse](#grpcgateway-pb-GetResourceFromDeviceResponse)
    - [GetResourceLinksRequest](#grpcgateway-pb-GetResourceLinksRequest)
    - [GetResourcesRequest](#grpcgateway-pb-GetResourcesRequest)
    - [LocalizedString](#grpcgateway-pb-LocalizedString)
    - [Resource](#grpcgateway-pb-Resource)
    - [ResourceIdFilter](#grpcgateway-pb-ResourceIdFilter)
    - [SubscribeToEvents](#grpcgateway-pb-SubscribeToEvents)
    - [SubscribeToEvents.CancelSubscription](#grpcgateway-pb-SubscribeToEvents-CancelSubscription)
    - [SubscribeToEvents.CreateSubscription](#grpcgateway-pb-SubscribeToEvents-CreateSubscription)
    - [UpdateResourceRequest](#grpcgateway-pb-UpdateResourceRequest)
    - [UpdateResourceResponse](#grpcgateway-pb-UpdateResourceResponse)
  
    - [Device.OwnershipStatus](#grpcgateway-pb-Device-OwnershipStatus)
    - [Event.OperationProcessed.ErrorStatus.Code](#grpcgateway-pb-Event-OperationProcessed-ErrorStatus-Code)
    - [GetDevicesRequest.Status](#grpcgateway-pb-GetDevicesRequest-Status)
    - [SubscribeToEvents.CreateSubscription.Event](#grpcgateway-pb-SubscribeToEvents-CreateSubscription-Event)
  
- [grpc-gateway/pb/events.proto](#grpc-gateway_pb_events-proto)
    - [GetEventsRequest](#grpcgateway-pb-GetEventsRequest)
    - [GetEventsResponse](#grpcgateway-pb-GetEventsResponse)
  
- [grpc-gateway/pb/getDevicesMetadata.proto](#grpc-gateway_pb_getDevicesMetadata-proto)
    - [GetDevicesMetadataRequest](#grpcgateway-pb-GetDevicesMetadataRequest)
  
- [grpc-gateway/pb/getPendingCommands.proto](#grpc-gateway_pb_getPendingCommands-proto)
    - [GetPendingCommandsRequest](#grpcgateway-pb-GetPendingCommandsRequest)
    - [PendingCommand](#grpcgateway-pb-PendingCommand)
  
    - [GetPendingCommandsRequest.Command](#grpcgateway-pb-GetPendingCommandsRequest-Command)
  
- [grpc-gateway/pb/hubConfiguration.proto](#grpc-gateway_pb_hubConfiguration-proto)
    - [BuildInfo](#grpcgateway-pb-BuildInfo)
    - [HubConfigurationRequest](#grpcgateway-pb-HubConfigurationRequest)
    - [HubConfigurationResponse](#grpcgateway-pb-HubConfigurationResponse)
    - [OAuthClient](#grpcgateway-pb-OAuthClient)
    - [UIConfiguration](#grpcgateway-pb-UIConfiguration)
    - [UIVisibility](#grpcgateway-pb-UIVisibility)
    - [UIVisibility.MainSidebar](#grpcgateway-pb-UIVisibility-MainSidebar)
  
- [grpc-gateway/pb/service.proto](#grpc-gateway_pb_service-proto)
    - [GrpcGateway](#grpcgateway-pb-GrpcGateway)
  
- [grpc-gateway/pb/updateDeviceMetadata.proto](#grpc-gateway_pb_updateDeviceMetadata-proto)
    - [UpdateDeviceMetadataRequest](#grpcgateway-pb-UpdateDeviceMetadataRequest)
    - [UpdateDeviceMetadataResponse](#grpcgateway-pb-UpdateDeviceMetadataResponse)
  
- [resource-aggregate/pb/resources.proto](#resource-aggregate_pb_resources-proto)
    - [AuditContext](#resourceaggregate-pb-AuditContext)
    - [Content](#resourceaggregate-pb-Content)
    - [EndpointInformation](#resourceaggregate-pb-EndpointInformation)
    - [Policy](#resourceaggregate-pb-Policy)
    - [Resource](#resourceaggregate-pb-Resource)
    - [ResourceId](#resourceaggregate-pb-ResourceId)
  
    - [Status](#resourceaggregate-pb-Status)
  
- [resource-aggregate/pb/events.proto](#resource-aggregate_pb_events-proto)
    - [DeviceMetadataSnapshotTaken](#resourceaggregate-pb-DeviceMetadataSnapshotTaken)
    - [DeviceMetadataUpdatePending](#resourceaggregate-pb-DeviceMetadataUpdatePending)
    - [DeviceMetadataUpdatePending.OpenTelemetryCarrierEntry](#resourceaggregate-pb-DeviceMetadataUpdatePending-OpenTelemetryCarrierEntry)
    - [DeviceMetadataUpdated](#resourceaggregate-pb-DeviceMetadataUpdated)
    - [DeviceMetadataUpdated.OpenTelemetryCarrierEntry](#resourceaggregate-pb-DeviceMetadataUpdated-OpenTelemetryCarrierEntry)
    - [EventMetadata](#resourceaggregate-pb-EventMetadata)
    - [ResourceChanged](#resourceaggregate-pb-ResourceChanged)
    - [ResourceChanged.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ResourceChanged-OpenTelemetryCarrierEntry)
    - [ResourceCreatePending](#resourceaggregate-pb-ResourceCreatePending)
    - [ResourceCreatePending.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ResourceCreatePending-OpenTelemetryCarrierEntry)
    - [ResourceCreated](#resourceaggregate-pb-ResourceCreated)
    - [ResourceCreated.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ResourceCreated-OpenTelemetryCarrierEntry)
    - [ResourceDeletePending](#resourceaggregate-pb-ResourceDeletePending)
    - [ResourceDeletePending.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ResourceDeletePending-OpenTelemetryCarrierEntry)
    - [ResourceDeleted](#resourceaggregate-pb-ResourceDeleted)
    - [ResourceDeleted.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ResourceDeleted-OpenTelemetryCarrierEntry)
    - [ResourceLinksPublished](#resourceaggregate-pb-ResourceLinksPublished)
    - [ResourceLinksPublished.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ResourceLinksPublished-OpenTelemetryCarrierEntry)
    - [ResourceLinksSnapshotTaken](#resourceaggregate-pb-ResourceLinksSnapshotTaken)
    - [ResourceLinksSnapshotTaken.ResourcesEntry](#resourceaggregate-pb-ResourceLinksSnapshotTaken-ResourcesEntry)
    - [ResourceLinksUnpublished](#resourceaggregate-pb-ResourceLinksUnpublished)
    - [ResourceLinksUnpublished.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ResourceLinksUnpublished-OpenTelemetryCarrierEntry)
    - [ResourceRetrievePending](#resourceaggregate-pb-ResourceRetrievePending)
    - [ResourceRetrievePending.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ResourceRetrievePending-OpenTelemetryCarrierEntry)
    - [ResourceRetrieved](#resourceaggregate-pb-ResourceRetrieved)
    - [ResourceRetrieved.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ResourceRetrieved-OpenTelemetryCarrierEntry)
    - [ResourceStateSnapshotTaken](#resourceaggregate-pb-ResourceStateSnapshotTaken)
    - [ResourceUpdatePending](#resourceaggregate-pb-ResourceUpdatePending)
    - [ResourceUpdatePending.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ResourceUpdatePending-OpenTelemetryCarrierEntry)
    - [ResourceUpdated](#resourceaggregate-pb-ResourceUpdated)
    - [ResourceUpdated.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ResourceUpdated-OpenTelemetryCarrierEntry)
    - [ServiceMetadataSnapshotTaken](#resourceaggregate-pb-ServiceMetadataSnapshotTaken)
    - [ServiceMetadataUpdated](#resourceaggregate-pb-ServiceMetadataUpdated)
    - [ServiceMetadataUpdated.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ServiceMetadataUpdated-OpenTelemetryCarrierEntry)
    - [ServicesHeartbeat](#resourceaggregate-pb-ServicesHeartbeat)
    - [ServicesHeartbeat.Heartbeat](#resourceaggregate-pb-ServicesHeartbeat-Heartbeat)
  
- [Scalar Value Types](#scalar-value-types)



<a name="grpc-gateway_pb_cancelCommands-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## grpc-gateway/pb/cancelCommands.proto



<a name="grpcgateway-pb-CancelPendingCommandsRequest"></a>

### CancelPendingCommandsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_id | [resourceaggregate.pb.ResourceId](#resourceaggregate-pb-ResourceId) |  |  |
| correlation_id_filter | [string](#string) | repeated | empty array means all. |






<a name="grpcgateway-pb-CancelPendingCommandsResponse"></a>

### CancelPendingCommandsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| correlation_ids | [string](#string) | repeated |  |






<a name="grpcgateway-pb-CancelPendingMetadataUpdatesRequest"></a>

### CancelPendingMetadataUpdatesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| device_id | [string](#string) |  |  |
| correlation_id_filter | [string](#string) | repeated |  |





 

 

 

 



<a name="grpc-gateway_pb_devices-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## grpc-gateway/pb/devices.proto



<a name="grpcgateway-pb-Content"></a>

### Content



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| content_type | [string](#string) |  |  |
| data | [bytes](#bytes) |  |  |






<a name="grpcgateway-pb-CreateResourceRequest"></a>

### CreateResourceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_id | [resourceaggregate.pb.ResourceId](#resourceaggregate-pb-ResourceId) |  |  |
| content | [Content](#grpcgateway-pb-Content) |  |  |
| time_to_live | [int64](#int64) |  | command validity in nanoseconds. 0 means forever and minimal value is 100000000 (100ms). |
| force | [bool](#bool) |  | if true, the command will be executed even if the resource does not exist |
| async | [bool](#bool) |  | if true, the command will finish immediately after pending event is created |






<a name="grpcgateway-pb-CreateResourceResponse"></a>

### CreateResourceResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [resourceaggregate.pb.ResourceCreated](#resourceaggregate-pb-ResourceCreated) |  |  |






<a name="grpcgateway-pb-DeleteDevicesRequest"></a>

### DeleteDevicesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| device_id_filter | [string](#string) | repeated |  |






<a name="grpcgateway-pb-DeleteDevicesResponse"></a>

### DeleteDevicesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| device_ids | [string](#string) | repeated |  |






<a name="grpcgateway-pb-DeleteResourceRequest"></a>

### DeleteResourceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_id | [resourceaggregate.pb.ResourceId](#resourceaggregate-pb-ResourceId) |  |  |
| time_to_live | [int64](#int64) |  | command validity in nanoseconds. 0 means forever and minimal value is 100000000 (100ms). |
| resource_interface | [string](#string) |  |  |
| force | [bool](#bool) |  | if true, the command will be executed even if the resource does not exist |
| async | [bool](#bool) |  | if true, the command will finish immediately after pending event is created |






<a name="grpcgateway-pb-DeleteResourceResponse"></a>

### DeleteResourceResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [resourceaggregate.pb.ResourceDeleted](#resourceaggregate-pb-ResourceDeleted) |  |  |






<a name="grpcgateway-pb-Device"></a>

### Device



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| types | [string](#string) | repeated |  |
| name | [string](#string) |  |  |
| metadata | [Device.Metadata](#grpcgateway-pb-Device-Metadata) |  |  |
| manufacturer_name | [LocalizedString](#grpcgateway-pb-LocalizedString) | repeated |  |
| model_number | [string](#string) |  |  |
| interfaces | [string](#string) | repeated |  |
| protocol_independent_id | [string](#string) |  |  |
| data | [resourceaggregate.pb.ResourceChanged](#resourceaggregate-pb-ResourceChanged) |  |  |
| ownership_status | [Device.OwnershipStatus](#grpcgateway-pb-Device-OwnershipStatus) |  | ownership status of the device |
| endpoints | [string](#string) | repeated | endpoints with schemas which are hosted by the device |






<a name="grpcgateway-pb-Device-Metadata"></a>

### Device.Metadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| connection | [resourceaggregate.pb.Connection](#resourceaggregate-pb-Connection) |  |  |
| twin_synchronization | [resourceaggregate.pb.TwinSynchronization](#resourceaggregate-pb-TwinSynchronization) |  |  |
| twin_enabled | [bool](#bool) |  |  |






<a name="grpcgateway-pb-Event"></a>

### Event



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| subscription_id | [string](#string) |  | subscription id provided by grpc |
| correlation_id | [string](#string) |  |  |
| device_registered | [Event.DeviceRegistered](#grpcgateway-pb-Event-DeviceRegistered) |  |  |
| device_unregistered | [Event.DeviceUnregistered](#grpcgateway-pb-Event-DeviceUnregistered) |  |  |
| resource_published | [resourceaggregate.pb.ResourceLinksPublished](#resourceaggregate-pb-ResourceLinksPublished) |  |  |
| resource_unpublished | [resourceaggregate.pb.ResourceLinksUnpublished](#resourceaggregate-pb-ResourceLinksUnpublished) |  |  |
| resource_changed | [resourceaggregate.pb.ResourceChanged](#resourceaggregate-pb-ResourceChanged) |  |  |
| operation_processed | [Event.OperationProcessed](#grpcgateway-pb-Event-OperationProcessed) |  |  |
| subscription_canceled | [Event.SubscriptionCanceled](#grpcgateway-pb-Event-SubscriptionCanceled) |  |  |
| resource_update_pending | [resourceaggregate.pb.ResourceUpdatePending](#resourceaggregate-pb-ResourceUpdatePending) |  |  |
| resource_updated | [resourceaggregate.pb.ResourceUpdated](#resourceaggregate-pb-ResourceUpdated) |  |  |
| resource_retrieve_pending | [resourceaggregate.pb.ResourceRetrievePending](#resourceaggregate-pb-ResourceRetrievePending) |  |  |
| resource_retrieved | [resourceaggregate.pb.ResourceRetrieved](#resourceaggregate-pb-ResourceRetrieved) |  |  |
| resource_delete_pending | [resourceaggregate.pb.ResourceDeletePending](#resourceaggregate-pb-ResourceDeletePending) |  |  |
| resource_deleted | [resourceaggregate.pb.ResourceDeleted](#resourceaggregate-pb-ResourceDeleted) |  |  |
| resource_create_pending | [resourceaggregate.pb.ResourceCreatePending](#resourceaggregate-pb-ResourceCreatePending) |  |  |
| resource_created | [resourceaggregate.pb.ResourceCreated](#resourceaggregate-pb-ResourceCreated) |  |  |
| device_metadata_update_pending | [resourceaggregate.pb.DeviceMetadataUpdatePending](#resourceaggregate-pb-DeviceMetadataUpdatePending) |  |  |
| device_metadata_updated | [resourceaggregate.pb.DeviceMetadataUpdated](#resourceaggregate-pb-DeviceMetadataUpdated) |  |  |






<a name="grpcgateway-pb-Event-DeviceRegistered"></a>

### Event.DeviceRegistered



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| device_ids | [string](#string) | repeated |  |
| event_metadata | [identitystore.pb.EventMetadata](#identitystore-pb-EventMetadata) |  | provides metadata of event |
| open_telemetry_carrier | [Event.DeviceRegistered.OpenTelemetryCarrierEntry](#grpcgateway-pb-Event-DeviceRegistered-OpenTelemetryCarrierEntry) | repeated | Open telemetry data propagated to asynchronous events |






<a name="grpcgateway-pb-Event-DeviceRegistered-OpenTelemetryCarrierEntry"></a>

### Event.DeviceRegistered.OpenTelemetryCarrierEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="grpcgateway-pb-Event-DeviceUnregistered"></a>

### Event.DeviceUnregistered



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| device_ids | [string](#string) | repeated |  |
| event_metadata | [identitystore.pb.EventMetadata](#identitystore-pb-EventMetadata) |  | provides metadata of event |
| open_telemetry_carrier | [Event.DeviceUnregistered.OpenTelemetryCarrierEntry](#grpcgateway-pb-Event-DeviceUnregistered-OpenTelemetryCarrierEntry) | repeated | Open telemetry data propagated to asynchronous events |






<a name="grpcgateway-pb-Event-DeviceUnregistered-OpenTelemetryCarrierEntry"></a>

### Event.DeviceUnregistered.OpenTelemetryCarrierEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="grpcgateway-pb-Event-OperationProcessed"></a>

### Event.OperationProcessed



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| error_status | [Event.OperationProcessed.ErrorStatus](#grpcgateway-pb-Event-OperationProcessed-ErrorStatus) |  |  |






<a name="grpcgateway-pb-Event-OperationProcessed-ErrorStatus"></a>

### Event.OperationProcessed.ErrorStatus



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code | [Event.OperationProcessed.ErrorStatus.Code](#grpcgateway-pb-Event-OperationProcessed-ErrorStatus-Code) |  |  |
| message | [string](#string) |  |  |






<a name="grpcgateway-pb-Event-SubscriptionCanceled"></a>

### Event.SubscriptionCanceled



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| reason | [string](#string) |  |  |






<a name="grpcgateway-pb-GetDevicesRequest"></a>

### GetDevicesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type_filter | [string](#string) | repeated |  |
| status_filter | [GetDevicesRequest.Status](#grpcgateway-pb-GetDevicesRequest-Status) | repeated |  |
| device_id_filter | [string](#string) | repeated |  |






<a name="grpcgateway-pb-GetResourceFromDeviceRequest"></a>

### GetResourceFromDeviceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_id | [resourceaggregate.pb.ResourceId](#resourceaggregate-pb-ResourceId) |  |  |
| resource_interface | [string](#string) |  |  |
| time_to_live | [int64](#int64) |  | command validity in nanoseconds. 0 means forever and minimal value is 100000000 (100ms). |
| etag | [bytes](#bytes) | repeated | optional |






<a name="grpcgateway-pb-GetResourceFromDeviceResponse"></a>

### GetResourceFromDeviceResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [resourceaggregate.pb.ResourceRetrieved](#resourceaggregate-pb-ResourceRetrieved) |  |  |






<a name="grpcgateway-pb-GetResourceLinksRequest"></a>

### GetResourceLinksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type_filter | [string](#string) | repeated |  |
| device_id_filter | [string](#string) | repeated |  |






<a name="grpcgateway-pb-GetResourcesRequest"></a>

### GetResourcesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| http_resource_id_filter | [string](#string) | repeated | **Deprecated.** Format: {deviceID}{href}(?etag=abc), e.g., &#34;ae424c58-e517-4494-6de7-583536c48213/oic/d?etag=abc&#34; |
| device_id_filter | [string](#string) | repeated | Filter devices by deviceID |
| type_filter | [string](#string) | repeated | Filter devices by resource types in the oic/d resource |
| resource_id_filter | [ResourceIdFilter](#grpcgateway-pb-ResourceIdFilter) | repeated | New resource ID filter. For HTTP requests, use it multiple times as a query parameter like &#34;resourceIdFilter={deviceID}{href}(?etag=abc)&#34; |






<a name="grpcgateway-pb-LocalizedString"></a>

### LocalizedString



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| language | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="grpcgateway-pb-Resource"></a>

### Resource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| types | [string](#string) | repeated |  |
| data | [resourceaggregate.pb.ResourceChanged](#resourceaggregate-pb-ResourceChanged) |  |  |






<a name="grpcgateway-pb-ResourceIdFilter"></a>

### ResourceIdFilter



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_id | [resourceaggregate.pb.ResourceId](#resourceaggregate-pb-ResourceId) |  | Filter specific resources |
| etag | [bytes](#bytes) | repeated | Optional; resource_id.{deviceId, href} must not be empty |






<a name="grpcgateway-pb-SubscribeToEvents"></a>

### SubscribeToEvents



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| create_subscription | [SubscribeToEvents.CreateSubscription](#grpcgateway-pb-SubscribeToEvents-CreateSubscription) |  |  |
| cancel_subscription | [SubscribeToEvents.CancelSubscription](#grpcgateway-pb-SubscribeToEvents-CancelSubscription) |  |  |
| correlation_id | [string](#string) |  | for pairing request SubscribeToEvents with Event.OperationProcessed |






<a name="grpcgateway-pb-SubscribeToEvents-CancelSubscription"></a>

### SubscribeToEvents.CancelSubscription



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| subscription_id | [string](#string) |  |  |






<a name="grpcgateway-pb-SubscribeToEvents-CreateSubscription"></a>

### SubscribeToEvents.CreateSubscription
If you want to subscribe to all events, leave the filter unset.
Use the event_filter in conjunction with other filters to filter events by type. If event_filter is set, only events with the specified type will be received.
To filter devices, use the device_id_filter. It follows the format {deviceID[0]&#43;&#34;/&#34;&#43;&#34;*&#34;, deviceID[1]&#43;&#34;/&#34;&#43;&#34;*&#34;, ...}.
To filter resources, use the href_filter. It follows the format {&#34;*&#34;&#43;href[0], &#34;*&#34;&#43;href[1], ...}.
When both device_id_filter and href_filter are set, the href_filter is applied to each device. {deviceID[0]&#43;href[0], ..., deviceID[1]&#43;href[0], ...}.
To filter resources of specific devices, use the resource_id_filter.
You can use either device_id_filter or resource_id_filter or both. In this case, the result is the union of both filters.
Certain filters perform a logical &#34;or&#34; operation among the elements of the filter.
Lead resource type filter applies to resource-level events (RESOURCE_UPDATE_PENDING..RESOURCE_CHANGED) only. For example, if you subscribe to RESOURCE_CHANGED
and RESOURCE_UPDATED with lead_resource_type_filter set to [&#34;oic.wk.d&#34;, &#34;oic.wk.p&#34;], you will receive events only for resources with the lead resource type
&#34;oic.wk.d&#34; or &#34;oic.wk.p&#34;.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| event_filter | [SubscribeToEvents.CreateSubscription.Event](#grpcgateway-pb-SubscribeToEvents-CreateSubscription-Event) | repeated | array of events. eg: [ REGISTERED, UNREGISTERED ] |
| device_id_filter | [string](#string) | repeated | array of format {deviceID}. eg [ &#34;ae424c58-e517-4494-6de7-583536c48213&#34; ] |
| http_resource_id_filter | [string](#string) | repeated | **Deprecated.** array of format {deviceID}{href}. eg [ &#34;ae424c58-e517-4494-6de7-583536c48213/oic/d&#34;, &#34;ae424c58-e517-4494-6de7-583536c48213/oic/p&#34; ] |
| href_filter | [string](#string) | repeated | array of format {href}. eg [ &#34;/oic/d&#34;, &#34;/oic/p&#34; ] |
| resource_id_filter | [ResourceIdFilter](#grpcgateway-pb-ResourceIdFilter) | repeated |  |
| lead_resource_type_filter | [string](#string) | repeated | filter by lead resource type |






<a name="grpcgateway-pb-UpdateResourceRequest"></a>

### UpdateResourceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_id | [resourceaggregate.pb.ResourceId](#resourceaggregate-pb-ResourceId) |  |  |
| resource_interface | [string](#string) |  |  |
| time_to_live | [int64](#int64) |  | command validity in nanoseconds. 0 means forever and minimal value is 100000000 (100ms). |
| content | [Content](#grpcgateway-pb-Content) |  |  |
| force | [bool](#bool) |  | if true, the command will be executed even if the resource does not exist |
| async | [bool](#bool) |  | if true, the command will finish immediately after pending event is created |






<a name="grpcgateway-pb-UpdateResourceResponse"></a>

### UpdateResourceResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [resourceaggregate.pb.ResourceUpdated](#resourceaggregate-pb-ResourceUpdated) |  |  |





 


<a name="grpcgateway-pb-Device-OwnershipStatus"></a>

### Device.OwnershipStatus


| Name | Number | Description |
| ---- | ------ | ----------- |
| UNKNOWN | 0 | cannot determine ownership status |
| UNOWNED | 1 | device is ready to be owned the user |
| OWNED | 2 | device is owned by the user. to determine who own the device you need to get ownership resource /oic/sec/doxm |
| UNSUPPORTED | 3 | set when device is not secured. (iotivity-lite was built without security) |



<a name="grpcgateway-pb-Event-OperationProcessed-ErrorStatus-Code"></a>

### Event.OperationProcessed.ErrorStatus.Code


| Name | Number | Description |
| ---- | ------ | ----------- |
| OK | 0 |  |
| ERROR | 1 |  |
| NOT_FOUND | 2 |  |



<a name="grpcgateway-pb-GetDevicesRequest-Status"></a>

### GetDevicesRequest.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| ONLINE | 0 |  |
| OFFLINE | 1 |  |



<a name="grpcgateway-pb-SubscribeToEvents-CreateSubscription-Event"></a>

### SubscribeToEvents.CreateSubscription.Event


| Name | Number | Description |
| ---- | ------ | ----------- |
| REGISTERED | 0 |  |
| UNREGISTERED | 1 |  |
| DEVICE_METADATA_UPDATED | 4 |  |
| DEVICE_METADATA_UPDATE_PENDING | 5 |  |
| RESOURCE_PUBLISHED | 6 |  |
| RESOURCE_UNPUBLISHED | 7 |  |
| RESOURCE_UPDATE_PENDING | 8 |  |
| RESOURCE_UPDATED | 9 |  |
| RESOURCE_RETRIEVE_PENDING | 10 |  |
| RESOURCE_RETRIEVED | 11 |  |
| RESOURCE_DELETE_PENDING | 12 |  |
| RESOURCE_DELETED | 13 |  |
| RESOURCE_CREATE_PENDING | 14 |  |
| RESOURCE_CREATED | 15 |  |
| RESOURCE_CHANGED | 16 |  |


 

 

 



<a name="grpc-gateway_pb_events-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## grpc-gateway/pb/events.proto



<a name="grpcgateway-pb-GetEventsRequest"></a>

### GetEventsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| device_id_filter | [string](#string) | repeated |  |
| http_resource_id_filter | [string](#string) | repeated | **Deprecated.** format {deviceID}{href}. eg &#34;ae424c58-e517-4494-6de7-583536c48213/oic/d&#34; |
| timestamp_filter | [int64](#int64) |  | filter events with timestamp &gt; than given value |
| resource_id_filter | [ResourceIdFilter](#grpcgateway-pb-ResourceIdFilter) | repeated | New resource ID filter. For HTTP requests, use it multiple times as a query parameter like &#34;resourceIdFilter={deviceID}{href}&#34;. |






<a name="grpcgateway-pb-GetEventsResponse"></a>

### GetEventsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_links_published | [resourceaggregate.pb.ResourceLinksPublished](#resourceaggregate-pb-ResourceLinksPublished) |  |  |
| resource_links_unpublished | [resourceaggregate.pb.ResourceLinksUnpublished](#resourceaggregate-pb-ResourceLinksUnpublished) |  |  |
| resource_links_snapshot_taken | [resourceaggregate.pb.ResourceLinksSnapshotTaken](#resourceaggregate-pb-ResourceLinksSnapshotTaken) |  |  |
| resource_changed | [resourceaggregate.pb.ResourceChanged](#resourceaggregate-pb-ResourceChanged) |  |  |
| resource_update_pending | [resourceaggregate.pb.ResourceUpdatePending](#resourceaggregate-pb-ResourceUpdatePending) |  |  |
| resource_updated | [resourceaggregate.pb.ResourceUpdated](#resourceaggregate-pb-ResourceUpdated) |  |  |
| resource_retrieve_pending | [resourceaggregate.pb.ResourceRetrievePending](#resourceaggregate-pb-ResourceRetrievePending) |  |  |
| resource_retrieved | [resourceaggregate.pb.ResourceRetrieved](#resourceaggregate-pb-ResourceRetrieved) |  |  |
| resource_delete_pending | [resourceaggregate.pb.ResourceDeletePending](#resourceaggregate-pb-ResourceDeletePending) |  |  |
| resource_deleted | [resourceaggregate.pb.ResourceDeleted](#resourceaggregate-pb-ResourceDeleted) |  |  |
| resource_create_pending | [resourceaggregate.pb.ResourceCreatePending](#resourceaggregate-pb-ResourceCreatePending) |  |  |
| resource_created | [resourceaggregate.pb.ResourceCreated](#resourceaggregate-pb-ResourceCreated) |  |  |
| resource_state_snapshot_taken | [resourceaggregate.pb.ResourceStateSnapshotTaken](#resourceaggregate-pb-ResourceStateSnapshotTaken) |  |  |
| device_metadata_update_pending | [resourceaggregate.pb.DeviceMetadataUpdatePending](#resourceaggregate-pb-DeviceMetadataUpdatePending) |  |  |
| device_metadata_updated | [resourceaggregate.pb.DeviceMetadataUpdated](#resourceaggregate-pb-DeviceMetadataUpdated) |  |  |
| device_metadata_snapshot_taken | [resourceaggregate.pb.DeviceMetadataSnapshotTaken](#resourceaggregate-pb-DeviceMetadataSnapshotTaken) |  |  |





 

 

 

 



<a name="grpc-gateway_pb_getDevicesMetadata-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## grpc-gateway/pb/getDevicesMetadata.proto



<a name="grpcgateway-pb-GetDevicesMetadataRequest"></a>

### GetDevicesMetadataRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| device_id_filter | [string](#string) | repeated |  |
| type_filter | [string](#string) | repeated |  |





 

 

 

 



<a name="grpc-gateway_pb_getPendingCommands-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## grpc-gateway/pb/getPendingCommands.proto



<a name="grpcgateway-pb-GetPendingCommandsRequest"></a>

### GetPendingCommandsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| command_filter | [GetPendingCommandsRequest.Command](#grpcgateway-pb-GetPendingCommandsRequest-Command) | repeated |  |
| http_resource_id_filter | [string](#string) | repeated | **Deprecated.**  |
| device_id_filter | [string](#string) | repeated |  |
| type_filter | [string](#string) | repeated |  |
| resource_id_filter | [ResourceIdFilter](#grpcgateway-pb-ResourceIdFilter) | repeated | New resource ID filter. For HTTP requests, use it multiple times as a query parameter like &#34;resourceIdFilter={deviceID}{href}&#34;. |
| include_hidden_resources | [bool](#bool) |  | Get all pending commands for all resources, even if the resource is not published. |






<a name="grpcgateway-pb-PendingCommand"></a>

### PendingCommand



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_create_pending | [resourceaggregate.pb.ResourceCreatePending](#resourceaggregate-pb-ResourceCreatePending) |  |  |
| resource_retrieve_pending | [resourceaggregate.pb.ResourceRetrievePending](#resourceaggregate-pb-ResourceRetrievePending) |  |  |
| resource_update_pending | [resourceaggregate.pb.ResourceUpdatePending](#resourceaggregate-pb-ResourceUpdatePending) |  |  |
| resource_delete_pending | [resourceaggregate.pb.ResourceDeletePending](#resourceaggregate-pb-ResourceDeletePending) |  |  |
| device_metadata_update_pending | [resourceaggregate.pb.DeviceMetadataUpdatePending](#resourceaggregate-pb-DeviceMetadataUpdatePending) |  |  |





 


<a name="grpcgateway-pb-GetPendingCommandsRequest-Command"></a>

### GetPendingCommandsRequest.Command


| Name | Number | Description |
| ---- | ------ | ----------- |
| RESOURCE_CREATE | 0 |  |
| RESOURCE_RETRIEVE | 1 |  |
| RESOURCE_UPDATE | 2 |  |
| RESOURCE_DELETE | 3 |  |
| DEVICE_METADATA_UPDATE | 4 |  |


 

 

 



<a name="grpc-gateway_pb_hubConfiguration-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## grpc-gateway/pb/hubConfiguration.proto



<a name="grpcgateway-pb-BuildInfo"></a>

### BuildInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version | [string](#string) |  | version of the service |
| build_date | [string](#string) |  | build date of the service |
| commit_hash | [string](#string) |  | commit hash of the service |
| commit_date | [string](#string) |  | commit date of the service |
| release_url | [string](#string) |  | release url of the service |






<a name="grpcgateway-pb-HubConfigurationRequest"></a>

### HubConfigurationRequest







<a name="grpcgateway-pb-HubConfigurationResponse"></a>

### HubConfigurationResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| current_time | [int64](#int64) |  | provides a current time of server in nanoseconds. |
| jwt_owner_claim | [string](#string) |  |  |
| jwt_device_id_claim | [string](#string) |  |  |
| id | [string](#string) |  |  |
| coap_gateway | [string](#string) |  |  |
| certificate_authorities | [string](#string) |  |  |
| authority | [string](#string) |  |  |
| default_command_time_to_live | [int64](#int64) |  | exposes default command time to live in nanoseconds for CreateResource, RetrieveResource, UpdateResource, DeleteResource, and UpdateDeviceMetadata commands when it is not set in the request. 0 - means forever. |
| certificate_authority | [string](#string) |  | certificate_authority in format https://host:port |
| http_gateway_address | [string](#string) |  | cfg for UI http-gateway |
| web_oauth_client | [OAuthClient](#grpcgateway-pb-OAuthClient) |  |  |
| device_oauth_client | [OAuthClient](#grpcgateway-pb-OAuthClient) |  |  |
| m2m_oauth_client | [OAuthClient](#grpcgateway-pb-OAuthClient) |  |  |
| ui | [UIConfiguration](#grpcgateway-pb-UIConfiguration) |  |  |
| build_info | [BuildInfo](#grpcgateway-pb-BuildInfo) |  | build info |






<a name="grpcgateway-pb-OAuthClient"></a>

### OAuthClient



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| client_id | [string](#string) |  | @gotags: yaml:&#34;clientID&#34; |
| audience | [string](#string) |  |  |
| scopes | [string](#string) | repeated |  |
| provider_name | [string](#string) |  | @gotags: yaml:&#34;providerName&#34; |
| client_assertion_type | [string](#string) |  | @gotags: yaml:&#34;clientAssertionType&#34; |
| authority | [string](#string) |  |  |
| grant_type | [string](#string) |  | @gotags: yaml:&#34;grantType&#34; |






<a name="grpcgateway-pb-UIConfiguration"></a>

### UIConfiguration
UI configuration


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| visibility | [UIVisibility](#grpcgateway-pb-UIVisibility) |  |  |
| device_provisioning_service | [string](#string) |  | Address to device provisioning service HTTP API in format https://host:port |
| snippet_service | [string](#string) |  | Address to snippet service HTTP API in format https://host:port |






<a name="grpcgateway-pb-UIVisibility"></a>

### UIVisibility
UI visibility configuration
If true - show UI element, if false - hide UI element


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| main_sidebar | [UIVisibility.MainSidebar](#grpcgateway-pb-UIVisibility-MainSidebar) |  | Main sidebar visibility |






<a name="grpcgateway-pb-UIVisibility-MainSidebar"></a>

### UIVisibility.MainSidebar



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| devices | [bool](#bool) |  |  |
| configuration | [bool](#bool) |  |  |
| remote_clients | [bool](#bool) |  |  |
| pending_commands | [bool](#bool) |  |  |
| certificates | [bool](#bool) |  |  |
| device_provisioning | [bool](#bool) |  |  |
| docs | [bool](#bool) |  |  |
| chat_room | [bool](#bool) |  |  |
| dashboard | [bool](#bool) |  |  |
| integrations | [bool](#bool) |  |  |
| device_firmware_update | [bool](#bool) |  |  |
| device_logs | [bool](#bool) |  |  |
| api_tokens | [bool](#bool) |  |  |
| schema_hub | [bool](#bool) |  |  |
| snippet_service | [bool](#bool) |  |  |





 

 

 

 



<a name="grpc-gateway_pb_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## grpc-gateway/pb/service.proto


 

 

 


<a name="grpcgateway-pb-GrpcGateway"></a>

### GrpcGateway


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetDevices | [GetDevicesRequest](#grpcgateway-pb-GetDevicesRequest) | [Device](#grpcgateway-pb-Device) stream | Get all devices |
| DeleteDevices | [DeleteDevicesRequest](#grpcgateway-pb-DeleteDevicesRequest) | [DeleteDevicesResponse](#grpcgateway-pb-DeleteDevicesResponse) | Delete selected devices. |
| GetResourceLinks | [GetResourceLinksRequest](#grpcgateway-pb-GetResourceLinksRequest) | [.resourceaggregate.pb.ResourceLinksPublished](#resourceaggregate-pb-ResourceLinksPublished) stream | Get resource links of devices. |
| GetResourceFromDevice | [GetResourceFromDeviceRequest](#grpcgateway-pb-GetResourceFromDeviceRequest) | [GetResourceFromDeviceResponse](#grpcgateway-pb-GetResourceFromDeviceResponse) | Get resource from the device. |
| GetResources | [GetResourcesRequest](#grpcgateway-pb-GetResourcesRequest) | [Resource](#grpcgateway-pb-Resource) stream | Get resources from the resource shadow. |
| UpdateResource | [UpdateResourceRequest](#grpcgateway-pb-UpdateResourceRequest) | [UpdateResourceResponse](#grpcgateway-pb-UpdateResourceResponse) | Update resource at the device. |
| SubscribeToEvents | [SubscribeToEvents](#grpcgateway-pb-SubscribeToEvents) stream | [Event](#grpcgateway-pb-Event) stream | When the client creates a subscription. Subscription doesn&#39;t guarantee that all events will be sent to the client. The client is responsible for synchronize events. |
| GetHubConfiguration | [HubConfigurationRequest](#grpcgateway-pb-HubConfigurationRequest) | [HubConfigurationResponse](#grpcgateway-pb-HubConfigurationResponse) | Get cloud configuration |
| DeleteResource | [DeleteResourceRequest](#grpcgateway-pb-DeleteResourceRequest) | [DeleteResourceResponse](#grpcgateway-pb-DeleteResourceResponse) | Delete resource at the device. |
| CreateResource | [CreateResourceRequest](#grpcgateway-pb-CreateResourceRequest) | [CreateResourceResponse](#grpcgateway-pb-CreateResourceResponse) | Create resource at the device. |
| UpdateDeviceMetadata | [UpdateDeviceMetadataRequest](#grpcgateway-pb-UpdateDeviceMetadataRequest) | [UpdateDeviceMetadataResponse](#grpcgateway-pb-UpdateDeviceMetadataResponse) | Enables/disables shadow synchronization for device. |
| GetPendingCommands | [GetPendingCommandsRequest](#grpcgateway-pb-GetPendingCommandsRequest) | [PendingCommand](#grpcgateway-pb-PendingCommand) stream | Gets pending commands for devices . |
| CancelPendingCommands | [CancelPendingCommandsRequest](#grpcgateway-pb-CancelPendingCommandsRequest) | [CancelPendingCommandsResponse](#grpcgateway-pb-CancelPendingCommandsResponse) | Cancels resource commands. |
| CancelPendingMetadataUpdates | [CancelPendingMetadataUpdatesRequest](#grpcgateway-pb-CancelPendingMetadataUpdatesRequest) | [CancelPendingCommandsResponse](#grpcgateway-pb-CancelPendingCommandsResponse) | Cancels device metadata updates. |
| GetDevicesMetadata | [GetDevicesMetadataRequest](#grpcgateway-pb-GetDevicesMetadataRequest) | [.resourceaggregate.pb.DeviceMetadataUpdated](#resourceaggregate-pb-DeviceMetadataUpdated) stream | Gets metadata of the devices. Is contains online/offline or shadown synchronization status. |
| GetEvents | [GetEventsRequest](#grpcgateway-pb-GetEventsRequest) | [GetEventsResponse](#grpcgateway-pb-GetEventsResponse) stream | Get events for given combination of device id, resource id and timestamp |

 



<a name="grpc-gateway_pb_updateDeviceMetadata-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## grpc-gateway/pb/updateDeviceMetadata.proto



<a name="grpcgateway-pb-UpdateDeviceMetadataRequest"></a>

### UpdateDeviceMetadataRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| device_id | [string](#string) |  |  |
| twin_enabled | [bool](#bool) |  |  |
| twin_force_synchronization | [bool](#bool) |  | force synchronization IoT hub with the device resources and set twin_enabled to true. Use to address potential synchronization issues and prevent operational discrepancies. |
| time_to_live | [int64](#int64) |  | command validity in nanoseconds. 0 means forever and minimal value is 100000000 (100ms). |






<a name="grpcgateway-pb-UpdateDeviceMetadataResponse"></a>

### UpdateDeviceMetadataResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [resourceaggregate.pb.DeviceMetadataUpdated](#resourceaggregate-pb-DeviceMetadataUpdated) |  |  |





 

 

 

 



<a name="resource-aggregate_pb_resources-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## resource-aggregate/pb/resources.proto



<a name="resourceaggregate-pb-AuditContext"></a>

### AuditContext



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user_id | [string](#string) |  |  |
| correlation_id | [string](#string) |  |  |
| owner | [string](#string) |  |  |






<a name="resourceaggregate-pb-Content"></a>

### Content



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [bytes](#bytes) |  |  |
| content_type | [string](#string) |  |  |
| coap_content_format | [int32](#int32) |  | -1 means content-format was not provided |






<a name="resourceaggregate-pb-EndpointInformation"></a>

### EndpointInformation



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| endpoint | [string](#string) |  |  |
| priority | [uint64](#uint64) |  |  |






<a name="resourceaggregate-pb-Policy"></a>

### Policy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bit_flags | [int32](#int32) |  |  |






<a name="resourceaggregate-pb-Resource"></a>

### Resource
https://github.com/openconnectivityfoundation/core/blob/master/schemas/oic.links.properties.core-schema.json


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| href | [string](#string) |  |  |
| device_id | [string](#string) |  |  |
| resource_types | [string](#string) | repeated |  |
| interfaces | [string](#string) | repeated |  |
| anchor | [string](#string) |  |  |
| title | [string](#string) |  |  |
| supported_content_types | [string](#string) | repeated |  |
| valid_until | [int64](#int64) |  |  |
| policy | [Policy](#resourceaggregate-pb-Policy) |  |  |
| endpoint_informations | [EndpointInformation](#resourceaggregate-pb-EndpointInformation) | repeated |  |






<a name="resourceaggregate-pb-ResourceId"></a>

### ResourceId



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| device_id | [string](#string) |  |  |
| href | [string](#string) |  |  |





 


<a name="resourceaggregate-pb-Status"></a>

### Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| UNKNOWN | 0 |  |
| OK | 1 |  |
| BAD_REQUEST | 2 |  |
| UNAUTHORIZED | 3 |  |
| FORBIDDEN | 4 |  |
| NOT_FOUND | 5 |  |
| UNAVAILABLE | 6 |  |
| NOT_IMPLEMENTED | 7 |  |
| ACCEPTED | 8 |  |
| ERROR | 9 |  |
| METHOD_NOT_ALLOWED | 10 |  |
| CREATED | 11 |  |
| CANCELED | 12 | Canceled indicates the operation was canceled (typically by the user). |
| NOT_MODIFIED | 13 | Valid indicates the content hasn&#39;t changed. (provided etag in GET request is same as the resource etag). |


 

 

 



<a name="resource-aggregate_pb_events-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## resource-aggregate/pb/events.proto



<a name="resourceaggregate-pb-DeviceMetadataSnapshotTaken"></a>

### DeviceMetadataSnapshotTaken



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| device_id | [string](#string) |  |  |
| device_metadata_updated | [DeviceMetadataUpdated](#resourceaggregate-pb-DeviceMetadataUpdated) |  |  |
| update_pendings | [DeviceMetadataUpdatePending](#resourceaggregate-pb-DeviceMetadataUpdatePending) | repeated | expired events will be removed by creating a new snapshot. |
| event_metadata | [EventMetadata](#resourceaggregate-pb-EventMetadata) |  |  |






<a name="resourceaggregate-pb-DeviceMetadataUpdatePending"></a>

### DeviceMetadataUpdatePending



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| device_id | [string](#string) |  |  |
| twin_enabled | [bool](#bool) |  |  |
| twin_force_synchronization | [bool](#bool) |  |  |
| audit_context | [AuditContext](#resourceaggregate-pb-AuditContext) |  |  |
| event_metadata | [EventMetadata](#resourceaggregate-pb-EventMetadata) |  |  |
| valid_until | [int64](#int64) |  | unix timestamp in nanoseconds (https://golang.org/pkg/time/#Time.UnixNano) when pending event is considered as expired. 0 means forever. |
| open_telemetry_carrier | [DeviceMetadataUpdatePending.OpenTelemetryCarrierEntry](#resourceaggregate-pb-DeviceMetadataUpdatePending-OpenTelemetryCarrierEntry) | repeated | Open telemetry data propagated to asynchronous events |






<a name="resourceaggregate-pb-DeviceMetadataUpdatePending-OpenTelemetryCarrierEntry"></a>

### DeviceMetadataUpdatePending.OpenTelemetryCarrierEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="resourceaggregate-pb-DeviceMetadataUpdated"></a>

### DeviceMetadataUpdated



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| device_id | [string](#string) |  |  |
| connection | [Connection](#resourceaggregate-pb-Connection) |  |  |
| twin_synchronization | [TwinSynchronization](#resourceaggregate-pb-TwinSynchronization) |  |  |
| twin_enabled | [bool](#bool) |  |  |
| audit_context | [AuditContext](#resourceaggregate-pb-AuditContext) |  |  |
| event_metadata | [EventMetadata](#resourceaggregate-pb-EventMetadata) |  |  |
| canceled | [bool](#bool) |  | it true then the command with audit_context.correlation_id was canceled. |
| open_telemetry_carrier | [DeviceMetadataUpdated.OpenTelemetryCarrierEntry](#resourceaggregate-pb-DeviceMetadataUpdated-OpenTelemetryCarrierEntry) | repeated | Open telemetry data propagated to asynchronous events |






<a name="resourceaggregate-pb-DeviceMetadataUpdated-OpenTelemetryCarrierEntry"></a>

### DeviceMetadataUpdated.OpenTelemetryCarrierEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="resourceaggregate-pb-EventMetadata"></a>

### EventMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version | [uint64](#uint64) |  |  |
| timestamp | [int64](#int64) |  |  |
| connection_id | [string](#string) |  |  |
| sequence | [uint64](#uint64) |  | sequence number within the same connection_id; the ResourceChanged event uses the value to skip old events, other event types might not fill the value |
| hub_id | [string](#string) |  | the hub which sent the event |






<a name="resourceaggregate-pb-ResourceChanged"></a>

### ResourceChanged



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_id | [ResourceId](#resourceaggregate-pb-ResourceId) |  |  |
| content | [Content](#resourceaggregate-pb-Content) |  |  |
| status | [Status](#resourceaggregate-pb-Status) |  |  |
| audit_context | [AuditContext](#resourceaggregate-pb-AuditContext) |  |  |
| event_metadata | [EventMetadata](#resourceaggregate-pb-EventMetadata) |  |  |
| etag | [bytes](#bytes) |  | etag of the resource used by twin synchronization |
| resource_types | [string](#string) | repeated |  |
| open_telemetry_carrier | [ResourceChanged.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ResourceChanged-OpenTelemetryCarrierEntry) | repeated | Open telemetry data propagated to asynchronous events |






<a name="resourceaggregate-pb-ResourceChanged-OpenTelemetryCarrierEntry"></a>

### ResourceChanged.OpenTelemetryCarrierEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="resourceaggregate-pb-ResourceCreatePending"></a>

### ResourceCreatePending



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_id | [ResourceId](#resourceaggregate-pb-ResourceId) |  |  |
| content | [Content](#resourceaggregate-pb-Content) |  |  |
| audit_context | [AuditContext](#resourceaggregate-pb-AuditContext) |  |  |
| event_metadata | [EventMetadata](#resourceaggregate-pb-EventMetadata) |  |  |
| valid_until | [int64](#int64) |  | unix timestamp in nanoseconds (https://golang.org/pkg/time/#Time.UnixNano) when pending event is considered as expired. 0 means forever. |
| resource_types | [string](#string) | repeated |  |
| open_telemetry_carrier | [ResourceCreatePending.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ResourceCreatePending-OpenTelemetryCarrierEntry) | repeated | Open telemetry data propagated to asynchronous events |






<a name="resourceaggregate-pb-ResourceCreatePending-OpenTelemetryCarrierEntry"></a>

### ResourceCreatePending.OpenTelemetryCarrierEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="resourceaggregate-pb-ResourceCreated"></a>

### ResourceCreated



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_id | [ResourceId](#resourceaggregate-pb-ResourceId) |  |  |
| status | [Status](#resourceaggregate-pb-Status) |  |  |
| content | [Content](#resourceaggregate-pb-Content) |  |  |
| audit_context | [AuditContext](#resourceaggregate-pb-AuditContext) |  |  |
| event_metadata | [EventMetadata](#resourceaggregate-pb-EventMetadata) |  |  |
| resource_types | [string](#string) | repeated |  |
| open_telemetry_carrier | [ResourceCreated.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ResourceCreated-OpenTelemetryCarrierEntry) | repeated | Open telemetry data propagated to asynchronous events |






<a name="resourceaggregate-pb-ResourceCreated-OpenTelemetryCarrierEntry"></a>

### ResourceCreated.OpenTelemetryCarrierEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="resourceaggregate-pb-ResourceDeletePending"></a>

### ResourceDeletePending



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_id | [ResourceId](#resourceaggregate-pb-ResourceId) |  |  |
| audit_context | [AuditContext](#resourceaggregate-pb-AuditContext) |  |  |
| event_metadata | [EventMetadata](#resourceaggregate-pb-EventMetadata) |  |  |
| valid_until | [int64](#int64) |  | unix timestamp in nanoseconds (https://golang.org/pkg/time/#Time.UnixNano) when pending event is considered as expired. 0 means forever. |
| resource_interface | [string](#string) |  |  |
| resource_types | [string](#string) | repeated |  |
| open_telemetry_carrier | [ResourceDeletePending.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ResourceDeletePending-OpenTelemetryCarrierEntry) | repeated | Open telemetry data propagated to asynchronous events |






<a name="resourceaggregate-pb-ResourceDeletePending-OpenTelemetryCarrierEntry"></a>

### ResourceDeletePending.OpenTelemetryCarrierEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="resourceaggregate-pb-ResourceDeleted"></a>

### ResourceDeleted



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_id | [ResourceId](#resourceaggregate-pb-ResourceId) |  |  |
| status | [Status](#resourceaggregate-pb-Status) |  |  |
| content | [Content](#resourceaggregate-pb-Content) |  |  |
| audit_context | [AuditContext](#resourceaggregate-pb-AuditContext) |  |  |
| event_metadata | [EventMetadata](#resourceaggregate-pb-EventMetadata) |  |  |
| resource_types | [string](#string) | repeated |  |
| open_telemetry_carrier | [ResourceDeleted.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ResourceDeleted-OpenTelemetryCarrierEntry) | repeated | Open telemetry data propagated to asynchronous events |






<a name="resourceaggregate-pb-ResourceDeleted-OpenTelemetryCarrierEntry"></a>

### ResourceDeleted.OpenTelemetryCarrierEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="resourceaggregate-pb-ResourceLinksPublished"></a>

### ResourceLinksPublished
https://github.com/openconnectivityfoundation/cloud-services/blob/master/swagger2.0/oic.wk.rd.swagger.json#L173


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| device_id | [string](#string) |  |  |
| resources | [Resource](#resourceaggregate-pb-Resource) | repeated |  |
| audit_context | [AuditContext](#resourceaggregate-pb-AuditContext) |  |  |
| event_metadata | [EventMetadata](#resourceaggregate-pb-EventMetadata) |  |  |
| open_telemetry_carrier | [ResourceLinksPublished.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ResourceLinksPublished-OpenTelemetryCarrierEntry) | repeated | Open telemetry data propagated to asynchronous events |






<a name="resourceaggregate-pb-ResourceLinksPublished-OpenTelemetryCarrierEntry"></a>

### ResourceLinksPublished.OpenTelemetryCarrierEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="resourceaggregate-pb-ResourceLinksSnapshotTaken"></a>

### ResourceLinksSnapshotTaken



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| device_id | [string](#string) |  |  |
| resources | [ResourceLinksSnapshotTaken.ResourcesEntry](#resourceaggregate-pb-ResourceLinksSnapshotTaken-ResourcesEntry) | repeated |  |
| event_metadata | [EventMetadata](#resourceaggregate-pb-EventMetadata) |  |  |
| audit_context | [AuditContext](#resourceaggregate-pb-AuditContext) |  |  |






<a name="resourceaggregate-pb-ResourceLinksSnapshotTaken-ResourcesEntry"></a>

### ResourceLinksSnapshotTaken.ResourcesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [Resource](#resourceaggregate-pb-Resource) |  |  |






<a name="resourceaggregate-pb-ResourceLinksUnpublished"></a>

### ResourceLinksUnpublished
https://github.com/openconnectivityfoundation/cloud-services/blob/master/swagger2.0/oic.wk.rd.swagger.json #Specification CR needed


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| device_id | [string](#string) |  |  |
| hrefs | [string](#string) | repeated |  |
| audit_context | [AuditContext](#resourceaggregate-pb-AuditContext) |  |  |
| event_metadata | [EventMetadata](#resourceaggregate-pb-EventMetadata) |  |  |
| open_telemetry_carrier | [ResourceLinksUnpublished.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ResourceLinksUnpublished-OpenTelemetryCarrierEntry) | repeated | Open telemetry data propagated to asynchronous events |






<a name="resourceaggregate-pb-ResourceLinksUnpublished-OpenTelemetryCarrierEntry"></a>

### ResourceLinksUnpublished.OpenTelemetryCarrierEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="resourceaggregate-pb-ResourceRetrievePending"></a>

### ResourceRetrievePending



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_id | [ResourceId](#resourceaggregate-pb-ResourceId) |  |  |
| resource_interface | [string](#string) |  |  |
| audit_context | [AuditContext](#resourceaggregate-pb-AuditContext) |  |  |
| event_metadata | [EventMetadata](#resourceaggregate-pb-EventMetadata) |  |  |
| valid_until | [int64](#int64) |  | unix timestamp in nanoseconds (https://golang.org/pkg/time/#Time.UnixNano) when pending event is considered as expired. 0 means forever. |
| etag | [bytes](#bytes) | repeated |  |
| resource_types | [string](#string) | repeated |  |
| open_telemetry_carrier | [ResourceRetrievePending.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ResourceRetrievePending-OpenTelemetryCarrierEntry) | repeated | Open telemetry data propagated to asynchronous events |






<a name="resourceaggregate-pb-ResourceRetrievePending-OpenTelemetryCarrierEntry"></a>

### ResourceRetrievePending.OpenTelemetryCarrierEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="resourceaggregate-pb-ResourceRetrieved"></a>

### ResourceRetrieved



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_id | [ResourceId](#resourceaggregate-pb-ResourceId) |  |  |
| status | [Status](#resourceaggregate-pb-Status) |  |  |
| content | [Content](#resourceaggregate-pb-Content) |  |  |
| audit_context | [AuditContext](#resourceaggregate-pb-AuditContext) |  |  |
| event_metadata | [EventMetadata](#resourceaggregate-pb-EventMetadata) |  |  |
| etag | [bytes](#bytes) |  |  |
| resource_types | [string](#string) | repeated |  |
| open_telemetry_carrier | [ResourceRetrieved.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ResourceRetrieved-OpenTelemetryCarrierEntry) | repeated | Open telemetry data propagated to asynchronous events |






<a name="resourceaggregate-pb-ResourceRetrieved-OpenTelemetryCarrierEntry"></a>

### ResourceRetrieved.OpenTelemetryCarrierEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="resourceaggregate-pb-ResourceStateSnapshotTaken"></a>

### ResourceStateSnapshotTaken



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_id | [ResourceId](#resourceaggregate-pb-ResourceId) |  |  |
| latest_resource_change | [ResourceChanged](#resourceaggregate-pb-ResourceChanged) |  |  |
| resource_create_pendings | [ResourceCreatePending](#resourceaggregate-pb-ResourceCreatePending) | repeated | expired events will be removed by creating a new snapshot. |
| resource_retrieve_pendings | [ResourceRetrievePending](#resourceaggregate-pb-ResourceRetrievePending) | repeated | expired events will be removed by creating a new snapshot. |
| resource_update_pendings | [ResourceUpdatePending](#resourceaggregate-pb-ResourceUpdatePending) | repeated | expired events will be removed by creating a new snapshot. |
| resource_delete_pendings | [ResourceDeletePending](#resourceaggregate-pb-ResourceDeletePending) | repeated | expired events will be removed by creating a new snapshot. |
| audit_context | [AuditContext](#resourceaggregate-pb-AuditContext) |  |  |
| event_metadata | [EventMetadata](#resourceaggregate-pb-EventMetadata) |  |  |
| resource_types | [string](#string) | repeated |  |






<a name="resourceaggregate-pb-ResourceUpdatePending"></a>

### ResourceUpdatePending



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_id | [ResourceId](#resourceaggregate-pb-ResourceId) |  |  |
| resource_interface | [string](#string) |  |  |
| content | [Content](#resourceaggregate-pb-Content) |  |  |
| audit_context | [AuditContext](#resourceaggregate-pb-AuditContext) |  |  |
| event_metadata | [EventMetadata](#resourceaggregate-pb-EventMetadata) |  |  |
| valid_until | [int64](#int64) |  | unix timestamp in nanoseconds (https://golang.org/pkg/time/#Time.UnixNano) when pending event is considered as expired. 0 means forever. |
| resource_types | [string](#string) | repeated |  |
| open_telemetry_carrier | [ResourceUpdatePending.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ResourceUpdatePending-OpenTelemetryCarrierEntry) | repeated | Open telemetry data propagated to asynchronous events |






<a name="resourceaggregate-pb-ResourceUpdatePending-OpenTelemetryCarrierEntry"></a>

### ResourceUpdatePending.OpenTelemetryCarrierEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="resourceaggregate-pb-ResourceUpdated"></a>

### ResourceUpdated



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_id | [ResourceId](#resourceaggregate-pb-ResourceId) |  |  |
| status | [Status](#resourceaggregate-pb-Status) |  |  |
| content | [Content](#resourceaggregate-pb-Content) |  |  |
| audit_context | [AuditContext](#resourceaggregate-pb-AuditContext) |  |  |
| event_metadata | [EventMetadata](#resourceaggregate-pb-EventMetadata) |  |  |
| resource_types | [string](#string) | repeated |  |
| open_telemetry_carrier | [ResourceUpdated.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ResourceUpdated-OpenTelemetryCarrierEntry) | repeated | Open telemetry data propagated to asynchronous events |






<a name="resourceaggregate-pb-ResourceUpdated-OpenTelemetryCarrierEntry"></a>

### ResourceUpdated.OpenTelemetryCarrierEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="resourceaggregate-pb-ServiceMetadataSnapshotTaken"></a>

### ServiceMetadataSnapshotTaken



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| service_metadata_updated | [ServiceMetadataUpdated](#resourceaggregate-pb-ServiceMetadataUpdated) |  |  |
| event_metadata | [EventMetadata](#resourceaggregate-pb-EventMetadata) |  |  |






<a name="resourceaggregate-pb-ServiceMetadataUpdated"></a>

### ServiceMetadataUpdated



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| services_heartbeat | [ServicesHeartbeat](#resourceaggregate-pb-ServicesHeartbeat) |  |  |
| event_metadata | [EventMetadata](#resourceaggregate-pb-EventMetadata) |  |  |
| audit_context | [AuditContext](#resourceaggregate-pb-AuditContext) |  |  |
| open_telemetry_carrier | [ServiceMetadataUpdated.OpenTelemetryCarrierEntry](#resourceaggregate-pb-ServiceMetadataUpdated-OpenTelemetryCarrierEntry) | repeated | Open telemetry data propagated to asynchronous events |






<a name="resourceaggregate-pb-ServiceMetadataUpdated-OpenTelemetryCarrierEntry"></a>

### ServiceMetadataUpdated.OpenTelemetryCarrierEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="resourceaggregate-pb-ServicesHeartbeat"></a>

### ServicesHeartbeat



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| valid | [ServicesHeartbeat.Heartbeat](#resourceaggregate-pb-ServicesHeartbeat-Heartbeat) | repeated | services which heartbeat is still valid |
| expired | [ServicesHeartbeat.Heartbeat](#resourceaggregate-pb-ServicesHeartbeat-Heartbeat) | repeated | services which heartbeat is already expired |






<a name="resourceaggregate-pb-ServicesHeartbeat-Heartbeat"></a>

### ServicesHeartbeat.Heartbeat



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| service_id | [string](#string) |  | generated unique id during start the service |
| valid_until | [int64](#int64) |  | unix timestamp in nanoseconds (https://golang.org/pkg/time/#Time.UnixNano) when service heartbeat is considered as expired. |





 

 

 

 



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |


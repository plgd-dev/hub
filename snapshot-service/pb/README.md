# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [snapshot-service/pb/service.proto](#snapshot-service_pb_service-proto)
    - [SnapshotService](#snapshotservice-pb-SnapshotService)
  
- [snapshot-service/pb/snapshotAssociations.proto](#snapshot-service_pb_snapshotAssociations-proto)
    - [CreateSnapshotAssociationRequest](#snapshotservice-pb-CreateSnapshotAssociationRequest)
    - [CreateSnapshotAssociationResponse](#snapshotservice-pb-CreateSnapshotAssociationResponse)
    - [DeleteSnapshotAssociationsRequest](#snapshotservice-pb-DeleteSnapshotAssociationsRequest)
    - [DeleteSnapshotAssociationsResponse](#snapshotservice-pb-DeleteSnapshotAssociationsResponse)
    - [ResourceFilter](#snapshotservice-pb-ResourceFilter)
  
- [snapshot-service/pb/snapshots.proto](#snapshot-service_pb_snapshots-proto)
    - [ApplySnapshotRequest](#snapshotservice-pb-ApplySnapshotRequest)
    - [ApplySnapshotResponse](#snapshotservice-pb-ApplySnapshotResponse)
    - [CreateSnapshotRequest](#snapshotservice-pb-CreateSnapshotRequest)
    - [CreateSnapshotResponse](#snapshotservice-pb-CreateSnapshotResponse)
    - [DeleteSnapshotsRequest](#snapshotservice-pb-DeleteSnapshotsRequest)
    - [DeleteSnapshotsResponse](#snapshotservice-pb-DeleteSnapshotsResponse)
    - [GetSnapshotsRequest](#snapshotservice-pb-GetSnapshotsRequest)
    - [ResourceSnapshot](#snapshotservice-pb-ResourceSnapshot)
    - [Snapshot](#snapshotservice-pb-Snapshot)
  
- [snapshot-service/pb/snapshotStatuses.proto](#snapshot-service_pb_snapshotStatuses-proto)
    - [DeleteSnapshotStatusesRequest](#snapshotservice-pb-DeleteSnapshotStatusesRequest)
    - [DeleteSnapshotStatusesResponse](#snapshotservice-pb-DeleteSnapshotStatusesResponse)
    - [GetSnapshotStatusesRequest](#snapshotservice-pb-GetSnapshotStatusesRequest)
    - [SnapshotResourceStatus](#snapshotservice-pb-SnapshotResourceStatus)
    - [SnapshotStatus](#snapshotservice-pb-SnapshotStatus)
  
- [Scalar Value Types](#scalar-value-types)



<a name="snapshot-service_pb_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## snapshot-service/pb/service.proto


 

 

 


<a name="snapshotservice-pb-SnapshotService"></a>

### SnapshotService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateSnapshot | [CreateSnapshotRequest](#snapshotservice-pb-CreateSnapshotRequest) | [CreateSnapshotResponse](#snapshotservice-pb-CreateSnapshotResponse) |  |
| GetSnapshots | [GetSnapshotsRequest](#snapshotservice-pb-GetSnapshotsRequest) | [Snapshot](#snapshotservice-pb-Snapshot) stream |  |
| DeleteSnapshots | [DeleteSnapshotsRequest](#snapshotservice-pb-DeleteSnapshotsRequest) | [DeleteSnapshotsResponse](#snapshotservice-pb-DeleteSnapshotsResponse) |  |
| ApplySnapshot | [ApplySnapshotRequest](#snapshotservice-pb-ApplySnapshotRequest) | [ApplySnapshotResponse](#snapshotservice-pb-ApplySnapshotResponse) |  |
| CreateSnapshotAssociation | [CreateSnapshotAssociationRequest](#snapshotservice-pb-CreateSnapshotAssociationRequest) | [CreateSnapshotAssociationResponse](#snapshotservice-pb-CreateSnapshotAssociationResponse) |  |
| DeleteSnapshotAssociations | [DeleteSnapshotAssociationsRequest](#snapshotservice-pb-DeleteSnapshotAssociationsRequest) | [DeleteSnapshotAssociationsResponse](#snapshotservice-pb-DeleteSnapshotAssociationsResponse) |  |
| GetSnapshotStatuses | [GetSnapshotStatusesRequest](#snapshotservice-pb-GetSnapshotStatusesRequest) | [SnapshotStatus](#snapshotservice-pb-SnapshotStatus) stream |  |
| DeleteSnapshotStatuses | [DeleteSnapshotStatusesRequest](#snapshotservice-pb-DeleteSnapshotStatusesRequest) | [DeleteSnapshotStatusesResponse](#snapshotservice-pb-DeleteSnapshotStatusesResponse) |  |

 



<a name="snapshot-service_pb_snapshotAssociations-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## snapshot-service/pb/snapshotAssociations.proto



<a name="snapshotservice-pb-CreateSnapshotAssociationRequest"></a>

### CreateSnapshotAssociationRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| snapshot_id | [string](#string) |  |  |
| resource_filter | [ResourceFilter](#snapshotservice-pb-ResourceFilter) | repeated |  |
| apply_time_to_live | [int64](#int64) |  | in nanoseconds how long will try to apply snapshot to resources which match resource_filter |






<a name="snapshotservice-pb-CreateSnapshotAssociationResponse"></a>

### CreateSnapshotAssociationResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="snapshotservice-pb-DeleteSnapshotAssociationsRequest"></a>

### DeleteSnapshotAssociationsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [string](#string) | repeated |  |






<a name="snapshotservice-pb-DeleteSnapshotAssociationsResponse"></a>

### DeleteSnapshotAssociationsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| count | [int64](#int64) |  |  |






<a name="snapshotservice-pb-ResourceFilter"></a>

### ResourceFilter



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| href | [string](#string) |  | required eg &#34;/oic/d&#34; |
| property_path | [string](#string) |  | optional in format &#34;.di&#34; |
| content | [resourceaggregate.pb.Content](#resourceaggregate-pb-Content) |  | required |





 

 

 

 



<a name="snapshot-service_pb_snapshots-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## snapshot-service/pb/snapshots.proto



<a name="snapshotservice-pb-ApplySnapshotRequest"></a>

### ApplySnapshotRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| device_id_filter | [string](#string) | repeated |  |






<a name="snapshotservice-pb-ApplySnapshotResponse"></a>

### ApplySnapshotResponse







<a name="snapshotservice-pb-CreateSnapshotRequest"></a>

### CreateSnapshotRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resources | [ResourceSnapshot](#snapshotservice-pb-ResourceSnapshot) | repeated |  |






<a name="snapshotservice-pb-CreateSnapshotResponse"></a>

### CreateSnapshotResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="snapshotservice-pb-DeleteSnapshotsRequest"></a>

### DeleteSnapshotsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [string](#string) | repeated |  |






<a name="snapshotservice-pb-DeleteSnapshotsResponse"></a>

### DeleteSnapshotsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| count | [int64](#int64) |  |  |






<a name="snapshotservice-pb-GetSnapshotsRequest"></a>

### GetSnapshotsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [string](#string) | repeated |  |






<a name="snapshotservice-pb-ResourceSnapshot"></a>

### ResourceSnapshot



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| href | [string](#string) |  |  |
| content | [resourceaggregate.pb.Content](#resourceaggregate-pb-Content) |  |  |
| resource_interface | [string](#string) |  |  |
| time_to_live | [int64](#int64) |  |  |






<a name="snapshotservice-pb-Snapshot"></a>

### Snapshot



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resources | [ResourceSnapshot](#snapshotservice-pb-ResourceSnapshot) | repeated |  |





 

 

 

 



<a name="snapshot-service_pb_snapshotStatuses-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## snapshot-service/pb/snapshotStatuses.proto



<a name="snapshotservice-pb-DeleteSnapshotStatusesRequest"></a>

### DeleteSnapshotStatusesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [string](#string) | repeated |  |






<a name="snapshotservice-pb-DeleteSnapshotStatusesResponse"></a>

### DeleteSnapshotStatusesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| count | [int64](#int64) |  |  |






<a name="snapshotservice-pb-GetSnapshotStatusesRequest"></a>

### GetSnapshotStatusesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| snapshot_id_filter | [string](#string) | repeated |  |
| device_id_filter | [string](#string) | repeated |  |






<a name="snapshotservice-pb-SnapshotResourceStatus"></a>

### SnapshotResourceStatus



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| href | [string](#string) |  |  |






<a name="snapshotservice-pb-SnapshotStatus"></a>

### SnapshotStatus



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| version | [string](#string) |  |  |
| device_id | [string](#string) |  |  |
| snapshot_id | [string](#string) |  |  |
| association_snapshot_id | [string](#string) |  |  |
| timestamp_start | [int64](#int64) |  |  |
| timestamp_end | [int64](#int64) |  |  |
| valid_until | [int64](#int64) |  |  |
| resourcesUpdated | [resourceaggregate.pb.ResourceUpdated](#resourceaggregate-pb-ResourceUpdated) | repeated |  |





 

 

 

 



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


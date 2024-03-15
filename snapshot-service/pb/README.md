# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [snapshot-service/pb/service.proto](#snapshot-service_pb_service-proto)
    - [AppliedConfiguration](#snapshotservice-pb-AppliedConfiguration)
    - [AppliedConfiguration.Resource](#snapshotservice-pb-AppliedConfiguration-Resource)
    - [Condition](#snapshotservice-pb-Condition)
    - [Condition.InvokeConfiguration](#snapshotservice-pb-Condition-InvokeConfiguration)
    - [Configuration](#snapshotservice-pb-Configuration)
    - [Configuration.Resource](#snapshotservice-pb-Configuration-Resource)
    - [CreateConditionReponse](#snapshotservice-pb-CreateConditionReponse)
    - [CreateConfigurationReponse](#snapshotservice-pb-CreateConfigurationReponse)
    - [DeleteAppliedConfigurationsRequest](#snapshotservice-pb-DeleteAppliedConfigurationsRequest)
    - [DeleteAppliedConfigurationsResponse](#snapshotservice-pb-DeleteAppliedConfigurationsResponse)
    - [DeleteConditionsRequest](#snapshotservice-pb-DeleteConditionsRequest)
    - [DeleteConditionsResponse](#snapshotservice-pb-DeleteConditionsResponse)
    - [DeleteConfigurationsRequest](#snapshotservice-pb-DeleteConfigurationsRequest)
    - [DeleteConfigurationsResponse](#snapshotservice-pb-DeleteConfigurationsResponse)
    - [GetAppliedConfigurationsRequest](#snapshotservice-pb-GetAppliedConfigurationsRequest)
    - [GetConditionsRequest](#snapshotservice-pb-GetConditionsRequest)
    - [GetConfigurationsRequest](#snapshotservice-pb-GetConfigurationsRequest)
    - [Id](#snapshotservice-pb-Id)
    - [IdFilter](#snapshotservice-pb-IdFilter)
    - [InvokeConfigurationRequest](#snapshotservice-pb-InvokeConfigurationRequest)
    - [ResourceTypes](#snapshotservice-pb-ResourceTypes)
    - [UpdateConditionResponse](#snapshotservice-pb-UpdateConditionResponse)
    - [UpdateConfigurationReponse](#snapshotservice-pb-UpdateConfigurationReponse)
  
    - [AppliedConfiguration.Resource.Status](#snapshotservice-pb-AppliedConfiguration-Resource-Status)
  
    - [SnapshotService](#snapshotservice-pb-SnapshotService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="snapshot-service_pb_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## snapshot-service/pb/service.proto



<a name="snapshotservice-pb-AppliedConfiguration"></a>

### AppliedConfiguration



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| device_id | [string](#string) |  |  |
| configuration_id | [Id](#snapshotservice-pb-Id) |  |  |
| on_demaned | [bool](#bool) |  |  |
| condition_id | [Id](#snapshotservice-pb-Id) |  |  |
| resources | [AppliedConfiguration.Resource](#snapshotservice-pb-AppliedConfiguration-Resource) | repeated |  |






<a name="snapshotservice-pb-AppliedConfiguration-Resource"></a>

### AppliedConfiguration.Resource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_id | [resourceaggregate.pb.ResourceId](#resourceaggregate-pb-ResourceId) |  |  |
| configuration_resources_idx | [uint32](#uint32) |  | index of resource in configuration resources array. For resource types it could be mutliple resources. |
| status | [AppliedConfiguration.Resource.Status](#snapshotservice-pb-AppliedConfiguration-Resource-Status) |  |  |
| timestamp_start | [int64](#int64) |  | when the rule association was applied |
| valid_until | [int64](#int64) |  | how long the command is valid |
| resource_updated | [resourceaggregate.pb.ResourceUpdated](#resourceaggregate-pb-ResourceUpdated) |  |  |






<a name="snapshotservice-pb-Condition"></a>

### Condition



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [Id](#snapshotservice-pb-Id) |  |  |
| name | [string](#string) |  |  |
| enabled | [bool](#bool) |  |  |
| device_id_filter | [string](#string) | repeated |  |
| resource_type_filter | [string](#string) | repeated |  |
| resource_href_filter | [string](#string) | repeated |  |
| yq_expression | [string](#string) |  |  |
| invoke_configuration | [Condition.InvokeConfiguration](#snapshotservice-pb-Condition-InvokeConfiguration) |  |  |






<a name="snapshotservice-pb-Condition-InvokeConfiguration"></a>

### Condition.InvokeConfiguration



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | latest version of cfg |
| keep_updating_on_failure | [bool](#bool) |  |  |
| api_access_token | [string](#string) |  | token used to update resources in the configuration |






<a name="snapshotservice-pb-Configuration"></a>

### Configuration



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [Id](#snapshotservice-pb-Id) |  |  |
| name | [string](#string) |  |  |
| resources | [Configuration.Resource](#snapshotservice-pb-Configuration-Resource) | repeated |  |






<a name="snapshotservice-pb-Configuration-Resource"></a>

### Configuration.Resource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| href | [string](#string) |  |  |
| resource_types | [ResourceTypes](#snapshotservice-pb-ResourceTypes) |  |  |
| content | [resourceaggregate.pb.Content](#resourceaggregate-pb-Content) |  |  |
| resource_interface | [string](#string) |  | optional update interface |
| time_to_live | [int64](#int64) |  | optional update command time to live |






<a name="snapshotservice-pb-CreateConditionReponse"></a>

### CreateConditionReponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [Id](#snapshotservice-pb-Id) |  |  |






<a name="snapshotservice-pb-CreateConfigurationReponse"></a>

### CreateConfigurationReponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [Id](#snapshotservice-pb-Id) |  |  |






<a name="snapshotservice-pb-DeleteAppliedConfigurationsRequest"></a>

### DeleteAppliedConfigurationsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [string](#string) | repeated |  |






<a name="snapshotservice-pb-DeleteAppliedConfigurationsResponse"></a>

### DeleteAppliedConfigurationsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| count | [int64](#int64) |  |  |






<a name="snapshotservice-pb-DeleteConditionsRequest"></a>

### DeleteConditionsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [IdFilter](#snapshotservice-pb-IdFilter) | repeated |  |






<a name="snapshotservice-pb-DeleteConditionsResponse"></a>

### DeleteConditionsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| count | [int64](#int64) |  |  |






<a name="snapshotservice-pb-DeleteConfigurationsRequest"></a>

### DeleteConfigurationsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [IdFilter](#snapshotservice-pb-IdFilter) | repeated |  |






<a name="snapshotservice-pb-DeleteConfigurationsResponse"></a>

### DeleteConfigurationsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| count | [int64](#int64) |  |  |






<a name="snapshotservice-pb-GetAppliedConfigurationsRequest"></a>

### GetAppliedConfigurationsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [string](#string) | repeated |  |
| configuration_id_filter | [IdFilter](#snapshotservice-pb-IdFilter) | repeated |  |
| device_id_filter | [string](#string) | repeated |  |
| condition_id_filter | [string](#string) | repeated |  |






<a name="snapshotservice-pb-GetConditionsRequest"></a>

### GetConditionsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [IdFilter](#snapshotservice-pb-IdFilter) | repeated |  |






<a name="snapshotservice-pb-GetConfigurationsRequest"></a>

### GetConfigurationsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [IdFilter](#snapshotservice-pb-IdFilter) | repeated |  |






<a name="snapshotservice-pb-Id"></a>

### Id



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| version | [uint64](#uint64) |  |  |






<a name="snapshotservice-pb-IdFilter"></a>

### IdFilter



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| value | [uint64](#uint64) |  |  |
| all | [bool](#bool) |  |  |
| max | [bool](#bool) |  |  |






<a name="snapshotservice-pb-InvokeConfigurationRequest"></a>

### InvokeConfigurationRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| device_id_filter | [string](#string) | repeated | at least one must be set |
| force | [bool](#bool) |  |  |
| keep_updating_on_failure | [bool](#bool) |  |  |






<a name="snapshotservice-pb-ResourceTypes"></a>

### ResourceTypes



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| types | [string](#string) | repeated | all types must match resource |
| min | [uint32](#uint32) |  | minimal number of resources that will be updated |






<a name="snapshotservice-pb-UpdateConditionResponse"></a>

### UpdateConditionResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [Id](#snapshotservice-pb-Id) |  | for new version |






<a name="snapshotservice-pb-UpdateConfigurationReponse"></a>

### UpdateConfigurationReponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [Id](#snapshotservice-pb-Id) |  | for new version |





 


<a name="snapshotservice-pb-AppliedConfiguration-Resource-Status"></a>

### AppliedConfiguration.Resource.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| QUEUED | 0 |  |
| INPROGRESS | 1 |  |
| WAITING_FOR_REOURCE | 2 |  |
| DONE | 3 |  |
| TIMEOUT | 4 |  |
| FAIL | 5 |  |


 

 


<a name="snapshotservice-pb-SnapshotService"></a>

### SnapshotService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateCondition | [Condition](#snapshotservice-pb-Condition) | [CreateConditionReponse](#snapshotservice-pb-CreateConditionReponse) |  |
| GetConditions | [GetConditionsRequest](#snapshotservice-pb-GetConditionsRequest) | [Condition](#snapshotservice-pb-Condition) stream |  |
| DeleteConditions | [DeleteConditionsRequest](#snapshotservice-pb-DeleteConditionsRequest) | [DeleteConditionsResponse](#snapshotservice-pb-DeleteConditionsResponse) |  |
| UpdateCondition | [Condition](#snapshotservice-pb-Condition) | [UpdateConditionResponse](#snapshotservice-pb-UpdateConditionResponse) |  |
| CreateConfiguration | [Configuration](#snapshotservice-pb-Configuration) | [CreateConfigurationReponse](#snapshotservice-pb-CreateConfigurationReponse) |  |
| GetConfigurations | [GetConfigurationsRequest](#snapshotservice-pb-GetConfigurationsRequest) | [Configuration](#snapshotservice-pb-Configuration) stream |  |
| DeleteConfigurations | [DeleteConfigurationsRequest](#snapshotservice-pb-DeleteConfigurationsRequest) | [DeleteConfigurationsResponse](#snapshotservice-pb-DeleteConfigurationsResponse) |  |
| UpdateConfiguration | [Configuration](#snapshotservice-pb-Configuration) | [UpdateConfigurationReponse](#snapshotservice-pb-UpdateConfigurationReponse) |  |
| InvokeConfiguration | [InvokeConfigurationRequest](#snapshotservice-pb-InvokeConfigurationRequest) | [AppliedConfiguration](#snapshotservice-pb-AppliedConfiguration) stream | streaming process of update configuration to invoker |
| GetAppliedConfigurations | [GetAppliedConfigurationsRequest](#snapshotservice-pb-GetAppliedConfigurationsRequest) | [AppliedConfiguration](#snapshotservice-pb-AppliedConfiguration) stream |  |
| DeleteAppliedConfigurations | [DeleteAppliedConfigurationsRequest](#snapshotservice-pb-DeleteAppliedConfigurationsRequest) | [DeleteAppliedConfigurationsResponse](#snapshotservice-pb-DeleteAppliedConfigurationsResponse) |  |

 



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


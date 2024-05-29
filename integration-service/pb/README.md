# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [integration-service/pb/service.proto](#integration-service_pb_service-proto)
    - [AppliedConfiguration](#integrationservice-pb-AppliedConfiguration)
    - [AppliedConfiguration.RelationTo](#integrationservice-pb-AppliedConfiguration-RelationTo)
    - [AppliedConfiguration.Resource](#integrationservice-pb-AppliedConfiguration-Resource)
    - [Condition](#integrationservice-pb-Condition)
    - [Configuration](#integrationservice-pb-Configuration)
    - [Configuration.Resource](#integrationservice-pb-Configuration-Resource)
    - [DeleteAppliedConfigurationsRequest](#integrationservice-pb-DeleteAppliedConfigurationsRequest)
    - [DeleteAppliedConfigurationsResponse](#integrationservice-pb-DeleteAppliedConfigurationsResponse)
    - [DeleteConditionsRequest](#integrationservice-pb-DeleteConditionsRequest)
    - [DeleteConditionsResponse](#integrationservice-pb-DeleteConditionsResponse)
    - [DeleteConfigurationsRequest](#integrationservice-pb-DeleteConfigurationsRequest)
    - [DeleteConfigurationsResponse](#integrationservice-pb-DeleteConfigurationsResponse)
    - [GetAppliedConfigurationsRequest](#integrationservice-pb-GetAppliedConfigurationsRequest)
    - [GetConditionsRequest](#integrationservice-pb-GetConditionsRequest)
    - [GetConfigurationsRequest](#integrationservice-pb-GetConfigurationsRequest)
    - [IDFilter](#integrationservice-pb-IDFilter)
    - [InvokeConfigurationRequest](#integrationservice-pb-InvokeConfigurationRequest)
  
    - [AppliedConfiguration.Resource.Status](#integrationservice-pb-AppliedConfiguration-Resource-Status)
  
    - [IntegrationService](#integrationservice-pb-IntegrationService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="integration-service_pb_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## integration-service/pb/service.proto



<a name="integrationservice-pb-AppliedConfiguration"></a>

### AppliedConfiguration
TODO naming


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| device_id | [string](#string) |  |  |
| configuration_id | [AppliedConfiguration.RelationTo](#integrationservice-pb-AppliedConfiguration-RelationTo) |  |  |
| on_demand | [bool](#bool) |  |  |
| condition_id | [AppliedConfiguration.RelationTo](#integrationservice-pb-AppliedConfiguration-RelationTo) |  | TODO Naming |
| resources | [AppliedConfiguration.Resource](#integrationservice-pb-AppliedConfiguration-Resource) | repeated | TODO naming |
| owner | [string](#string) |  |  |






<a name="integrationservice-pb-AppliedConfiguration-RelationTo"></a>

### AppliedConfiguration.RelationTo
TODO naming


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| version | [uint64](#uint64) |  |  |






<a name="integrationservice-pb-AppliedConfiguration-Resource"></a>

### AppliedConfiguration.Resource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_id | [resourceaggregate.pb.ResourceId](#resourceaggregate-pb-ResourceId) |  | TODO Jozo href only? |
| correlation_id | [string](#string) |  | Reused from invoke command or generated. Can be used to retrieve corresponding pending command. |
| status | [AppliedConfiguration.Resource.Status](#integrationservice-pb-AppliedConfiguration-Resource-Status) |  |  |
| resource_updated | [resourceaggregate.pb.ResourceUpdated](#resourceaggregate-pb-ResourceUpdated) |  |  |






<a name="integrationservice-pb-Condition"></a>

### Condition
driven by resource change event


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| version | [uint64](#uint64) |  |  |
| name | [string](#string) |  |  |
| enabled | [bool](#bool) |  |  |
| configuration_id | [string](#string) |  |  |
| device_id_filter | [string](#string) | repeated |  |
| resource_type_filter | [string](#string) | repeated |  |
| resource_href_filter | [string](#string) | repeated |  |
| jq_expression_filter | [string](#string) |  |  |
| api_access_token | [string](#string) |  | token used to update resources in the configuration |
| owner | [string](#string) |  |  |






<a name="integrationservice-pb-Configuration"></a>

### Configuration



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| version | [uint64](#uint64) |  |  |
| name | [string](#string) |  |  |
| resources | [Configuration.Resource](#integrationservice-pb-Configuration-Resource) | repeated |  |
| owner | [string](#string) |  |  |






<a name="integrationservice-pb-Configuration-Resource"></a>

### Configuration.Resource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| href | [string](#string) |  |  |
| content | [resourceaggregate.pb.Content](#resourceaggregate-pb-Content) |  |  |
| time_to_live | [int64](#int64) |  | optional update command time to live, 0 is infinite |






<a name="integrationservice-pb-DeleteAppliedConfigurationsRequest"></a>

### DeleteAppliedConfigurationsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [string](#string) | repeated |  |






<a name="integrationservice-pb-DeleteAppliedConfigurationsResponse"></a>

### DeleteAppliedConfigurationsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| count | [int64](#int64) |  |  |






<a name="integrationservice-pb-DeleteConditionsRequest"></a>

### DeleteConditionsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [IDFilter](#integrationservice-pb-IDFilter) | repeated |  |






<a name="integrationservice-pb-DeleteConditionsResponse"></a>

### DeleteConditionsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| count | [int64](#int64) |  |  |






<a name="integrationservice-pb-DeleteConfigurationsRequest"></a>

### DeleteConfigurationsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [IDFilter](#integrationservice-pb-IDFilter) | repeated |  |






<a name="integrationservice-pb-DeleteConfigurationsResponse"></a>

### DeleteConfigurationsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| count | [int64](#int64) |  |  |






<a name="integrationservice-pb-GetAppliedConfigurationsRequest"></a>

### GetAppliedConfigurationsRequest
TODO Naming


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [string](#string) | repeated |  |
| configuration_id_filter | [IDFilter](#integrationservice-pb-IDFilter) | repeated |  |
| device_id_filter | [string](#string) | repeated |  |
| condition_id_filter | [IDFilter](#integrationservice-pb-IDFilter) | repeated |  |






<a name="integrationservice-pb-GetConditionsRequest"></a>

### GetConditionsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [IDFilter](#integrationservice-pb-IDFilter) | repeated |  |






<a name="integrationservice-pb-GetConfigurationsRequest"></a>

### GetConfigurationsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [IDFilter](#integrationservice-pb-IDFilter) | repeated |  |






<a name="integrationservice-pb-IDFilter"></a>

### IDFilter
configuration/123?version=latest :) Jozko spravi :)


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| value | [uint64](#uint64) |  |  |
| all | [bool](#bool) |  |  |
| latest | [bool](#bool) |  |  |






<a name="integrationservice-pb-InvokeConfigurationRequest"></a>

### InvokeConfigurationRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| configuration_id | [string](#string) |  | applies latest configuration |
| device_id | [string](#string) |  |  |
| force | [bool](#bool) |  | force update even if the configuration has already been applied to device |
| correlation_id | [string](#string) |  | propagated down to the resource update command |
| id | [string](#string) |  |  |





 


<a name="integrationservice-pb-AppliedConfiguration-Resource-Status"></a>

### AppliedConfiguration.Resource.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| QUEUED | 0 |  |
| PENDING | 1 |  |
| DONE | 2 | If done look to resource_updated even update resource failed for resource aggregate. |
| TIMEOUT | 3 |  |


 

 


<a name="integrationservice-pb-IntegrationService"></a>

### IntegrationService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateCondition | [Condition](#integrationservice-pb-Condition) | [Condition](#integrationservice-pb-Condition) |  |
| GetConditions | [GetConditionsRequest](#integrationservice-pb-GetConditionsRequest) | [Condition](#integrationservice-pb-Condition) stream |  |
| DeleteConditions | [DeleteConditionsRequest](#integrationservice-pb-DeleteConditionsRequest) | [DeleteConditionsResponse](#integrationservice-pb-DeleteConditionsResponse) |  |
| UpdateCondition | [Condition](#integrationservice-pb-Condition) | [Condition](#integrationservice-pb-Condition) |  |
| CreateConfiguration | [Configuration](#integrationservice-pb-Configuration) | [Configuration](#integrationservice-pb-Configuration) |  |
| GetConfigurations | [GetConfigurationsRequest](#integrationservice-pb-GetConfigurationsRequest) | [Configuration](#integrationservice-pb-Configuration) stream |  |
| DeleteConfigurations | [DeleteConfigurationsRequest](#integrationservice-pb-DeleteConfigurationsRequest) | [DeleteConfigurationsResponse](#integrationservice-pb-DeleteConfigurationsResponse) |  |
| UpdateConfiguration | [Configuration](#integrationservice-pb-Configuration) | [Configuration](#integrationservice-pb-Configuration) |  |
| InvokeConfiguration | [InvokeConfigurationRequest](#integrationservice-pb-InvokeConfigurationRequest) | [AppliedConfiguration](#integrationservice-pb-AppliedConfiguration) stream | streaming process of update configuration to invoker |
| GetAppliedConfigurations | [GetAppliedConfigurationsRequest](#integrationservice-pb-GetAppliedConfigurationsRequest) | [AppliedConfiguration](#integrationservice-pb-AppliedConfiguration) stream |  |
| DeleteAppliedConfigurations | [DeleteAppliedConfigurationsRequest](#integrationservice-pb-DeleteAppliedConfigurationsRequest) | [DeleteAppliedConfigurationsResponse](#integrationservice-pb-DeleteAppliedConfigurationsResponse) |  |

 



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


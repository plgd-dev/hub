# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [snippet-service/pb/service.proto](#snippet-service_pb_service-proto)
    - [AppliedConfiguration](#snippetservice-pb-AppliedConfiguration)
    - [AppliedConfiguration.LinkedTo](#snippetservice-pb-AppliedConfiguration-LinkedTo)
    - [AppliedConfiguration.Resource](#snippetservice-pb-AppliedConfiguration-Resource)
    - [Condition](#snippetservice-pb-Condition)
    - [Configuration](#snippetservice-pb-Configuration)
    - [Configuration.Resource](#snippetservice-pb-Configuration-Resource)
    - [DeleteAppliedConfigurationsRequest](#snippetservice-pb-DeleteAppliedConfigurationsRequest)
    - [DeleteAppliedConfigurationsResponse](#snippetservice-pb-DeleteAppliedConfigurationsResponse)
    - [DeleteConditionsRequest](#snippetservice-pb-DeleteConditionsRequest)
    - [DeleteConditionsResponse](#snippetservice-pb-DeleteConditionsResponse)
    - [DeleteConfigurationsRequest](#snippetservice-pb-DeleteConfigurationsRequest)
    - [DeleteConfigurationsResponse](#snippetservice-pb-DeleteConfigurationsResponse)
    - [GetAppliedConfigurationsRequest](#snippetservice-pb-GetAppliedConfigurationsRequest)
    - [GetConditionsRequest](#snippetservice-pb-GetConditionsRequest)
    - [GetConfigurationsRequest](#snippetservice-pb-GetConfigurationsRequest)
    - [IDFilter](#snippetservice-pb-IDFilter)
    - [InvokeConfigurationRequest](#snippetservice-pb-InvokeConfigurationRequest)
    - [InvokeConfigurationResponse](#snippetservice-pb-InvokeConfigurationResponse)
  
    - [AppliedConfiguration.Resource.Status](#snippetservice-pb-AppliedConfiguration-Resource-Status)
  
    - [SnippetService](#snippetservice-pb-SnippetService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="snippet-service_pb_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## snippet-service/pb/service.proto



<a name="snippetservice-pb-AppliedConfiguration"></a>

### AppliedConfiguration



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| device_id | [string](#string) |  | @gotags: bson:&#34;deviceId&#34; |
| configuration_id | [AppliedConfiguration.LinkedTo](#snippetservice-pb-AppliedConfiguration-LinkedTo) |  | @gotags: bson:&#34;configurationId&#34; |
| on_demand | [bool](#bool) |  |  |
| condition_id | [AppliedConfiguration.LinkedTo](#snippetservice-pb-AppliedConfiguration-LinkedTo) |  | @gotags: bson:&#34;conditionId&#34; |
| resources | [AppliedConfiguration.Resource](#snippetservice-pb-AppliedConfiguration-Resource) | repeated |  |
| owner | [string](#string) |  |  |
| timestamp | [int64](#int64) |  | Unix timestamp in ns when the applied device configuration has been created/updated |






<a name="snippetservice-pb-AppliedConfiguration-LinkedTo"></a>

### AppliedConfiguration.LinkedTo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| version | [uint64](#uint64) |  |  |






<a name="snippetservice-pb-AppliedConfiguration-Resource"></a>

### AppliedConfiguration.Resource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| href | [string](#string) |  |  |
| correlation_id | [string](#string) |  | Reused from invoke command or generated. Can be used to retrieve corresponding pending command.

@gotags: bson:&#34;correlationId&#34; |
| status | [AppliedConfiguration.Resource.Status](#snippetservice-pb-AppliedConfiguration-Resource-Status) |  |  |
| resource_updated | [resourceaggregate.pb.ResourceUpdated](#resourceaggregate-pb-ResourceUpdated) |  | @gotags: bson:&#34;resourceUpdated,omitempty&#34; |
| valid_until | [int64](#int64) |  | Unix nanoseconds timestamp for resource in PENDING status, until which the pending update is valid

@gotags: bson:&#34;validUntil,omitempty&#34; |






<a name="snippetservice-pb-Condition"></a>

### Condition
driven by resource change event


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Condition ID |
| version | [uint64](#uint64) |  | Condition version |
| name | [string](#string) |  | User-friendly condition name |
| enabled | [bool](#bool) |  | Condition is enabled/disabled |
| configuration_id | [string](#string) |  | ID of the configuration to be applied when the condition is satisfied |
| device_id_filter | [string](#string) | repeated | list of device IDs to which the condition applies |
| resource_type_filter | [string](#string) | repeated |  |
| resource_href_filter | [string](#string) | repeated | list of resource hrefs to which the condition applies |
| jq_expression_filter | [string](#string) |  |  |
| api_access_token | [string](#string) |  | Token used to update resources in the configuration |
| owner | [string](#string) |  | Condition owner |
| timestamp | [int64](#int64) |  | Unix timestamp in ns when the condition has been created/updated |






<a name="snippetservice-pb-Configuration"></a>

### Configuration



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Configuration ID |
| version | [uint64](#uint64) |  | Configuration version |
| name | [string](#string) |  | User-friendly configuration name |
| resources | [Configuration.Resource](#snippetservice-pb-Configuration-Resource) | repeated | List of resource updates to be applied |
| owner | [string](#string) |  | Configuration owner |
| timestamp | [int64](#int64) |  | Unix timestamp in ns when the configuration has been created/updated |






<a name="snippetservice-pb-Configuration-Resource"></a>

### Configuration.Resource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| href | [string](#string) |  | href of the resource |
| content | [resourceaggregate.pb.Content](#resourceaggregate-pb-Content) |  | content update of the resource |
| time_to_live | [int64](#int64) |  | optional update command time to live, 0 is infinite |






<a name="snippetservice-pb-DeleteAppliedConfigurationsRequest"></a>

### DeleteAppliedConfigurationsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [string](#string) | repeated |  |






<a name="snippetservice-pb-DeleteAppliedConfigurationsResponse"></a>

### DeleteAppliedConfigurationsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| success | [bool](#bool) |  |  |






<a name="snippetservice-pb-DeleteConditionsRequest"></a>

### DeleteConditionsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [IDFilter](#snippetservice-pb-IDFilter) | repeated |  |
| http_id_filter | [string](#string) | repeated | **Deprecated.** Format: {id}/{version}, e.g., &#34;ae424c58-e517-4494-6de7-583536c48213/all&#34; or &#34;ae424c58-e517-4494-6de7-583536c48213/latest&#34; or &#34;ae424c58-e517-4494-6de7-583536c48213/1234&#34; |






<a name="snippetservice-pb-DeleteConditionsResponse"></a>

### DeleteConditionsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| success | [bool](#bool) |  |  |






<a name="snippetservice-pb-DeleteConfigurationsRequest"></a>

### DeleteConfigurationsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [IDFilter](#snippetservice-pb-IDFilter) | repeated |  |
| http_id_filter | [string](#string) | repeated | **Deprecated.** Format: {id}/{version}, e.g., &#34;ae424c58-e517-4494-6de7-583536c48213/all&#34; or &#34;ae424c58-e517-4494-6de7-583536c48213/latest&#34; or &#34;ae424c58-e517-4494-6de7-583536c48213/1234&#34; |






<a name="snippetservice-pb-DeleteConfigurationsResponse"></a>

### DeleteConfigurationsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| success | [bool](#bool) |  |  |






<a name="snippetservice-pb-GetAppliedConfigurationsRequest"></a>

### GetAppliedConfigurationsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [string](#string) | repeated |  |
| device_id_filter | [string](#string) | repeated |  |
| configuration_id_filter | [IDFilter](#snippetservice-pb-IDFilter) | repeated |  |
| http_configuration_id_filter | [string](#string) | repeated | **Deprecated.** Format: {id}/{version}, e.g., &#34;ae424c58-e517-4494-6de7-583536c48213/all&#34; or &#34;ae424c58-e517-4494-6de7-583536c48213/latest&#34; or &#34;ae424c58-e517-4494-6de7-583536c48213/1234&#34; |
| condition_id_filter | [IDFilter](#snippetservice-pb-IDFilter) | repeated |  |
| http_condition_id_filter | [string](#string) | repeated | **Deprecated.** Format: {id}/{version}, e.g., &#34;ae424c58-e517-4494-6de7-583536c48213/all&#34; or &#34;ae424c58-e517-4494-6de7-583536c48213/latest&#34; or &#34;ae424c58-e517-4494-6de7-583536c48213/1234&#34; |






<a name="snippetservice-pb-GetConditionsRequest"></a>

### GetConditionsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [IDFilter](#snippetservice-pb-IDFilter) | repeated |  |
| configuration_id_filter | [string](#string) | repeated | returns latest conditions for given configurationId |
| http_id_filter | [string](#string) | repeated | **Deprecated.** Format: {id}/{version}, e.g., &#34;ae424c58-e517-4494-6de7-583536c48213/all&#34; or &#34;ae424c58-e517-4494-6de7-583536c48213/latest&#34; or &#34;ae424c58-e517-4494-6de7-583536c48213/1234&#34; |






<a name="snippetservice-pb-GetConfigurationsRequest"></a>

### GetConfigurationsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [IDFilter](#snippetservice-pb-IDFilter) | repeated |  |
| http_id_filter | [string](#string) | repeated | **Deprecated.** Format: {id}/{version}, e.g., &#34;ae424c58-e517-4494-6de7-583536c48213/all&#34; or &#34;ae424c58-e517-4494-6de7-583536c48213/latest&#34; or &#34;ae424c58-e517-4494-6de7-583536c48213/1234&#34; |






<a name="snippetservice-pb-IDFilter"></a>

### IDFilter



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| value | [uint64](#uint64) |  |  |
| all | [bool](#bool) |  |  |
| latest | [bool](#bool) |  |  |






<a name="snippetservice-pb-InvokeConfigurationRequest"></a>

### InvokeConfigurationRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| configuration_id | [string](#string) |  | applies latest configuration |
| device_id | [string](#string) |  |  |
| force | [bool](#bool) |  | force update even if the configuration has already been applied to device |
| correlation_id | [string](#string) |  | propagated down to the resource update command |






<a name="snippetservice-pb-InvokeConfigurationResponse"></a>

### InvokeConfigurationResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| applied_configuration_id | [string](#string) |  |  |





 


<a name="snippetservice-pb-AppliedConfiguration-Resource-Status"></a>

### AppliedConfiguration.Resource.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| UNSPECIFIED | 0 |  |
| QUEUED | 1 |  |
| PENDING | 2 |  |
| DONE | 3 | If done look to resource_updated if update resource failed for resource aggregate. |
| TIMEOUT | 4 |  |


 

 


<a name="snippetservice-pb-SnippetService"></a>

### SnippetService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateCondition | [Condition](#snippetservice-pb-Condition) | [Condition](#snippetservice-pb-Condition) |  |
| GetConditions | [GetConditionsRequest](#snippetservice-pb-GetConditionsRequest) | [Condition](#snippetservice-pb-Condition) stream |  |
| DeleteConditions | [DeleteConditionsRequest](#snippetservice-pb-DeleteConditionsRequest) | [DeleteConditionsResponse](#snippetservice-pb-DeleteConditionsResponse) |  |
| UpdateCondition | [Condition](#snippetservice-pb-Condition) | [Condition](#snippetservice-pb-Condition) | For update the condition whole condition is required and the version must be incremented. |
| CreateConfiguration | [Configuration](#snippetservice-pb-Configuration) | [Configuration](#snippetservice-pb-Configuration) |  |
| GetConfigurations | [GetConfigurationsRequest](#snippetservice-pb-GetConfigurationsRequest) | [Configuration](#snippetservice-pb-Configuration) stream |  |
| DeleteConfigurations | [DeleteConfigurationsRequest](#snippetservice-pb-DeleteConfigurationsRequest) | [DeleteConfigurationsResponse](#snippetservice-pb-DeleteConfigurationsResponse) |  |
| UpdateConfiguration | [Configuration](#snippetservice-pb-Configuration) | [Configuration](#snippetservice-pb-Configuration) | For update the configuration whole configuration is required and the version must be incremented. |
| InvokeConfiguration | [InvokeConfigurationRequest](#snippetservice-pb-InvokeConfigurationRequest) | [InvokeConfigurationResponse](#snippetservice-pb-InvokeConfigurationResponse) | streaming process of update configuration to invoker |
| GetAppliedConfigurations | [GetAppliedConfigurationsRequest](#snippetservice-pb-GetAppliedConfigurationsRequest) | [AppliedConfiguration](#snippetservice-pb-AppliedConfiguration) stream |  |
| DeleteAppliedConfigurations | [DeleteAppliedConfigurationsRequest](#snippetservice-pb-DeleteAppliedConfigurationsRequest) | [DeleteAppliedConfigurationsResponse](#snippetservice-pb-DeleteAppliedConfigurationsResponse) |  |

 



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


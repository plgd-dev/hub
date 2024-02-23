# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [snapshot-service/pb/ruleActionLink.proto](#snapshot-service_pb_ruleActionLink-proto)
    - [AppliedRuleActionLink](#snapshotservice-pb-AppliedRuleActionLink)
    - [RuleActionLink](#snapshotservice-pb-RuleActionLink)
    - [RuleActionParameters](#snapshotservice-pb-RuleActionParameters)
    - [RuleActionParameters.Origin](#snapshotservice-pb-RuleActionParameters-Origin)
  
- [snapshot-service/pb/rule.proto](#snapshot-service_pb_rule-proto)
    - [LogicalExpression](#snapshotservice-pb-LogicalExpression)
    - [NegationExpression](#snapshotservice-pb-NegationExpression)
    - [OriginId](#snapshotservice-pb-OriginId)
    - [ParenthesesExpression](#snapshotservice-pb-ParenthesesExpression)
    - [RelationalExpression](#snapshotservice-pb-RelationalExpression)
    - [RelationalOperand](#snapshotservice-pb-RelationalOperand)
    - [RepeatedString](#snapshotservice-pb-RepeatedString)
    - [ResourceProperty](#snapshotservice-pb-ResourceProperty)
    - [Rule](#snapshotservice-pb-Rule)
    - [RuleExpression](#snapshotservice-pb-RuleExpression)
  
    - [LogicalOperator](#snapshotservice-pb-LogicalOperator)
    - [RelationalOperator](#snapshotservice-pb-RelationalOperator)
    - [StringOperator](#snapshotservice-pb-StringOperator)
  
- [snapshot-service/pb/scene.proto](#snapshot-service_pb_scene-proto)
    - [GenerateContent](#snapshotservice-pb-GenerateContent)
    - [ResourceSnapshot](#snapshotservice-pb-ResourceSnapshot)
    - [ResourceSnapshot.ApplyToDevices](#snapshotservice-pb-ResourceSnapshot-ApplyToDevices)
    - [ResourceSnapshot.ApplyToResources](#snapshotservice-pb-ResourceSnapshot-ApplyToResources)
    - [Scene](#snapshotservice-pb-Scene)
  
- [snapshot-service/pb/service.proto](#snapshot-service_pb_service-proto)
    - [CreateRuleActionLinkResponse](#snapshotservice-pb-CreateRuleActionLinkResponse)
    - [CreateRuleResponse](#snapshotservice-pb-CreateRuleResponse)
    - [CreateSceneRequest](#snapshotservice-pb-CreateSceneRequest)
    - [CreateSceneResponse](#snapshotservice-pb-CreateSceneResponse)
    - [DeleteAppliedRuleActionLinkRequest](#snapshotservice-pb-DeleteAppliedRuleActionLinkRequest)
    - [DeleteAppliedRuleActionLinkResponse](#snapshotservice-pb-DeleteAppliedRuleActionLinkResponse)
    - [DeleteRuleActionLinksRequest](#snapshotservice-pb-DeleteRuleActionLinksRequest)
    - [DeleteRuleActionLinksResponse](#snapshotservice-pb-DeleteRuleActionLinksResponse)
    - [DeleteRulesRequest](#snapshotservice-pb-DeleteRulesRequest)
    - [DeleteRulesResponse](#snapshotservice-pb-DeleteRulesResponse)
    - [DeleteScenesRequest](#snapshotservice-pb-DeleteScenesRequest)
    - [DeleteScenesResponse](#snapshotservice-pb-DeleteScenesResponse)
    - [GetAppliedRuleActionLinksRequest](#snapshotservice-pb-GetAppliedRuleActionLinksRequest)
    - [GetRuleActionLinksRequest](#snapshotservice-pb-GetRuleActionLinksRequest)
    - [GetRulesRequest](#snapshotservice-pb-GetRulesRequest)
    - [GetScenesRequest](#snapshotservice-pb-GetScenesRequest)
    - [InvokeRuleActionLinkRequest](#snapshotservice-pb-InvokeRuleActionLinkRequest)
    - [InvokeRuleActionLinkResponse](#snapshotservice-pb-InvokeRuleActionLinkResponse)
    - [InvokeRuleRequest](#snapshotservice-pb-InvokeRuleRequest)
    - [InvokeRuleResponse](#snapshotservice-pb-InvokeRuleResponse)
    - [UpdateRuleActionLinkResponse](#snapshotservice-pb-UpdateRuleActionLinkResponse)
    - [UpdateRuleResponse](#snapshotservice-pb-UpdateRuleResponse)
    - [UpdateSceneResponse](#snapshotservice-pb-UpdateSceneResponse)
  
    - [RuleEngine](#snapshotservice-pb-RuleEngine)
  
- [snapshot-service/pb/yqEngine.proto](#snapshot-service_pb_yqEngine-proto)
    - [YQEngine](#snapshotservice-pb-YQEngine)
    - [YQEngine.Input](#snapshotservice-pb-YQEngine-Input)
  
- [Scalar Value Types](#scalar-value-types)



<a name="snapshot-service_pb_ruleActionLink-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## snapshot-service/pb/ruleActionLink.proto



<a name="snapshotservice-pb-AppliedRuleActionLink"></a>

### AppliedRuleActionLink



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | the id of the rule link |
| origin_invoker | [resourceaggregate.pb.ResourceId](#resourceaggregate-pb-ResourceId) |  | for once_per_device or once_per_resource |
| timestamp_start | [int64](#int64) |  | when the rule association was applied |
| timestamp_end | [int64](#int64) |  | when the rule association was removed |
| resources_updated | [resourceaggregate.pb.ResourceUpdated](#resourceaggregate-pb-ResourceUpdated) | repeated |  |






<a name="snapshotservice-pb-RuleActionLink"></a>

### RuleActionLink



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| name | [string](#string) |  |  |
| enabled | [bool](#bool) |  |  |
| rule_id | [string](#string) |  |  |
| owner | [string](#string) |  |  |
| once_per_device | [bool](#bool) |  | the rule association is applied once per device |
| once_per_resource | [bool](#bool) |  | the rule will be invoked once per resource |
| interval_per_device | [int64](#int64) |  | in nanoseconds |
| interval_per_resource | [int64](#int64) |  | in nanoseconds |
| always | [bool](#bool) |  |  |
| scenes | [RuleActionParameters](#snapshotservice-pb-RuleActionParameters) | repeated | scenes to invoke |
| rules | [RuleActionParameters](#snapshotservice-pb-RuleActionParameters) | repeated | rules to invoke |






<a name="snapshotservice-pb-RuleActionParameters"></a>

### RuleActionParameters



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| origin | [RuleActionParameters.Origin](#snapshotservice-pb-RuleActionParameters-Origin) |  | use the origin resource |
| resource_id | [resourceaggregate.pb.ResourceId](#resourceaggregate-pb-ResourceId) |  | use the given resource |






<a name="snapshotservice-pb-RuleActionParameters-Origin"></a>

### RuleActionParameters.Origin



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| new_href | [string](#string) |  | replace the origin href with the given href |





 

 

 

 



<a name="snapshot-service_pb_rule-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## snapshot-service/pb/rule.proto



<a name="snapshotservice-pb-LogicalExpression"></a>

### LogicalExpression



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| left_expression | [RuleExpression](#snapshotservice-pb-RuleExpression) |  |  |
| operator | [LogicalOperator](#snapshotservice-pb-LogicalOperator) |  |  |
| right_expression | [RuleExpression](#snapshotservice-pb-RuleExpression) |  |  |






<a name="snapshotservice-pb-NegationExpression"></a>

### NegationExpression



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| inner_expression | [RuleExpression](#snapshotservice-pb-RuleExpression) |  |  |






<a name="snapshotservice-pb-OriginId"></a>

### OriginId



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| href | [string](#string) |  | path to resource in the device (e.g. /oic/d ) |
| types | [RepeatedString](#snapshotservice-pb-RepeatedString) |  | resource must contains all types (e.g. [oic.r.switch.binary]) |






<a name="snapshotservice-pb-ParenthesesExpression"></a>

### ParenthesesExpression



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| inner_expression | [RuleExpression](#snapshotservice-pb-RuleExpression) |  |  |






<a name="snapshotservice-pb-RelationalExpression"></a>

### RelationalExpression



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| left_operand | [RelationalOperand](#snapshotservice-pb-RelationalOperand) |  |  |
| compare | [RelationalOperator](#snapshotservice-pb-RelationalOperator) |  |  |
| string | [StringOperator](#snapshotservice-pb-StringOperator) |  |  |
| right_operand | [RelationalOperand](#snapshotservice-pb-RelationalOperand) |  |  |






<a name="snapshotservice-pb-RelationalOperand"></a>

### RelationalOperand



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| scalar | [google.protobuf.Value](#google-protobuf-Value) |  |  |
| resource_property | [ResourceProperty](#snapshotservice-pb-ResourceProperty) |  |  |






<a name="snapshotservice-pb-RepeatedString"></a>

### RepeatedString



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| list | [string](#string) | repeated |  |






<a name="snapshotservice-pb-ResourceProperty"></a>

### ResourceProperty



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| origin_id | [OriginId](#snapshotservice-pb-OriginId) |  |  |
| resource_id | [resourceaggregate.pb.ResourceId](#resourceaggregate-pb-ResourceId) |  |  |
| property_path | [string](#string) |  | jq-like path |






<a name="snapshotservice-pb-Rule"></a>

### Rule



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| name | [string](#string) |  |  |
| description | [string](#string) |  |  |
| yq_engine | [YQEngine](#snapshotservice-pb-YQEngine) |  |  |
| enabled | [bool](#bool) |  | RuleExpression expression = 4; |
| owner | [string](#string) |  |  |






<a name="snapshotservice-pb-RuleExpression"></a>

### RuleExpression



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bool | [bool](#bool) |  |  |
| relational_expression | [RelationalExpression](#snapshotservice-pb-RelationalExpression) |  |  |
| logical_expression | [LogicalExpression](#snapshotservice-pb-LogicalExpression) |  |  |
| parentheses_expression | [ParenthesesExpression](#snapshotservice-pb-ParenthesesExpression) |  |  |
| negation_expression | [NegationExpression](#snapshotservice-pb-NegationExpression) |  |  |





 


<a name="snapshotservice-pb-LogicalOperator"></a>

### LogicalOperator


| Name | Number | Description |
| ---- | ------ | ----------- |
| AND | 0 |  |
| OR | 1 |  |



<a name="snapshotservice-pb-RelationalOperator"></a>

### RelationalOperator


| Name | Number | Description |
| ---- | ------ | ----------- |
| EQUAL | 0 |  |
| NOT_EQUAL | 1 |  |
| LESS_THAN | 2 |  |
| LESS_THAN_OR_EQUAL | 3 |  |
| GREATER_THAN | 4 |  |
| GREATER_THAN_OR_EQUAL | 5 |  |



<a name="snapshotservice-pb-StringOperator"></a>

### StringOperator


| Name | Number | Description |
| ---- | ------ | ----------- |
| CONTAINS | 0 |  |
| DOES_NOT_CONTAIN | 1 |  |
| STARTS_WITH | 2 |  |
| ENDS_WITH | 3 |  |


 

 

 



<a name="snapshot-service_pb_scene-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## snapshot-service/pb/scene.proto



<a name="snapshotservice-pb-GenerateContent"></a>

### GenerateContent



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| yq_engine | [YQEngine](#snapshotservice-pb-YQEngine) |  |  |
| content_type | [string](#string) |  |  |






<a name="snapshotservice-pb-ResourceSnapshot"></a>

### ResourceSnapshot



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| device_filter | [ResourceSnapshot.ApplyToDevices](#snapshotservice-pb-ResourceSnapshot-ApplyToDevices) |  |  |
| resource_filter | [ResourceSnapshot.ApplyToResources](#snapshotservice-pb-ResourceSnapshot-ApplyToResources) |  |  |
| defined | [resourceaggregate.pb.Content](#resourceaggregate-pb-Content) |  |  |
| generated | [GenerateContent](#snapshotservice-pb-GenerateContent) |  |  |
| resource_interface | [string](#string) |  | optional update interface |
| time_to_live | [int64](#int64) |  | optional update command time to live |






<a name="snapshotservice-pb-ResourceSnapshot-ApplyToDevices"></a>

### ResourceSnapshot.ApplyToDevices



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| origin | [bool](#bool) |  | device that is originally started the scene |
| device_ids | [string](#string) | repeated | identifies devices |
| device_types | [string](#string) | repeated | device type just for filtering device when device_ids and current device is not set |






<a name="snapshotservice-pb-ResourceSnapshot-ApplyToResources"></a>

### ResourceSnapshot.ApplyToResources



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| hrefs | [string](#string) | repeated | identifies resources at the device. |
| types | [string](#string) | repeated | and operator among types -&gt; type defines a list of properties in content |






<a name="snapshotservice-pb-Scene"></a>

### Scene



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| name | [string](#string) |  |  |
| resources | [ResourceSnapshot](#snapshotservice-pb-ResourceSnapshot) | repeated |  |





 

 

 

 



<a name="snapshot-service_pb_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## snapshot-service/pb/service.proto



<a name="snapshotservice-pb-CreateRuleActionLinkResponse"></a>

### CreateRuleActionLinkResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="snapshotservice-pb-CreateRuleResponse"></a>

### CreateRuleResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="snapshotservice-pb-CreateSceneRequest"></a>

### CreateSceneRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| resources | [ResourceSnapshot](#snapshotservice-pb-ResourceSnapshot) | repeated |  |
| device_id_filter | [string](#string) | repeated |  |
| resource_id | [resourceaggregate.pb.ResourceId](#resourceaggregate-pb-ResourceId) | repeated |  |
| type_filter | [string](#string) | repeated |  |






<a name="snapshotservice-pb-CreateSceneResponse"></a>

### CreateSceneResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="snapshotservice-pb-DeleteAppliedRuleActionLinkRequest"></a>

### DeleteAppliedRuleActionLinkRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [string](#string) | repeated |  |






<a name="snapshotservice-pb-DeleteAppliedRuleActionLinkResponse"></a>

### DeleteAppliedRuleActionLinkResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| count | [int64](#int64) |  |  |






<a name="snapshotservice-pb-DeleteRuleActionLinksRequest"></a>

### DeleteRuleActionLinksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [string](#string) | repeated |  |






<a name="snapshotservice-pb-DeleteRuleActionLinksResponse"></a>

### DeleteRuleActionLinksResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| count | [int64](#int64) |  |  |






<a name="snapshotservice-pb-DeleteRulesRequest"></a>

### DeleteRulesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [string](#string) | repeated |  |






<a name="snapshotservice-pb-DeleteRulesResponse"></a>

### DeleteRulesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| count | [int64](#int64) |  |  |






<a name="snapshotservice-pb-DeleteScenesRequest"></a>

### DeleteScenesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [string](#string) | repeated |  |






<a name="snapshotservice-pb-DeleteScenesResponse"></a>

### DeleteScenesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| count | [int64](#int64) |  |  |






<a name="snapshotservice-pb-GetAppliedRuleActionLinksRequest"></a>

### GetAppliedRuleActionLinksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [string](#string) | repeated |  |
| rule_id_filter | [string](#string) | repeated |  |
| device_id_filter | [string](#string) | repeated |  |
| resource_id_filter | [resourceaggregate.pb.ResourceId](#resourceaggregate-pb-ResourceId) | repeated |  |






<a name="snapshotservice-pb-GetRuleActionLinksRequest"></a>

### GetRuleActionLinksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [string](#string) | repeated |  |






<a name="snapshotservice-pb-GetRulesRequest"></a>

### GetRulesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [string](#string) | repeated |  |






<a name="snapshotservice-pb-GetScenesRequest"></a>

### GetScenesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [string](#string) | repeated |  |






<a name="snapshotservice-pb-InvokeRuleActionLinkRequest"></a>

### InvokeRuleActionLinkRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| resource_id | [resourceaggregate.pb.ResourceId](#resourceaggregate-pb-ResourceId) |  | for once_per_device or once_per_resource |
| force | [bool](#bool) |  | override the frequency |






<a name="snapshotservice-pb-InvokeRuleActionLinkResponse"></a>

### InvokeRuleActionLinkResponse







<a name="snapshotservice-pb-InvokeRuleRequest"></a>

### InvokeRuleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| resource_id | [resourceaggregate.pb.ResourceId](#resourceaggregate-pb-ResourceId) |  | origin resource that starts the execution |
| content | [resourceaggregate.pb.Content](#resourceaggregate-pb-Content) |  | use content instead of load it for resource_id |






<a name="snapshotservice-pb-InvokeRuleResponse"></a>

### InvokeRuleResponse







<a name="snapshotservice-pb-UpdateRuleActionLinkResponse"></a>

### UpdateRuleActionLinkResponse







<a name="snapshotservice-pb-UpdateRuleResponse"></a>

### UpdateRuleResponse







<a name="snapshotservice-pb-UpdateSceneResponse"></a>

### UpdateSceneResponse






 

 

 


<a name="snapshotservice-pb-RuleEngine"></a>

### RuleEngine


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateScene | [Scene](#snapshotservice-pb-Scene) | [CreateSceneResponse](#snapshotservice-pb-CreateSceneResponse) |  |
| GetScenes | [GetScenesRequest](#snapshotservice-pb-GetScenesRequest) | [Scene](#snapshotservice-pb-Scene) stream |  |
| DeleteScenes | [DeleteScenesRequest](#snapshotservice-pb-DeleteScenesRequest) | [DeleteScenesResponse](#snapshotservice-pb-DeleteScenesResponse) |  |
| UpdateScene | [Scene](#snapshotservice-pb-Scene) | [UpdateSceneResponse](#snapshotservice-pb-UpdateSceneResponse) |  |
| CreateRule | [Rule](#snapshotservice-pb-Rule) | [CreateRuleResponse](#snapshotservice-pb-CreateRuleResponse) |  |
| GetRules | [GetRulesRequest](#snapshotservice-pb-GetRulesRequest) | [Rule](#snapshotservice-pb-Rule) stream |  |
| DeleteRules | [DeleteRulesRequest](#snapshotservice-pb-DeleteRulesRequest) | [DeleteRulesResponse](#snapshotservice-pb-DeleteRulesResponse) |  |
| UpdateRule | [Rule](#snapshotservice-pb-Rule) | [UpdateRuleResponse](#snapshotservice-pb-UpdateRuleResponse) |  |
| InvokeRule | [InvokeRuleRequest](#snapshotservice-pb-InvokeRuleRequest) | [InvokeRuleResponse](#snapshotservice-pb-InvokeRuleResponse) |  |
| CreateRuleActionLink | [RuleActionLink](#snapshotservice-pb-RuleActionLink) | [CreateRuleActionLinkResponse](#snapshotservice-pb-CreateRuleActionLinkResponse) |  |
| GetRuleActionLinks | [GetRuleActionLinksRequest](#snapshotservice-pb-GetRuleActionLinksRequest) | [RuleActionLink](#snapshotservice-pb-RuleActionLink) stream |  |
| DeleteRuleActionLinks | [DeleteRuleActionLinksRequest](#snapshotservice-pb-DeleteRuleActionLinksRequest) | [DeleteRuleActionLinksResponse](#snapshotservice-pb-DeleteRuleActionLinksResponse) |  |
| UpdateRuleActionLink | [RuleActionLink](#snapshotservice-pb-RuleActionLink) | [UpdateRuleActionLinkResponse](#snapshotservice-pb-UpdateRuleActionLinkResponse) |  |
| InvokeRuleActionLink | [InvokeRuleActionLinkRequest](#snapshotservice-pb-InvokeRuleActionLinkRequest) | [InvokeRuleActionLinkResponse](#snapshotservice-pb-InvokeRuleActionLinkResponse) |  |
| GetAppliedRuleActionLinks | [GetAppliedRuleActionLinksRequest](#snapshotservice-pb-GetAppliedRuleActionLinksRequest) | [AppliedRuleActionLink](#snapshotservice-pb-AppliedRuleActionLink) |  |
| DeleteAppliedRuleActionLink | [DeleteAppliedRuleActionLinkRequest](#snapshotservice-pb-DeleteAppliedRuleActionLinkRequest) | [DeleteAppliedRuleActionLinkResponse](#snapshotservice-pb-DeleteAppliedRuleActionLinkResponse) |  |

 



<a name="snapshot-service_pb_yqEngine-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## snapshot-service/pb/yqEngine.proto



<a name="snapshotservice-pb-YQEngine"></a>

### YQEngine



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| inputs | [YQEngine.Input](#snapshotservice-pb-YQEngine-Input) | repeated | content of resources will encoded on yq via $name |
| yq_expression | [string](#string) |  |  |






<a name="snapshotservice-pb-YQEngine-Input"></a>

### YQEngine.Input



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| resource_id | [resourceaggregate.pb.ResourceId](#resourceaggregate-pb-ResourceId) |  |  |
| value | [google.protobuf.Value](#google-protobuf-Value) |  |  |





 

 

 

 



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


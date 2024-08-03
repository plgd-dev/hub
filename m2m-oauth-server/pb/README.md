# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [m2m-oauth-server/pb/service.proto](#m2m-oauth-server_pb_service-proto)
    - [CreateTokenRequest](#m2moauthserver-pb-CreateTokenRequest)
    - [CreateTokenResponse](#m2moauthserver-pb-CreateTokenResponse)
    - [DeleteTokensRequest](#m2moauthserver-pb-DeleteTokensRequest)
    - [DeleteTokensResponse](#m2moauthserver-pb-DeleteTokensResponse)
    - [GetTokensRequest](#m2moauthserver-pb-GetTokensRequest)
    - [Token](#m2moauthserver-pb-Token)
    - [Token.BlackListed](#m2moauthserver-pb-Token-BlackListed)
  
    - [M2MOAuthService](#m2moauthserver-pb-M2MOAuthService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="m2m-oauth-server_pb_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## m2m-oauth-server/pb/service.proto



<a name="m2moauthserver-pb-CreateTokenRequest"></a>

### CreateTokenRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| client_id | [string](#string) |  | Client ID |
| client_secret | [string](#string) |  | Client Secret |
| audience | [string](#string) | repeated | Requested token Audience |
| scope | [string](#string) | repeated | Requested token scopes |
| expiration | [int64](#int64) |  | The requested expiration time in unit timestamp seconds from the client. If not provided, the token will use the maximum allowed by the client, or if it exceeds the maximum allowed, an error will occur. |
| client_assertion_type | [string](#string) |  | Client assertion type |
| client_assertion | [string](#string) |  | Client assertion |
| token_name | [string](#string) |  | Token name |
| grant_type | [string](#string) |  | Grant type |






<a name="m2moauthserver-pb-CreateTokenResponse"></a>

### CreateTokenResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| access_token | [string](#string) |  |  |
| token_type | [string](#string) |  |  |
| expires_in | [int64](#int64) |  |  |
| scope | [string](#string) | repeated |  |






<a name="m2moauthserver-pb-DeleteTokensRequest"></a>

### DeleteTokensRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [string](#string) | repeated |  |






<a name="m2moauthserver-pb-DeleteTokensResponse"></a>

### DeleteTokensResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| blacklisted_count | [int64](#int64) |  |  |
| deleted_count | [int64](#int64) |  |  |






<a name="m2moauthserver-pb-GetTokensRequest"></a>

### GetTokensRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [string](#string) | repeated |  |
| include_blacklisted | [bool](#bool) |  |  |






<a name="m2moauthserver-pb-Token"></a>

### Token
Tokens are deleted from DB after they are expired and blacklisted/revoked

driven by resource change event


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Token ID / jti |
| version | [uint64](#uint64) |  | Incremental version for update |
| name | [string](#string) |  | User-friendly token name |
| owner | [string](#string) |  | Owner of the token |
| issued_at | [int64](#int64) |  | Unix timestamp in s when the condition has been created/updated |
| audience | [string](#string) | repeated | Token Audience |
| scope | [string](#string) | repeated | Token scopes |
| expiration | [int64](#int64) |  | Token expiration in Unix timestamp seconds |
| client_id | [string](#string) |  | Client ID |
| original_token_claims | [google.protobuf.Value](#google-protobuf-Value) |  | Original token claims |
| blacklisted | [Token.BlackListed](#m2moauthserver-pb-Token-BlackListed) |  | Token black list section |
| subject | [string](#string) |  | Subject of the token |






<a name="m2moauthserver-pb-Token-BlackListed"></a>

### Token.BlackListed



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| flag | [bool](#bool) |  | Blacklisted enabled flag, if once token has been blacklisted then it can&#39;t be unblacklisted/unrevoked |
| timestamp | [int64](#int64) |  | Unix timestamp in s when the token has been blacklisted |





 

 

 


<a name="m2moauthserver-pb-M2MOAuthService"></a>

### M2MOAuthService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateToken | [CreateTokenRequest](#m2moauthserver-pb-CreateTokenRequest) | [CreateTokenResponse](#m2moauthserver-pb-CreateTokenResponse) | Creates a new token |
| GetTokens | [GetTokensRequest](#m2moauthserver-pb-GetTokensRequest) | [Token](#m2moauthserver-pb-Token) stream | Returns all tokens of the owner |
| DeleteTokens | [DeleteTokensRequest](#m2moauthserver-pb-DeleteTokensRequest) | [DeleteTokensResponse](#m2moauthserver-pb-DeleteTokensResponse) | Deletes/blacklist tokens |

 



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


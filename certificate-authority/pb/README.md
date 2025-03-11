# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [certificate-authority/pb/cert.proto](#certificate-authority_pb_cert-proto)
    - [SignCertificateRequest](#certificateauthority-pb-SignCertificateRequest)
    - [SignCertificateResponse](#certificateauthority-pb-SignCertificateResponse)
  
- [certificate-authority/pb/service.proto](#certificate-authority_pb_service-proto)
    - [CertificateAuthority](#certificateauthority-pb-CertificateAuthority)
  
- [certificate-authority/pb/signingRecords.proto](#certificate-authority_pb_signingRecords-proto)
    - [CredentialStatus](#certificateauthority-pb-CredentialStatus)
    - [DeleteSigningRecordsRequest](#certificateauthority-pb-DeleteSigningRecordsRequest)
    - [DeletedSigningRecords](#certificateauthority-pb-DeletedSigningRecords)
    - [GetSigningRecordsRequest](#certificateauthority-pb-GetSigningRecordsRequest)
    - [SigningRecord](#certificateauthority-pb-SigningRecord)
  
- [Scalar Value Types](#scalar-value-types)



<a name="certificate-authority_pb_cert-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## certificate-authority/pb/cert.proto



<a name="certificateauthority-pb-SignCertificateRequest"></a>

### SignCertificateRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| certificate_signing_request | [bytes](#bytes) |  | PEM format |






<a name="certificateauthority-pb-SignCertificateResponse"></a>

### SignCertificateResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| certificate | [bytes](#bytes) |  | PEM format |





 

 

 

 



<a name="certificate-authority_pb_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## certificate-authority/pb/service.proto


 

 

 


<a name="certificateauthority-pb-CertificateAuthority"></a>

### CertificateAuthority


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| SignIdentityCertificate | [SignCertificateRequest](#certificateauthority-pb-SignCertificateRequest) | [SignCertificateResponse](#certificateauthority-pb-SignCertificateResponse) | SignIdentityCertificate sends a Identity Certificate Signing Request to the certificate authority and obtains a signed certificate. Both in the PEM format. It adds EKU: &#39;1.3.6.1.4.1.44924.1.6&#39; . |
| SignCertificate | [SignCertificateRequest](#certificateauthority-pb-SignCertificateRequest) | [SignCertificateResponse](#certificateauthority-pb-SignCertificateResponse) | SignCertificate sends a Certificate Signing Request to the certificate authority and obtains a signed certificate. Both in the PEM format. |
| GetSigningRecords | [GetSigningRecordsRequest](#certificateauthority-pb-GetSigningRecordsRequest) | [SigningRecord](#certificateauthority-pb-SigningRecord) stream | Get signed certificate records. |
| DeleteSigningRecords | [DeleteSigningRecordsRequest](#certificateauthority-pb-DeleteSigningRecordsRequest) | [DeletedSigningRecords](#certificateauthority-pb-DeletedSigningRecords) | Revoke signed certificate records or delete expired signed certificate records. |

 



<a name="certificate-authority_pb_signingRecords-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## certificate-authority/pb/signingRecords.proto



<a name="certificateauthority-pb-CredentialStatus"></a>

### CredentialStatus



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| date | [int64](#int64) |  | Last time the device requested provisioning, in unix nanoseconds timestamp format.

@gotags: bson:&#34;date&#34; |
| certificate_pem | [string](#string) |  | Last certificate issued.

@gotags: bson:&#34;identityCertificate&#34; |
| valid_until_date | [int64](#int64) |  | Record valid until date, in unix nanoseconds timestamp format

@gotags: bson:&#34;validUntilDate&#34; |
| serial | [string](#string) |  | Serial number of the last certificate issued

@gotags: bson:&#34;serial&#34; |
| issuer_id | [string](#string) |  | Issuer id is calculated from the issuer&#39;s public certificate, and it is computed as uuid.NewSHA1(uuid.NameSpaceX500, publicKeyRaw)

@gotags: bson:&#34;issuerId&#34; |






<a name="certificateauthority-pb-DeleteSigningRecordsRequest"></a>

### DeleteSigningRecordsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [string](#string) | repeated | Filter by id. |
| device_id_filter | [string](#string) | repeated | Filter by common_name. |






<a name="certificateauthority-pb-DeletedSigningRecords"></a>

### DeletedSigningRecords
Revoke or delete certificates


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| count | [int64](#int64) |  | Number of deleted records. |






<a name="certificateauthority-pb-GetSigningRecordsRequest"></a>

### GetSigningRecordsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_filter | [string](#string) | repeated | Filter by id. |
| common_name_filter | [string](#string) | repeated | Filter by common_name. |
| device_id_filter | [string](#string) | repeated | Filter by device_id - provides only identity certificates. |






<a name="certificateauthority-pb-SigningRecord"></a>

### SigningRecord



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | The registration ID is determined by applying a formula that utilizes the certificate properties, and it is computed as uuid.NewSHA1(uuid.NameSpaceX500, common_name &#43; uuid.NewSHA1(uuid.NameSpaceX500, publicKeyRaw)).

@gotags: bson:&#34;_id&#34; |
| owner | [string](#string) |  | Certificate owner.

@gotags: bson:&#34;owner&#34; |
| common_name | [string](#string) |  | Common name of the certificate. If device_id is provided in the common name, then for update public key must be same.

@gotags: bson:&#34;commonName&#34; |
| device_id | [string](#string) |  | DeviceID of the identity certificate.

@gotags: bson:&#34;deviceId,omitempty&#34; |
| public_key | [string](#string) |  | Public key fingerprint in uuid.NewSHA1(uuid.NameSpaceX500, publicKeyRaw) of the certificate.

@gotags: bson:&#34;publicKey&#34; |
| creation_date | [int64](#int64) |  | Record creation date, in unix nanoseconds timestamp format

@gotags: bson:&#34;creationDate,omitempty&#34; |
| credential | [CredentialStatus](#certificateauthority-pb-CredentialStatus) |  | Last credential provision overview.

@gotags: bson:&#34;credential&#34; |





 

 

 

 



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


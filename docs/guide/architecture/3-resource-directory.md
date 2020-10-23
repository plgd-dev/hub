# 3. Resource directory

## Description

According to CQRS pattern it creates/updates projection for resource directory and resource shadow.

## API

All requests to service must contains valid access token in [grpc metadata](https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-auth-support.md#oauth2).

### Commands

- get devices - list devices
- get resource links - list resource links
- retrieve resource from device - get content from the device
- retrieve resources values - get resources from the resource shadow
- update resources values - update resource at the device
- subscribe for events - provides notification about device registered/unregistered/online/offline, resource published/unpublished/content changed/ ...
- get client configuration - provides public configuration for clients(mobile, web, onboarding tool)

### Contract

- [service](https://github.com/plgd-dev/cloud/blob/master/grpc-gateway/pb/service.proto)
- [requets/responses](https://github.com/plgd-dev/cloud/blob/master/grpc-gateway/pb/devices.proto)
- [client configuration](https://github.com/plgd-dev/cloud/blob/master/grpc-gateway/pb/clientConfiguration.proto)

## Docker Image

```bash
docker pull plgd/resource-directory:vnext
```

## Configuration

| Option | ENV variable | Type | Description | Default |
| ------ | --------- | ----------- | ------- | ------- |
| `-` | `ADDRESS` | string | `listen address` | `"0.0.0.0:9100"` |
| `-` | `AUTH_SERVER_ADDRESS` | string | `authoriztion server address` | `"127.0.0.1:9100"` |
| `-` | `RESOURCE_AGGREGATE_ADDRESS` | string | `resource aggregate address` | `"127.0.0.1:9100"` |
| `-` | `TIMEOUT_FOR_REQUESTS` | string | `wait for update/retrieve resource` | `10s` |
| `-` | `PROJECTION_CACHE_EXPIRATION` | string | `expiration time of projection` | `"30s"` |
| `-` | `JWKS_URL` | string | `url to get JSON Web Key` | `""` |
| `-` | `USER_MGMT_TICK_FREQUENCY` | string | `pull interval to refresh user devices` | `"15s"` |
| `-` | `USER_MGMT_EXPIRATION` | string | `expiration time of record about user devices` | `"1m"` |
| `-` | `SERVICE_CLIENT_CONFIGURATION_CLOUD_CA_POOL` | string | `path root CA which was used to signe coap-gw certificate` | `""` |
| `-` | `SERVICE_CLIENT_CONFIGURATION_ACCESSTOKENURL` | string | `url where user can get OAuth token via implicit flow` | `""` |
| `-` | `SERVICE_CLIENT_CONFIGURATION_AUTHCODEURL` | string | `url where user can get OAuth authorization code for the device` | `""` |
| `-` | `SERVICE_CLIENT_CONFIGURATION_CLOUDID` | string | `cloud id which is stored in coap-gw certificate` | `""` |
| `-` | `SERVICE_CLIENT_CONFIGURATION_CLOUDURL` | string | `cloud url for onboard device` | `""` |
| `-` | `SERVICE_CLIENT_CONFIGURATION_CLOUDAUTHORIZATIONPROVIDER` | string | `oauth authorization provider for onboard device` | `""` |
| `-` | `SERVICE_CLIENT_CONFIGURATION_SIGNINGSERVERADDRESS` | string | `address of ceritificate authority for plgd-dev/sdk` | `""` |  
| `-` | `SERVICE_OAUTH_ENDPOINT_TOKEN_URL` | string | `url to get service access token via OAUTH client credential flow` | `""` |
| `-` | `SERVICE_OAUTH_CLIENT_ID` | string | `client id for authentication to get access token` | `""` |
| `-` | `SERVICE_OAUTH_CLIENT_SECRET` | string | `secrest for authentication to get access token` | `""` |
| `-` | `SERVICE_OAUTH_AUDIENCE` | string | `refer to the resource servers that should accept the token` | `""` |
| `-` | `NATS_URL` | string | `url to nats messaging system` | `"nats://localhost:4222"` |
| `-` | `MONGODB_URI` | string | `uri to mongo database` | `"mongodb://localhost:27017"` |
| `-` | `MONGODB_DATABASE` | string | `name of database` | `"eventstore"` |
| `-` | `MONGODB_BATCH_SIZE` | int | `maximum number resources in one batch request`  | `16` |
| `-` | `MONGODB_MAX_POOL_SIZE` | int | `maximum parallel request to DB` | `16` |
| `-` | `MONGODB_MAX_CONN_IDLE_TIME` | string |  `maximum time of idle connection` | `"240s"` |
| `-` | `DIAL_TYPE` | string | `defines how to obtain dial TLS certificates - options: acme|file` | `"acme"` |
| `-` | `DIAL_ACME_CA_POOL` | string | `path to pem file of CAs` | `""` |
| `-` | `DIAL_ACME_DIRECTORY_URL` | string |  `url of acme directory` | `""` |
| `-` | `DIAL_ACME_DOMAINS` | string | `list of domains for which will be in certificate provided from acme` | `""` |
| `-` | `DIAL_ACME_REGISTRATION_EMAIL` | string | `registration email for acme` | `""` |
| `-` | `DIAL_ACME_TICK_FREQUENCY` | string | `interval of validate certificate` | `""` |
| `-` | `DIAL_ACME_USE_SYSTEM_CERTIFICATION_POOL` | bool | `load CAs from system` | `false` |
| `-` | `DIAL_FILE_CA_POOL` | string | tbd | `path to pem file of CAs` |
| `-` | `DIAL_FILE_CERT_KEY_NAME` | string | `name of pem certificate key file` | `""` |
| `-` | `DIAL_FILE_CERT_DIR_PATH` | string | `path to directory which contains DIAL_FILE_CERT_KEY_NAME and DIAL_FILE_CERT_NAME` | `""` |
| `-` | `DIAL_FILE_CERT_NAME` | string | `name of pem certificate file` | `""` |
| `-` | `DIAL_FILE_USE_SYSTEM_CERTIFICATION_POOL` | bool | `load CAs from system` | `false` |
| `-` | `LISTEN_TYPE` | string | `defines how to obtain listen TLS certificates - options: acme|file` | `"acme"` |
| `-` | `LISTEN_ACME_CA_POOL` | string | `path to pem file of CAs` | `""` |
| `-` | `LISTEN_ACME_DIRECTORY_URL` | string |  `url of acme directory` | `""` |
| `-` | `LISTEN_ACME_DOMAINS` | string | `list of domains for which will be in certificate provided from acme` | `""` |
| `-` | `LISTEN_ACME_REGISTRATION_EMAIL` | string | `registration email for acme` | `""` |
| `-` | `LISTEN_ACME_TICK_FREQUENCY` | string | `interval of validate certificate` | `""` |
| `-` | `LISTEN_ACME_USE_SYSTEM_CERTIFICATION_POOL` | bool | `load CAs from system` | `false` |
| `-` | `LISTEN_FILE_CA_POOL` | string | tbd | `path to pem file of CAs` |
| `-` | `LISTEN_FILE_CERT_KEY_NAME` | string | `name of pem certificate key file` | `""` |
| `-` | `LISTEN_FILE_CERT_DIR_PATH` | string | `path to directory which contains LISTEN_FILE_CERT_KEY_NAME and LISTEN_FILE_CERT_NAME` | `""` |
| `-` | `LISTEN_FILE_CERT_NAME` | string | `name of pem certificate file` | `""` |
| `-` | `LISTEN_FILE_USE_SYSTEM_CERTIFICATION_POOL` | bool | `load CAs from system` | `false` |
| `-` | `LOG_ENABLE_DEBUG` | bool | `debug logging` | `false` |

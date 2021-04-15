# Working with GRPC Client

## What it GRPC

Please follow [link](https://grpc.io/docs/what-is-grpc/introduction/)

## How to create GRPC client for grpc-gateway

For creating grpc-client you need to generate a code for your language from proto files, which are stored at [cloud](https://github.com/plgd-dev/cloud/tree/v2/grpc-gateway/pb). 

### API

All requests to service must contains valid access token in [grpc metadata](https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-auth-support.md#oauth2).

::: warning
Each request to the gRPC Gateway shall contain a valid access token as a part of [grpc metadata](https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-auth-support.md#oauth2).
:::

### Get Devices
The `GetDevices` command supports various filter options. If all of them are **unset**, all devices of a user identified by the access token are returned. 
Example usages of filter options:
    - to retrieve certain devices use `GetDevicesRequest.device_ids_filter` where ids of these devices needs to be set
    - to retrieve all offline devices set `GetDevicesRequest.status_filter` to `OFFLINE`
    - to retrieve all devices of certain types use `GetDevicesRequest.type_filter` (e.g. `x.com.plgd.light`)

To return only online devices with ids `deviceID1` and `deviceID2`, following options shall be set: `GetDevicesRequest.device_ids_filter("[deviceID1, deviceID2]") && GetDevicesRequest.status_filter([ONLINE])`.

### Get Resource Links
The `GetResourceLinks` command supports various filter options. If all of them are **unset**, all links of all devices user is authorized to use are returned. 
Example usages of filter options:
    - to retrieve links of certain devices use `GetResourceLinksRequest.device_ids_filter` where ids of these devices needs to be set
    - to retrieve links of certain types use `GetResourceLinksRequest.type_filter` (e.g. `oic.r.switch.binary`)

To return only binary switches resources hosted by devices with ids `deviceID1` and `deviceID2`, following options shall be set: `GetResourceLinksRequest.device_ids_filter("[deviceID1, deviceID2]") && GetResourceLinksRequest.type_filter([oic.r.switch.binary])`.

### Retrieve Resource Content
The `RetrieveResourcesValues` command supports various filter options. If all of them are **unset**, all resource contents of all devices user is authorized to use are returned. 
Example usages of filter options:
    - to retrieve of resources identified by their hrefs use `RetrieveResourcesValuesRequest.resource_ids_filter` where combinations `deviceID` and `href` needs to be set
    - to retrieve resource values of certain devices use `RetrieveResourcesValuesRequest.device_ids_filter` where ids of these devices needs to be set
    - to retrieve values from resources of a specific type use `RetrieveResourcesValuesRequest.type_filter`

To return values of binary switch resources hosted by devices with ids `deviceID1` and `deviceID2`, following options shall be set: `RetrieveResourcesValuesRequest.device_ids_filter("[deviceID1, deviceID2]") && RetrieveResourcesValuesRequest.type_filter([oic.r.switch.binary])`.

### Subscribe to Events
The `SubscribeForEvents` command opens the stream which content is controlled by sending messages with filter options.
To control what will be pushed to the stream, send a `SubscribeForEvents` message with option:
    - `filter_by.devices_event.filter_events` set to e.g. `ONLINE` to receive **devices events** which changed their status to `ONLINE`
    - `filter_by.device_event.{device_id, filter_events}` set to e.g. `RESOURCE_PUBLISHED` to receive **device events**
    - `filter_by.device_event.{resource_id.{device_id, href}, filter_events}` set to e.g. `CONTENT_CHANGED` to receive **resource events**

Frst event returned after successful subscription is of type `OperationProcessed`. Property `OperationProcessed.error_status.code` contains information if the subscription was successful. If it was successful, property `subscriptionId` is set. All events belonging to single `SubscribeForEvents` request are then identified by this `subscriptionId`.


If user losts a device (unregister, not more shared with the user), the client receives the event `SubscriptionCanceled` with corresponding `subcriptionId`.

### Retrieve Resource from Device
The `RetrieveResourceFromDevice` retrieves resource content directly from the device - resource shadow value is not returned.
> This command is expensive as it has to go synchronously directly to the device.

### Update Resource Content
The `UpdateResource` command requests resource update on the device.

### Create Resource
The `Create Resource` command requests creation of a new resource on a specific collection on the device.

### Delete Resource
The `DeleteResource` command requests device to delete a specific resource. Confirmation message doesn't mean that the resource was deleted. When the deletion was successfuly done by the device, event `RESOURCE_UNPUBLISHED`is send.

#### Contract

- [service](https://github.com/plgd-dev/cloud/blob/v2/grpc-gateway/pb/service.proto)
- [requests/responses](https://github.com/plgd-dev/cloud/blob/v2/grpc-gateway/pb/devices.proto)
- [client configuration](https://github.com/plgd-dev/cloud/blob/v2/grpc-gateway/pb/clientConfiguration.proto)

### Go-Lang GRPC client

### Creating client

Grpc-gateway uses TLS listener, so client must have properly configured TLS.  Here is simple example how to create a grpc client.

```go
import (
    "google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
    "github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/grpc-gateway/client"
)

    ...
    // Create TLS connection to the grpc-gateway.
    gwConn, err := grpc.Dial(
        address,
        grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
    )
    if err != nil {
        panic("cannot connect to grpc-gateway: " + err.Error())
    }
    // Create basic client which was generated from proto files.
    basicClient := pb.NewGrpcGatewayClient(gwConn)
    
    // Create Extended client which provide us more friendly functions.
    extendedClient := client.NewClient(basicClient)
    ...
```

### Using extended grpc client

More info in [doc](https://pkg.go.dev/github.com/plgd-dev/cloud/grpc-gateway/client)

# Working with GRPC Client

## What it GRPC

Please follow [link](https://grpc.io/docs/what-is-grpc/introduction/)

## How to create GRPC client for grpc-gateway

For creating grpc-client you need to generate a code for your language from proto files, which are stored at [cloud](https://github.com/plgd-dev/cloud/tree/v2/grpc-gateway/pb). 

### API

All requests to service must contains valid access token in [grpc metadata](https://github.com/grpc/grpc-go/blob/master/Documentation/grpc-auth-support.md#oauth2).

#### Commands

- get devices - list devices
- get resource links - list resource links
- create resource at device - create resource at the device
- retrieve resource from device - get content from the device
- retrieve resources values - get resources from the resource shadow
- update resources values - update resource at the device
- delete resource - delete resource at the device
- subscribe for events - provides notification about device registered/unregistered/online/offline, resource published/unpublished/content changed/ ...
- get client configuration - provides public configuration for clients(mobile, web, onboarding tool)

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

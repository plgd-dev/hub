# HTTP Gateway

- [Documentation](#documentation)
- [WebSocket](#websocket)

## Documentation
- [REST API](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/go-ocf/cloud/master/http-gateway/swagger.yaml) ([swagger](/http-gateway/swagger.yaml), [uri](/http-gateway/uri/uri.go), [mux](/http-gateway/service/httpApi.go))
- [WebSocket API](#websocket)

## WebSocket

HTTP Gateway provides WebSocket API for observations:

| URI                                                       | Description                                                     |
| ---                                                       | ---                                                             |
| `/api/ws/devices`                                         | [observe devices](/http-gateway/service/observeDevices_test.go)                   |
| `/api/ws/devices/`{DeviceID}                              | [observe device resources](/http-gateway/service/observeDeviceResources_test.go)  |
| `/api/ws/devices/`{DeviceID}{ResourceHref}                | [observe resource](/http-gateway/service/observeResource_test.go)                 |


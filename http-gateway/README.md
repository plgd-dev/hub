# HTTP Gateway

- [Documentation](#documentation)
- [WebSocket](#websocket)

## Documentation
- [REST API](https://petstore.swagger.io/?url=) ([swagger](/swagger.yaml), [uri](/uri/uri.go), [mux](/service/httpApi.go))
- [WebSocket API](#websocket)

## WebSocket

WebAPI provides WebSocket API for observations:

| URI                                                       | Description                                                     |
| ---                                                       | ---                                                             |
| `/api/ws/devices`                                         | [observe devices](/service/observeDevices_test.go)                   |
| `/api/ws/devices/`{DeviceID}                              | [observe device resources](/service/observeDeviceResources_test.go)  |
| `/api/ws/devices/`{DeviceID}{ResourceHref}                | [observe resource](/service/observeResource_test.go)                 |


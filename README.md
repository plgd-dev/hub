[![Build Status](https://travis-ci.com/go-ocf/cloud.svg?branch=master)](https://travis-ci.com/go-ocf/cloud)
[![codecov](https://codecov.io/gh/go-ocf/cloud/branch/master/graph/badge.svg)](https://codecov.io/gh/go-ocf/cloud)
[![Go Report](https://goreportcard.com/badge/github.com/go-ocf/cloud)](https://goreportcard.com/report/github.com/go-ocf/cloud)
[![Gitter](https://badges.gitter.im/ocfcloud/Lobby.svg)](https://gitter.im/ocfcloud/Lobby?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

# OCF Native Cloud

## Try
* [pluggedin.cloud](https://pluggedin.cloud)
* [single docker image](https://github.com/go-ocf/cloud/tree/master/bundle)

## Micro-services
* [coap-gateway](https://github.com/go-ocf/cloud/tree/master/coap-gateway) provides gateway for [**Device**](https://github.com/iotivity/iotivity-lite)
* [grpc-gateway](https://github.com/go-ocf/cloud/tree/master/grpc-gateway) provides gateway for **Service**
* [http-gateway](https://github.com/go-ocf/cloud/tree/master/http-gateway) provides gateway for **GUI**
* [cloud2cloud-gateway](https://github.com/go-ocf/cloud/tree/master/cloud2cloud-gateway) provides gateway for **Cloud**
* [cloud2cloud-connector](https://github.com/go-ocf/cloud/tree/master/cloud2cloud-gateway) provides connector to **Cloud**
* ...

## Features
* all services are scalable
* internal communication is secured using GRPC
* supports ACME protocol
* following [CQRS pattern](https://leanpub.com/esversioning/read)
* test [cloud](https://github.com/go-ocf/cloud/tree/master/bundle) in a single docker image
* ...

## License
Apache 2.0

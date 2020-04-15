[![Build Status](https://travis-ci.com/go-ocf/cloud.svg?branch=master)](https://travis-ci.com/go-ocf/cloud)
[![codecov](https://codecov.io/gh/go-ocf/cloud/branch/master/graph/badge.svg)](https://codecov.io/gh/go-ocf/cloud)
[![Go Report](https://goreportcard.com/badge/github.com/go-ocf/cloud)](https://goreportcard.com/report/github.com/go-ocf/cloud)
[![Gitter](https://badges.gitter.im/ocfcloud/Lobby.svg)](https://gitter.im/ocfcloud/Lobby?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

# OCF Native Cloud

Cloud-native means, simply, that applications are designed to be deployed in the cloud from the beginning, easing horizontal scalability.
Applications are composed of dozens of micro-services, each expressed as a RESTful API, whose provisioning code is in a software container, and whose lifetime is limited to the interaction with a single client

## Micros-services
* [coap-gateway](https://github.com/go-ocf/cloud/tree/master/coap-gateway) provides gateway for [**a Device**](https://github.com/iotivity/iotivity-lite)
* [grpc-gateway](https://github.com/go-ocf/cloud/tree/master/grpc-gateway) provides gateway for **a Service**
* [http-gateway](https://github.com/go-ocf/cloud/tree/master/http-gateway) provides gateway for **a GUI**
* [cloud2cloud-gateway](https://github.com/go-ocf/cloud/tree/master/cloud2cloud-gateway) provides gateway for **a Cloud**
* [cloud2cloud-connector](https://github.com/go-ocf/cloud/tree/master/cloud2cloud-gateway) provides connector to **a Cloud**
* ...

## Features
* all micro-services are scalable
* internal communication goes through grpc with mTLS
* supporting ACME protocol
* using [CQRS pattern](https://leanpub.com/esversioning/read)
* test [cloud](https://github.com/go-ocf/cloud/tree/master/bundle) in a single docker image
* ...

## License
Apache 2.0

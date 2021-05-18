module github.com/plgd-dev/cloud

go 1.14

require (
	github.com/buaazp/fasthttprouter v0.1.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gofrs/uuid v3.4.0+incompatible
	github.com/golang/snappy v0.0.2
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/uuid v1.1.2
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/jessevdk/go-flags v1.4.0
	github.com/jtacoma/uritemplates v1.0.0
	github.com/karrick/tparse/v2 v2.8.2
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/lestrrat-go/jwx v1.0.5
	github.com/nats-io/nats.go v1.10.1-0.20201111151633-9e1f4a0d80d8
	github.com/panjf2000/ants/v2 v2.4.3
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/plgd-dev/go-coap/v2 v2.4.1-0.20210517130748-95c37ac8e1fa
	github.com/plgd-dev/kit v0.0.0-20210517131053-7dfd49bb6277
	github.com/plgd-dev/sdk v0.0.0-20210517131411-530870e2d96d
	github.com/stretchr/testify v1.7.0
	github.com/valyala/fasthttp v1.16.0
	go.mongodb.org/mongo-driver v1.4.2
	go.uber.org/atomic v1.7.0
	go.uber.org/zap v1.15.0
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43
	golang.org/x/sync v0.0.0-20201008141435-b3e1573b7520
	google.golang.org/genproto v0.0.0-20200825200019-8632dd797987
	google.golang.org/grpc v1.34.0
	google.golang.org/grpc/examples v0.0.0-20210129004707-0bc741730b81 // indirect
	google.golang.org/protobuf v1.25.0
	gopkg.in/yaml.v2 v2.3.0
)

replace gopkg.in/yaml.v2 v2.3.0 => github.com/cizmazia/yaml v0.0.0-20200220134304-2008791f5454

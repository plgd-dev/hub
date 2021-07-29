module github.com/plgd-dev/cloud

go 1.16

require (
	github.com/buaazp/fasthttprouter v0.1.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gofrs/uuid v4.0.0+incompatible
	github.com/golang/snappy v0.0.4
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.1.0
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.5.0
	github.com/jessevdk/go-flags v1.5.0
	github.com/json-iterator/go v1.1.11
	github.com/jtacoma/uritemplates v1.0.0
	github.com/karrick/tparse/v2 v2.8.2
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/lestrrat-go/jwx v1.2.1
	github.com/nats-io/nats.go v1.11.0
	github.com/panjf2000/ants/v2 v2.4.6
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/plgd-dev/go-coap/v2 v2.4.1-0.20210720210042-9b9018ae0577
	github.com/plgd-dev/kit v0.0.0-20210614190235-99984a49de48
	github.com/plgd-dev/sdk v0.0.0-20210726094740-ef7dbae58d81
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.7.0
	github.com/valyala/fasthttp v1.28.0
	go.mongodb.org/mongo-driver v1.7.0
	go.uber.org/atomic v1.8.0
	go.uber.org/zap v1.18.1
	golang.org/x/net v0.0.0-20210510120150-4163338589ed
	golang.org/x/oauth2 v0.0.0-20210628180205-a41e5a781914
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	google.golang.org/genproto v0.0.0-20210707164411-8c882eb9abba
	google.golang.org/grpc v1.39.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.1.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/yaml.v2 v2.4.0
)

replace gopkg.in/yaml.v2 v2.4.0 => github.com/cizmazia/yaml v0.0.0-20200220134304-2008791f5454

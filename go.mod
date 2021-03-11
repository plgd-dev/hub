module github.com/plgd-dev/cloud

go 1.14

require (
	github.com/buaazp/fasthttprouter v0.1.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/go-chi/chi v4.1.2+incompatible
	github.com/gofrs/uuid v3.3.0+incompatible
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
	github.com/plgd-dev/go-coap/v2 v2.3.1-0.20210204203130-38a043760811
	github.com/plgd-dev/kit v0.0.0-20210205135220-6c16c4ac0fd5
	github.com/plgd-dev/sdk v0.0.0-20201105135357-8507ce8ec280
	github.com/smallstep/certificates v0.13.4-0.20191007194430-e2858e17b094
	github.com/smallstep/nosql v0.2.0
	github.com/stretchr/testify v1.6.1
	github.com/valyala/fasthttp v1.16.0
	go.mongodb.org/mongo-driver v1.4.2
	go.uber.org/atomic v1.7.0
	go.uber.org/zap v1.15.0
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43
	golang.org/x/sync v0.0.0-20201008141435-b3e1573b7520
	google.golang.org/grpc v1.34.0
	google.golang.org/grpc/examples v0.0.0-20210129004707-0bc741730b81 // indirect
	google.golang.org/protobuf v1.25.0
	gopkg.in/yaml.v2 v2.3.0
)

replace gopkg.in/yaml.v2 v2.3.0 => github.com/cizmazia/yaml v0.0.0-20200220134304-2008791f5454

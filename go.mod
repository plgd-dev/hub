module github.com/go-ocf/cloud

go 1.14

require (
	github.com/awalterschulze/goderive v0.0.0-20200222153121-9a5b9356be09 // indirect
	github.com/buaazp/fasthttprouter v0.1.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/fullstorydev/grpchan v1.0.1
	github.com/go-chi/chi v4.1.1+incompatible
	github.com/go-ocf/cqrs v0.0.0-20200324131357-db8a7b8c83be
	github.com/go-ocf/go-coap/v2 v2.0.0
	github.com/go-ocf/kit 01631a881369
	github.com/go-ocf/sdk v0.0.0-20200610191654-01cea092557e
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.4.2
	github.com/golang/snappy v0.0.2-0.20190904063534-ff6b7dc882cf
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.4
	github.com/gorilla/websocket v1.4.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.0
	github.com/jessevdk/go-flags v1.4.0
	github.com/jhump/goprotoc v0.2.0 // indirect
	github.com/jhump/protoreflect v1.6.1 // indirect
	github.com/jtacoma/uritemplates v1.0.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/lestrrat-go/jwx v1.0.2
	github.com/nats-io/nats.go v1.9.2
	github.com/panjf2000/ants v1.3.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/satori/go.uuid v1.2.0
	github.com/smallstep/certificates v0.13.4-0.20191007194430-e2858e17b094
	github.com/smallstep/nosql v0.2.0
	github.com/stretchr/testify v1.5.1
	github.com/ugorji/go/codec v1.1.7
	github.com/valyala/fasthttp v1.12.0
	go.mongodb.org/mongo-driver v1.3.2
	go.uber.org/atomic v1.6.0
	go.uber.org/zap v1.15.0
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a
	golang.org/x/tools v0.0.0-20200603170713-0310561d584d // indirect
	google.golang.org/grpc v1.29.1
	google.golang.org/protobuf v1.24.0 // indirect
	gopkg.in/yaml.v2 v2.2.8
)

replace gopkg.in/yaml.v2 v2.2.8 => github.com/cizmazia/yaml v0.0.0-20200220134304-2008791f5454

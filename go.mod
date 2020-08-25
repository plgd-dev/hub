module github.com/go-ocf/cloud

go 1.13

require (
	github.com/buaazp/fasthttprouter v0.1.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/fullstorydev/grpchan v1.0.1
	github.com/go-chi/chi v4.1.0+incompatible
	github.com/go-ocf/cqrs v0.0.0-20200324131357-db8a7b8c83be
	github.com/go-ocf/go-coap v0.0.0-20200406073902-cf923db524db
	github.com/go-ocf/kit v0.0.0-20200728130040-4aebdb6982bc
	github.com/go-ocf/sdk v0.0.0-20200409112144-f306e826ae33
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.4.1
	github.com/golang/snappy v0.0.2-0.20190904063534-ff6b7dc882cf
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.4
	github.com/gorilla/websocket v1.4.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.0
	github.com/jessevdk/go-flags v1.4.0
	github.com/jtacoma/uritemplates v1.0.0
	github.com/karrick/tparse/v2 v2.8.2
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/lestrrat-go/jwx v1.0.2
	github.com/nats-io/nats.go v1.9.2
	github.com/panjf2000/ants v1.3.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/plgd-dev/kit v0.0.0-20200825124924-f07b62fe8d61
	github.com/satori/go.uuid v1.2.0
	github.com/smallstep/certificates v0.13.4-0.20191007194430-e2858e17b094
	github.com/smallstep/nosql v0.2.0
	github.com/stretchr/testify v1.5.1
	github.com/ugorji/go/codec v1.1.7
	github.com/valyala/fasthttp v1.12.0
	go.mongodb.org/mongo-driver v1.3.2
	go.uber.org/atomic v1.6.0
	golang.org/x/net v0.0.0-20200506145744-7e3656a0809f
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a
	google.golang.org/grpc v1.29.1
	gopkg.in/yaml.v2 v2.2.8
)

replace gopkg.in/yaml.v2 v2.2.8 => github.com/cizmazia/yaml v0.0.0-20200220134304-2008791f5454

module github.com/plgd-dev/cloud

go 1.14

require (
	github.com/andybalholm/brotli v1.0.1 // indirect
	github.com/buaazp/fasthttprouter v0.1.1
	github.com/dgraph-io/badger v1.6.2 // indirect
	github.com/dgraph-io/ristretto v0.0.3-0.20200630154024-f66de99634de // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/fullstorydev/grpchan v1.0.1
	github.com/go-chi/chi v4.1.2+incompatible
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.4.2
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
	github.com/klauspost/compress v1.11.1 // indirect
	github.com/lestrrat-go/jwx v1.0.5
	github.com/nats-io/jwt v1.0.1 // indirect
	github.com/nats-io/nats.go v1.10.0
	github.com/nats-io/nkeys v0.2.0 // indirect
	github.com/panjf2000/ants/v2 v2.4.2
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/plgd-dev/cqrs v0.0.0-20200819120550-a6063edff616
	github.com/plgd-dev/go-coap/v2 v2.0.5-0.20200922125453-917a9b325e68
	github.com/plgd-dev/kit v0.0.0-20201002113054-9f5d47d18a8c
	github.com/plgd-dev/sdk v0.0.0-20201008144930-a520d56a75b7
	github.com/satori/go.uuid v1.2.0
	github.com/smallstep/certificates v0.13.4-0.20191007194430-e2858e17b094
	github.com/smallstep/nosql v0.2.0
	github.com/stretchr/testify v1.6.1
	github.com/ugorji/go/codec v1.1.10
	github.com/valyala/fasthttp v1.16.0
	go.mongodb.org/mongo-driver v1.4.2
	go.uber.org/atomic v1.7.0
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0 // indirect
	golang.org/x/net v0.0.0-20201006153459-a7d1128ccaa0
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43
	golang.org/x/sync v0.0.0-20201008141435-b3e1573b7520
	golang.org/x/sys v0.0.0-20201008064518-c1f3e3309c71 // indirect
	google.golang.org/genproto v0.0.0-20201008135153-289734e2e40c // indirect
	google.golang.org/grpc v1.32.0
	google.golang.org/grpc/examples v0.0.0-20201007173958-5af60402cd8c // indirect
	gopkg.in/yaml.v2 v2.2.8
)

replace gopkg.in/yaml.v2 v2.2.8 => github.com/cizmazia/yaml v0.0.0-20200220134304-2008791f5454

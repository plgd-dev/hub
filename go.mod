module github.com/plgd-dev/hub/v2

go 1.22

toolchain go1.22.0

require (
	github.com/favadi/protoc-go-inject-tag v1.4.0
	github.com/felixge/httpsnoop v1.0.4
	github.com/fsnotify/fsnotify v1.7.0
	github.com/fullstorydev/grpchan v1.1.1
	github.com/fxamacker/cbor/v2 v2.6.0
	github.com/go-co-op/gocron/v2 v2.3.0
	github.com/gocql/gocql v1.6.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/golang/snappy v0.0.4
	github.com/google/go-querystring v1.1.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.1
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.1
	github.com/hashicorp/go-multierror v1.1.1
	github.com/jessevdk/go-flags v1.5.0
	github.com/json-iterator/go v1.1.12
	github.com/jtacoma/uritemplates v1.0.0
	github.com/karrick/tparse/v2 v2.8.2
	github.com/lestrrat-go/jwx/v2 v2.0.21
	github.com/nats-io/nats.go v1.34.1
	github.com/panjf2000/ants/v2 v2.9.1
	github.com/pion/dtls/v2 v2.2.8-0.20240501061905-2c36d63320a0
	github.com/pion/logging v0.2.2
	github.com/plgd-dev/device/v2 v2.5.1-0.20240513064831-b553d1a87e1c
	github.com/plgd-dev/go-coap/v3 v3.3.4
	github.com/plgd-dev/kit/v2 v2.0.0-20211006190727-057b33161b90
	github.com/pseudomuto/protoc-gen-doc v1.5.1
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/testify v1.9.0
	github.com/tidwall/gjson v1.17.1
	github.com/tidwall/sjson v1.2.5
	github.com/ugorji/go/codec v1.2.12
	github.com/vincent-petithory/dataurl v1.0.0
	github.com/web-of-things-open-source/thingdescription-go v0.0.0-20240510130416-741fef736e1e
	go.mongodb.org/mongo-driver v1.15.0
	go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo v0.49.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.49.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0
	go.opentelemetry.io/otel v1.24.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.24.0
	go.opentelemetry.io/otel/metric v1.24.0
	go.opentelemetry.io/otel/sdk v1.24.0
	go.opentelemetry.io/otel/trace v1.24.0
	go.uber.org/atomic v1.11.0
	go.uber.org/zap v1.27.0
	golang.org/x/exp v0.0.0-20240416160154-fe59bbe5cc7f
	golang.org/x/net v0.24.0
	golang.org/x/oauth2 v0.19.0
	golang.org/x/sync v0.7.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240429193739-8cf5692501f6
	google.golang.org/grpc v1.63.2
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.3.0
	google.golang.org/protobuf v1.34.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cloud.google.com/go/compute v1.25.1 // indirect
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/Masterminds/sprig v2.15.0+incompatible // indirect
	github.com/aokoli/goutils v1.0.1 // indirect
	github.com/bufbuild/protocompile v0.13.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.3.0 // indirect
	github.com/dsnet/golib/memfile v1.0.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.0.4 // indirect
	github.com/fredbi/uri v1.1.0 // indirect
	github.com/go-json-experiment/json v0.0.0-20240418180308-af2d5061e6c2 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.1.0 // indirect
	github.com/hailocab/go-hostpool v0.0.0-20160125115350-e80d13ce29ed // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/huandu/xstrings v1.0.0 // indirect
	github.com/imdario/mergo v0.3.4 // indirect
	github.com/jhump/protoreflect v1.16.0 // indirect
	github.com/jonboulle/clockwork v0.4.0 // indirect
	github.com/klauspost/compress v1.17.8 // indirect
	github.com/lestrrat-go/blackmagic v1.0.2 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/httprc v1.0.5 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/mwitkow/go-proto-validators v0.0.0-20180403085117-0950a7990007 // indirect
	github.com/nats-io/nkeys v0.4.7 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pion/transport/v3 v3.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/pseudomuto/protokit v0.2.0 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.24.0 // indirect
	go.opentelemetry.io/proto/otlp v1.2.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.22.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240429193739-8cf5692501f6 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
)

replace (
	// note: github.com/pion/dtls/v2/pkg/net package is not yet available in release branches,
	// so we force to the use of the pinned master branch
	github.com/pion/dtls/v2 => github.com/pion/dtls/v2 v2.2.8-0.20240501061905-2c36d63320a0
	// later versions require go 1.22
	github.com/youmark/pkcs8 => github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a
	// later versions require go 1.21
	go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo => go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo v0.49.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc => go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.49.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0
	go.opentelemetry.io/otel => go.opentelemetry.io/otel v1.24.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc => go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.24.0
	go.opentelemetry.io/otel/metric => go.opentelemetry.io/otel/metric v1.24.0
	go.opentelemetry.io/otel/sdk => go.opentelemetry.io/otel/sdk v1.24.0
	go.opentelemetry.io/otel/trace => go.opentelemetry.io/otel/trace v1.24.0
)

FROM golang:1.22.3-alpine AS build
ARG DIRECTORY
ARG NAME
ARG VERSION
ARG COMMIT_DATE
ARG SHORT_COMMIT
ARG DATE
ARG RELEASE_URL
RUN apk add --no-cache build-base curl git
WORKDIR $GOPATH/src/github.com/plgd-dev/hub
COPY go.mod go.sum ./
RUN go mod download
COPY . .
WORKDIR /usr/local/go
RUN ( patch -p1 < "$GOPATH/src/github.com/plgd-dev/hub/tools/docker/patches/shrink_tls_conn.patch" )
WORKDIR $GOPATH/src/github.com/plgd-dev/hub/tools/grpc-reflection
RUN CGO_ENABLED=0 go build \
    -ldflags "-X github.com/plgd-dev/hub/v2/pkg/build.CommitDate=$COMMIT_DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.CommitHash=$SHORT_COMMIT \
    -X github.com/plgd-dev/hub/v2/pkg/build.BuildDate=$DATE \
    -X github.com/plgd-dev/hub/v2/pkg/build.Version=$VERSION \
    -X github.com/plgd-dev/hub/v2/pkg/build.ReleaseURL=$RELEASE_URL" \
    -o /go/bin/grpc-reflection \
    ./cmd/service

FROM alpine:3.19 AS security-provider
RUN addgroup -S nonroot \
    && adduser -S nonroot -G nonroot

FROM scratch AS service
COPY --from=security-provider /etc/passwd /etc/passwd
COPY --from=build /go/bin/grpc-reflection /usr/local/bin/grpc-reflection
USER nonroot
ENTRYPOINT [ "/usr/local/bin/grpc-reflection" ]

FROM golang:1.20.14-alpine AS build
ARG DIRECTORY
ARG NAME
ARG VERSION
ARG COMMIT_DATE
ARG SHORT_COMMIT
ARG DATE
ARG RELEASE_URL

# Install necessary packages
RUN apk add --no-cache curl git build-base

# Set the working directory and copy go.mod and go.sum
WORKDIR $GOPATH/src/github.com/plgd-dev/hub
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy all source files to the working directory
COPY . .

# Apply the patch
WORKDIR /usr/local/go
RUN ( patch -p1 < "$GOPATH/src/github.com/plgd-dev/hub/tools/docker/patches/shrink_tls_conn.patch" )

# Ensure the grpc-reflection directory exists and copy the grpc-reflection specific files
WORKDIR $GOPATH/src/github.com/plgd-dev/hub/tools/grpc-reflection
COPY ./tools/grpc-reflection/* .

# Debugging step: list files in the grpc-reflection directory
RUN echo "Contents of grpc-reflection:" && ls -al

# Build the grpc-reflection binary
RUN CGO_ENABLED=0 go build \
    -ldflags "-X github.com/plgd-dev/hub/v2/pkg/build.CommitDate=$COMMIT_DATE -X github.com/plgd-dev/hub/v2/pkg/build.CommitHash=$SHORT_COMMIT -X github.com/plgd-dev/hub/v2/pkg/build.BuildDate=$DATE -X github.com/plgd-dev/hub/v2/pkg/build.Version=$VERSION -X github.com/plgd-dev/hub/v2/pkg/build.ReleaseURL=$RELEASE_URL" \
    -o /go/bin/grpc-reflection \
    ./

# Create a security provider image
FROM alpine:3.19 AS security-provider
RUN addgroup -S nonroot \
    && adduser -S nonroot -G nonroot

# Create the final scratch image
FROM scratch AS service
COPY --from=security-provider /etc/passwd /etc/passwd
COPY --from=build /go/bin/grpc-reflection /usr/local/bin/grpc-reflection
USER nonroot
ENTRYPOINT [ "/usr/local/bin/grpc-reflection" ]



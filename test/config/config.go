package config

import (
	"time"

	"github.com/go-ocf/cloud/authorization/uri"
)

const AUTH_HOST = "localhost:7005"
const AUTH_HTTP_HOST = "localhost:7006"
const GW_HOST = "localhost:55684"
const RESOURCE_AGGREGATE_HOST = "localhost:9083"
const RESOURCE_DIRECTORY_HOST = "localhost:9082"
const GRPC_HOST = "localhost:9086"
const TEST_TIMEOUT = time.Second * 15
const OAUTH_MANAGER_CLIENT_ID = "service"

var OAUTH_MANAGER_ENDPOINT_TOKENURL = "https://" + AUTH_HTTP_HOST + uri.AccessToken
var JWKS_URL = "https://" + AUTH_HTTP_HOST + uri.JWKs

package config

import (
	"time"

	"github.com/go-ocf/cloud/authorization/uri"
	c2curi "github.com/go-ocf/cloud/cloud2cloud-connector/uri"
)

const AUTH_HOST = "localhost:20000"
const AUTH_HTTP_HOST = "localhost:20001"
const GW_HOST = "localhost:20002"
const RESOURCE_AGGREGATE_HOST = "localhost:20003"
const RESOURCE_DIRECTORY_HOST = "localhost:20004"
const GRPC_HOST = "localhost:20005"
const C2C_CONNECTOR_HOST = "localhost:20006"
const C2C_GW_HOST = "localhost:20007"
const TEST_TIMEOUT = time.Second * 20
const OAUTH_MANAGER_CLIENT_ID = "service"

var OAUTH_MANAGER_ENDPOINT_AUTHURL = "https://" + AUTH_HTTP_HOST + uri.AuthorizationCode
var OAUTH_MANAGER_ENDPOINT_TOKENURL = "https://" + AUTH_HTTP_HOST + uri.AccessToken
var C2C_CONNECTOR_EVENTS_URL = "https://" + C2C_CONNECTOR_HOST + c2curi.NotifyLinkedAccount
var C2C_CONNECTOR_OAUTH_CALLBACK = "https://" + C2C_CONNECTOR_HOST + "/oauthCbk"
var JWKS_URL = "https://" + AUTH_HTTP_HOST + uri.JWKs

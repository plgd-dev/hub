package config

import (
	"os"
	"time"

	c2curi "github.com/plgd-dev/cloud/cloud2cloud-connector/uri"
	"github.com/plgd-dev/cloud/test/oauth-server/service"
	"github.com/plgd-dev/cloud/test/oauth-server/uri"
)

const AUTH_HOST = "localhost:20000"
const AUTH_HTTP_HOST = "localhost:20001"
const GW_HOST = "localhost:20002"
const RESOURCE_AGGREGATE_HOST = "localhost:20003"
const RESOURCE_DIRECTORY_HOST = "localhost:20004"
const GRPC_HOST = "localhost:20005"
const C2C_CONNECTOR_HOST = "localhost:20006"
const C2C_GW_HOST = "localhost:20007"
const GW_UNSECURE_HOST = "localhost:20008"
const OAUTH_SERVER_HOST = "localhost:20009"
const TEST_TIMEOUT = time.Second * 20
const OAUTH_MANAGER_CLIENT_ID = service.ClientTest
const OAUTH_MANAGER_AUDIENCE = "localhost"

var CA_POOL = os.Getenv("LISTEN_FILE_CA_POOL")
var KEY_FILE = os.Getenv("LISTEN_FILE_CERT_DIR_PATH") + "/" + os.Getenv("LISTEN_FILE_CERT_KEY_NAME")
var CERT_FILE = os.Getenv("LISTEN_FILE_CERT_DIR_PATH") + "/" + os.Getenv("LISTEN_FILE_CERT_NAME")
var MONGODB_URI = "mongodb://localhost:27017"

var OAUTH_MANAGER_ENDPOINT_AUTHURL = "https://" + OAUTH_SERVER_HOST + uri.Authorize
var OAUTH_MANAGER_ENDPOINT_TOKENURL = "https://" + OAUTH_SERVER_HOST + uri.Token
var C2C_CONNECTOR_EVENTS_URL = "https://" + C2C_CONNECTOR_HOST + c2curi.Events
var C2C_CONNECTOR_OAUTH_CALLBACK = "https://" + C2C_CONNECTOR_HOST + "/oauthCbk"
var JWKS_URL = "https://" + OAUTH_SERVER_HOST + uri.JWKs

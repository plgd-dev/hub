package test

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/store"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/store/mongodb"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/uri"
	c2cGwUri "github.com/plgd-dev/hub/v2/cloud2cloud-gateway/uri"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	cmClient "github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2/oauth"
	"github.com/plgd-dev/hub/v2/test"
	testCfg "github.com/plgd-dev/hub/v2/test/config"
	testHttp "github.com/plgd-dev/hub/v2/test/http"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func countOpenFiles() int64 {
	out, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("lsof -p %v", os.Getpid())).Output()
	if err != nil {
		fmt.Println(err.Error())
	}
	lines := strings.Split(string(out), "\n")
	return int64(len(lines) - 1)
}

func SetUpClouds(ctx context.Context, t *testing.T, deviceID string, supportedEvents store.Events, switchIDs ...string) func() {
	cloud1 := service.SetUp(ctx, t)
	cloud2 := SetUpCloudWithConnector(t)
	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	cloud1Conn, err := grpc.Dial(testCfg.GRPC_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	c1 := pb.NewGrpcGatewayClient(cloud1Conn)
	_, shutdownDevSim := test.OnboardDevSim(ctx, t, c1, deviceID, testCfg.GW_HOST, test.GetAllBackendResourceLinks())
	if len(switchIDs) > 0 {
		test.AddDeviceSwitchResources(ctx, t, deviceID, c1, switchIDs...)
	}

	rootCAs := make([]string, 0, 1)
	certs := test.GetRootCertificateAuthorities(t)
	for _, c := range certs {
		rootCAs = append(rootCAs, string(pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: c.Raw,
		})))
	}

	linkedCloud := store.LinkedCloud{
		Name: t.Name(),
		Endpoint: store.Endpoint{
			URL:     testHttp.HTTPS_SCHEME + testCfg.C2C_GW_HOST + c2cGwUri.Version,
			RootCAs: rootCAs,
		},
		OAuth: oauth.Config{
			ClientID:     oauthTest.ClientTest,
			Audience:     testCfg.C2C_GW_HOST,
			ClientSecret: "testClientSecret",
			AuthURL:      testCfg.OAUTH_MANAGER_ENDPOINT_AUTHURL,
			TokenURL:     testCfg.OAUTH_MANAGER_ENDPOINT_TOKENURL,
			Scopes:       []string{"r:*", "w:*"},
		},
		SupportedSubscriptionEvents: supportedEvents,
	}
	data, err := json.Encode(linkedCloud)
	require.NoError(t, err)

	token := oauthTest.GetAccessToken(t, OAUTH_HOST, oauthTest.ClientTest)
	req := testHttp.NewHTTPRequest(http.MethodPost, testHttp.HTTPS_SCHEME+C2C_CONNECTOR_HOST+uri.LinkedClouds, bytes.NewBuffer(data)).AuthToken(token).Build(ctx, t)
	resp := testHttp.DoHTTPRequest(t, req)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer func(r *http.Response) {
		_ = r.Body.Close()
	}(resp)
	var linkCloud store.LinkedCloud
	err = json.ReadFrom(resp.Body, &linkCloud)
	require.NoError(t, err)
	req = testHttp.NewHTTPRequest(http.MethodGet, testHttp.HTTPS_SCHEME+C2C_CONNECTOR_HOST+uri.Version+"/clouds/"+linkCloud.ID+"/accounts", nil).AuthToken(token).Build(ctx, t)
	resp = testHttp.DoHTTPRequest(t, req)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer func(r *http.Response) {
		_ = r.Body.Close()
	}(resp)

	// for pulling
	time.Sleep(time.Second * 10)

	req = testHttp.NewHTTPRequest(http.MethodGet, testHttp.HTTPS_SCHEME+C2C_CONNECTOR_HOST+uri.Version+"/clouds", nil).AuthToken(token).Build(ctx, t)
	resp = testHttp.DoHTTPRequest(t, req)
	defer func(r *http.Response) {
		_ = r.Body.Close()
	}(resp)
	b, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	fmt.Println(string(b))

	return func() {
		req := testHttp.NewHTTPRequest(http.MethodDelete, testHttp.HTTPS_SCHEME+C2C_CONNECTOR_HOST+uri.Version+"/clouds/"+linkCloud.ID, nil).AuthToken(token).Build(ctx, t)
		resp := testHttp.DoHTTPRequest(t, req)
		defer func(r *http.Response) {
			_ = r.Body.Close()
		}(resp)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		cloud2()
		shutdownDevSim()
		err := cloud1Conn.Close()
		require.NoError(t, err)
		cloud1()
		runtime.GC()

		fmt.Printf("NUM FDS used %v\n", countOpenFiles())
	}
}

func NewMongoStore(t *testing.T) (*mongodb.Store, func()) {
	cfg := MakeConfig(t)

	logger := log.NewLogger(cfg.Log)

	certManager, err := cmClient.New(cfg.Clients.Storage.MongoDB.TLS, logger)
	require.NoError(t, err)

	ctx := context.Background()
	store, err := mongodb.NewStore(ctx, cfg.Clients.Storage.MongoDB, certManager.GetTLSConfig())
	require.NoError(t, err)

	cleanUp := func() {
		err := store.Clear(ctx)
		require.NoError(t, err)
		_ = store.Close(ctx)
		certManager.Close()
	}

	return store, cleanUp
}

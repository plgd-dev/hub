package service_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/schema/csr"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/options"
	"github.com/plgd-dev/go-coap/v3/tcp"
	caService "github.com/plgd-dev/hub/v2/certificate-authority/test"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/service"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/uri"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	pkgCoapService "github.com/plgd-dev/hub/v2/pkg/net/coap/service"
	hubTestService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/plgd-dev/kit/v2/security/generateCertificate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func connect(t *testing.T, dpsCfg service.Config) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	// create connection via mfg certificate
	c, err := tcp.Dial(dpsCfg.APIs.COAP.Addr, options.WithTLS(setupTLSConfig(t)), options.WithContext(ctx))
	require.NoError(t, err)
	defer func() {
		errC := c.Close()
		require.NoError(t, errC)
		<-c.Done()
	}()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	deviceID, err := uuid.NewRandom()
	require.NoError(t, err)

	csrReq, err := generateCertificate.GenerateIdentityCSR(generateCertificate.Configuration{}, deviceID.String(), priv)
	require.NoError(t, err)

	req := service.CredentialsRequest{
		CSR: service.CSR{
			Encoding: csr.CertificateEncoding_PEM,
			Data:     string(csrReq),
		},
	}

	resp, err := c.Post(ctx, uri.Credentials, message.AppOcfCbor, bytes.NewReader(toCbor(t, req)))
	require.NoError(t, err)

	if resp.Code() != codes.Changed {
		fmt.Printf("resp: %+v\n", resp)
		return
	}

	_, err = c.Post(ctx, uri.CloudConfiguration, message.AppOcfCbor, bytes.NewReader(toCbor(t, service.ProvisionCloudConfigurationRequest{DeviceID: uuid.NewString()})))
	require.NoError(t, err)
}

func testNConnections(t *testing.T, n int, debugging bool) {
	go func() {
		server := &http.Server{
			Addr:         "localhost:8080",
			ReadTimeout:  time.Hour,
			WriteTimeout: time.Hour,
		}
		err := server.ListenAndServe()
		assert.NoError(t, err)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer test.ClearDB(t)
	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesResourceDirectory|hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesOAuth|hubTestService.SetUpServicesId)
	defer hubShutdown()
	caCfg := caService.MakeConfig(t)
	caCfg.Signer.ExpiresIn = time.Hour * 10
	caShutdown := caService.New(t, caCfg)
	defer caShutdown()

	psk := testPSK
	pskFile := writeToTempFile(t, "psk1.key", []byte(psk))
	defer func() {
		err := os.Remove(pskFile)
		require.NoError(t, err)
	}()

	dpsCfg := test.MakeConfig(t)
	dpsCfg.APIs.COAP.InactivityMonitor = &pkgCoapService.InactivityMonitor{
		Timeout: time.Second,
	}
	dpsCfg.EnrollmentGroups[0].PreSharedKeyFile = urischeme.URIScheme(pskFile)
	shutDown := test.NewWithContext(ctx, t, dpsCfg)
	defer shutDown()

	if debugging {
		fmt.Printf("start collecting heap\n")
		time.Sleep(time.Second * 10)
	}

	for i := range n {
		fmt.Printf("connect %d\n", i)
		connect(t, dpsCfg)
	}

	for i := range 3 {
		fmt.Printf("gc %d\n", i)
		time.Sleep(time.Second)
		runtime.GC()
	}

	if debugging {
		fmt.Printf("done\n")
		time.Sleep(time.Second * 3600)
	}
}

func TestNConnections(t *testing.T) {
	n := 10
	debugging := false
	testNConnections(t, n, debugging)
}

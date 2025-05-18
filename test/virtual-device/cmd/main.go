package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/net/blockwise"
	"github.com/plgd-dev/go-coap/v3/net/responsewriter"
	"github.com/plgd-dev/go-coap/v3/options"
	"github.com/plgd-dev/go-coap/v3/tcp"
	"github.com/plgd-dev/go-coap/v3/tcp/client"
	"github.com/plgd-dev/hub/v2/coap-gateway/service"
	"github.com/plgd-dev/hub/v2/coap-gateway/uri"
	pkgX509 "github.com/plgd-dev/hub/v2/pkg/security/x509"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
)

const (
	deviceResourceType = "oic.d.virtualDevice"
	lightResourceType  = "oic.r.light"
	errSetResponseFmt  = "cannot set response: %v"
	ocfSchemePrefix    = "ocf://"
)

func generateIdentityCert(deviceID string, signerCert []*x509.Certificate, signerKey *ecdsa.PrivateKey) (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	certData, err := generateCertificate.GenerateIdentityCert(generateCertificate.Configuration{
		ValidFrom: time.Now().Add(-time.Hour).Format(time.RFC3339),
		ValidFor:  24 * time.Hour,
	}, deviceID, priv, signerCert, signerKey)
	if err != nil {
		return tls.Certificate{}, err
	}
	b, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	key := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
	crt, err := tls.X509KeyPair(certData, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	return crt, nil
}

func makeVerifyCertificate(signerCert []*x509.Certificate) func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		if len(rawCerts) == 0 {
			return errors.New("empty certificates chain")
		}
		intermediateCAPool := x509.NewCertPool()
		certs := make([]*x509.Certificate, 0, len(rawCerts))
		for _, rawCert := range rawCerts {
			cert, err := x509.ParseCertificate(rawCert)
			if err != nil {
				return err
			}
			certs = append(certs, cert)
		}
		for _, cert := range certs[1:] {
			intermediateCAPool.AddCert(cert)
		}
		caPool := x509.NewCertPool()
		for _, c := range signerCert {
			caPool.AddCert(c)
		}
		_, err := certs[0].Verify(x509.VerifyOptions{
			Roots:         caPool,
			Intermediates: intermediateCAPool,
			CurrentTime:   time.Now(),
			KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func connectDevice(ctx context.Context, signerCert []*x509.Certificate, signerKey *ecdsa.PrivateKey, deviceID string, numResources, resourceDataSize int, f func(w *responsewriter.ResponseWriter[*client.Conn], r *pool.Message)) (*client.Conn, error) {
	var tlsConfig *tls.Config

	crt, err := generateIdentityCert(deviceID, signerCert, signerKey)
	if err != nil {
		return nil, err
	}

	caPool := x509.NewCertPool()
	for _, c := range signerCert {
		caPool.AddCert(c)
	}
	tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{
			crt,
		},
		InsecureSkipVerify:    true,
		VerifyPeerCertificate: makeVerifyCertificate(signerCert),
	}

	conn, err := tcp.Dial(config.COAP_GW_HOST, options.WithTLS(tlsConfig), options.WithHandlerFunc(f), options.WithContext(ctx), options.WithMaxMessageSize(uint32(numResources*(1024+resourceDataSize))+8*1024), options.WithBlockwise(false, blockwise.SZX1024, time.Second*4))
	if err != nil {
		return nil, err
	}
	return conn, err
}

type testingT struct {
	err error
}

func (t *testingT) Errorf(format string, args ...interface{}) {
	t.err = fmt.Errorf(format, args...)
}

func (t *testingT) FailNow() {
	// do nothing - we don't want to stop test
}

func signUpDevice(ctx context.Context, deviceID string, co *client.Conn) (service.CoapSignUpResponse, error) {
	t := testingT{}
	code := oauthTest.GetDefaultDeviceAuthorizationCode(&t, deviceID)
	if t.err != nil {
		return service.CoapSignUpResponse{}, t.err
	}
	signUpReq := service.CoapSignUpRequest{
		DeviceID:              deviceID,
		AuthorizationCode:     code,
		AuthorizationProvider: config.DEVICE_PROVIDER,
	}
	inputCbor, err := cbor.Encode(signUpReq)
	if err != nil {
		return service.CoapSignUpResponse{}, err
	}
	req := co.AcquireMessage(ctx)
	defer co.ReleaseMessage(req)
	token, err := message.GetToken()
	if err != nil {
		return service.CoapSignUpResponse{}, err
	}
	req.SetCode(codes.POST)
	req.SetToken(token)
	err = req.SetPath(uri.SignUp)
	if err != nil {
		return service.CoapSignUpResponse{}, err
	}
	req.SetContentFormat(message.AppOcfCbor)
	req.SetBody(bytes.NewReader(inputCbor))

	resp, err := co.Do(req)
	if err != nil {
		return service.CoapSignUpResponse{}, err
	}
	defer co.ReleaseMessage(resp)

	if codes.Changed != resp.Code() {
		return service.CoapSignUpResponse{}, fmt.Errorf("cannot sign up device: %v", resp.Code())
	}
	var signUpResp service.CoapSignUpResponse
	err = cbor.ReadFrom(resp.Body(), &signUpResp)
	if err != nil {
		return service.CoapSignUpResponse{}, err
	}
	if signUpResp.AccessToken == "" {
		return service.CoapSignUpResponse{}, errors.New("cannot sign up device: empty access token")
	}
	return signUpResp, nil
}

func signInDevice(ctx context.Context, deviceID string, co *client.Conn, r service.CoapSignUpResponse) error {
	signInReq := service.CoapSignInReq{
		DeviceID:    deviceID,
		UserID:      r.UserID,
		AccessToken: r.AccessToken,
		Login:       true,
	}
	inputCbor, err := cbor.Encode(signInReq)
	if err != nil {
		return err
	}

	req := co.AcquireMessage(ctx)
	defer co.ReleaseMessage(req)
	token, err := message.GetToken()
	if err != nil {
		return err
	}
	req.SetCode(codes.POST)
	req.SetToken(token)
	err = req.SetPath(uri.SignIn)
	if err != nil {
		return err
	}
	req.SetContentFormat(message.AppOcfCbor)
	req.SetBody(bytes.NewReader(inputCbor))

	resp, err := co.Do(req)
	if err != nil {
		return err
	}
	if codes.Changed != resp.Code() {
		return fmt.Errorf("cannot sign in device: %v", resp.Code())
	}
	return nil
}

type wkRd struct {
	DeviceID   string               `json:"di"`
	Links      schema.ResourceLinks `json:"links"`
	TimeToLive int                  `json:"ttl"`
}

func publishResources(ctx context.Context, deviceID string, co *client.Conn, numResources int) error {
	links := make(schema.ResourceLinks, 0, 1)
	links = append(links, schema.ResourceLink{
		Href: platform.ResourceURI,
		Interfaces: []string{
			interfaces.OC_IF_R,
			interfaces.OC_IF_BASELINE,
		},
		ResourceTypes: []string{
			platform.ResourceType,
		},
		Policy: &schema.Policy{
			BitMask: schema.Discoverable | schema.Observable,
		},
		InstanceID: 318534269,
	}, schema.ResourceLink{
		Href: device.ResourceURI,
		Interfaces: []string{
			interfaces.OC_IF_R,
			interfaces.OC_IF_BASELINE,
		},
		ResourceTypes: []string{
			deviceResourceType,
			device.ResourceType,
		},
		Policy: &schema.Policy{
			BitMask: schema.Discoverable | schema.Observable,
		},
		InstanceID: 136750592,
	})
	for i := range numResources {
		links = append(links, schema.ResourceLink{
			Href: test.TestResourceLightInstanceHref(strconv.Itoa(i + 1)),
			Interfaces: []string{
				interfaces.OC_IF_RW,
				interfaces.OC_IF_BASELINE,
			},
			ResourceTypes: []string{
				lightResourceType,
			},
			Policy: &schema.Policy{
				BitMask: schema.Discoverable | schema.Observable,
			},
			InstanceID: int64(i),
		})
	}

	wk := wkRd{
		DeviceID:   deviceID,
		Links:      links,
		TimeToLive: 0,
	}
	inputCbor, err := cbor.Encode(wk)
	if err != nil {
		return err
	}

	req := co.AcquireMessage(ctx)
	defer co.ReleaseMessage(req)
	token, err := message.GetToken()
	if err != nil {
		return err
	}
	req.SetCode(codes.POST)
	req.SetToken(token)
	err = req.SetPath(uri.ResourceDirectory)
	if err != nil {
		return err
	}
	req.SetContentFormat(message.AppOcfCbor)
	req.SetBody(bytes.NewReader(inputCbor))
	resp, err := co.Do(req)
	if err != nil {
		return err
	}
	if codes.Changed != resp.Code() {
		return fmt.Errorf("cannot publish resources: %v", resp.Code())
	}
	return nil
}

// encodePlatformResource returns encoded platform resource
func encodePlatformResource(w *responsewriter.ResponseWriter[*client.Conn]) []byte {
	p := platform.Platform{
		ManufacturerName:   "OCF",
		SerialNumber:       "123456",
		PlatformVersion:    "1.0",
		PlatformIdentifier: "ocf",
		Interfaces: []string{
			interfaces.OC_IF_R,
			interfaces.OC_IF_BASELINE,
		},
		ResourceTypes: []string{
			platform.ResourceType,
		},
		ManufacturersURL:     "https://openconnectivity.org",
		ManufacturersSupport: "https://openconnectivity.org/support",
		ModelNumber:          "123456",
	}
	pCbor, err := cbor.Encode(p)
	if err != nil {
		fmt.Printf("cannot encode platform: %v", err)
		err = w.SetResponse(codes.Content, message.AppOcfCbor, bytes.NewReader([]byte{}))
		if err != nil {
			fmt.Printf(errSetResponseFmt, err)
		}
		os.Exit(1)
	}
	return pCbor
}

func encodeDeviceResource(deviceID string, w *responsewriter.ResponseWriter[*client.Conn]) []byte {
	d := device.Device{
		Name: deviceID,
		ID:   deviceID,
		Interfaces: []string{
			interfaces.OC_IF_R,
			interfaces.OC_IF_BASELINE,
		},
		ResourceTypes: []string{
			deviceResourceType,
			device.ResourceType,
		},
		ProtocolIndependentID: deviceID,
	}
	dCbor, err := cbor.Encode(d)
	if err != nil {
		fmt.Printf("cannot encode platform: %v", err)
		err := w.SetResponse(codes.Content, message.AppOcfCbor, bytes.NewReader([]byte{}))
		if err != nil {
			fmt.Printf(errSetResponseFmt, err)
		}
		os.Exit(1)
	}
	return dCbor
}

func encodeLightResource(i int, dataString []byte, w *responsewriter.ResponseWriter[*client.Conn]) []byte {
	l := map[string]interface{}{
		"n": "Light " + strconv.Itoa(i+1),
		"if": []string{
			interfaces.OC_IF_R,
			interfaces.OC_IF_BASELINE,
		},
		"rt": []string{
			lightResourceType,
		},
		"status":         false,
		"dimmingSetting": 0,
		"data":           string(dataString),
	}
	lCbor, err := cbor.Encode(l)
	if err != nil {
		fmt.Printf("cannot encode light: %v", err)
		err := w.SetResponse(codes.Content, message.AppOcfCbor, bytes.NewReader([]byte{}))
		if err != nil {
			fmt.Printf(errSetResponseFmt, err)
		}
		os.Exit(1)
	}
	return lCbor
}

func processBatchResourceLinks(w *responsewriter.ResponseWriter[*client.Conn], deviceID string, numResources, resourceDataSize int) {
	var data resources.BatchResourceDiscovery

	data = append(data, resources.BatchRepresentation{
		HrefRaw: ocfSchemePrefix + deviceID + platform.ResourceURI,
		Content: encodePlatformResource(w),
	}, resources.BatchRepresentation{
		HrefRaw: ocfSchemePrefix + deviceID + device.ResourceURI,
		Content: encodeDeviceResource(deviceID, w),
	})

	dataString := make([]byte, resourceDataSize)
	for i := 0; i < len(dataString); i++ {
		dataString[i] = (byte(i) % 32) + 'a'
	}
	for i := range numResources {
		data = append(data, resources.BatchRepresentation{
			HrefRaw: ocfSchemePrefix + deviceID + test.TestResourceLightInstanceHref(strconv.Itoa(i+1)),
			Content: encodeLightResource(i, dataString, w),
		})
	}
	inputCbor, err := cbor.Encode(data)
	if err != nil {
		fmt.Printf("cannot encode resource data: %v", err)
		err = w.SetResponse(codes.Content, message.AppOcfCbor, bytes.NewReader([]byte{}))
		if err != nil {
			fmt.Printf(errSetResponseFmt, err)
		}
		os.Exit(1)
		return
	}
	err = w.SetResponse(codes.Content, message.AppOcfCbor, bytes.NewReader(inputCbor))
	if err != nil {
		fmt.Printf(errSetResponseFmt, err)
		os.Exit(1)
	}
	w.Message().SetObserve(1000)
}

func processGetResourceLinks(w *responsewriter.ResponseWriter[*client.Conn], deviceID string, numResources int) {
	var links schema.ResourceLinks
	links = append(links, schema.ResourceLink{
		Anchor:   ocfSchemePrefix + deviceID + platform.ResourceURI,
		DeviceID: deviceID,
		Href:     platform.ResourceURI,
		Interfaces: []string{
			interfaces.OC_IF_R,
			interfaces.OC_IF_BASELINE,
		},
		ResourceTypes: []string{
			platform.ResourceType,
		},
		Policy: &schema.Policy{
			BitMask: schema.Discoverable | schema.Observable,
		},
		InstanceID: 318534269,
	}, schema.ResourceLink{
		Anchor:   ocfSchemePrefix + deviceID + device.ResourceURI,
		DeviceID: deviceID,
		Href:     device.ResourceURI,
		Interfaces: []string{
			interfaces.OC_IF_R,
			interfaces.OC_IF_BASELINE,
		},
		ResourceTypes: []string{
			deviceResourceType,
			device.ResourceType,
		},
		Policy: &schema.Policy{
			BitMask: schema.Discoverable | schema.Observable,
		},
		InstanceID: 136750592,
	}, schema.ResourceLink{
		Anchor:   ocfSchemePrefix + deviceID + resources.ResourceURI,
		DeviceID: deviceID,
		Href:     resources.ResourceURI,
		Interfaces: []string{
			interfaces.OC_IF_R,
			interfaces.OC_IF_BASELINE,
			interfaces.OC_IF_LL,
			interfaces.OC_IF_B,
		},
		ResourceTypes: []string{
			resources.ResourceType,
		},
		Policy: &schema.Policy{
			BitMask: schema.Discoverable | schema.Observable,
		},
		InstanceID: 136750593,
	},
	)
	for i := range numResources {
		links = append(links, schema.ResourceLink{
			Anchor:   ocfSchemePrefix + deviceID + test.TestResourceLightInstanceHref(strconv.Itoa(i+1)),
			DeviceID: deviceID,
			Href:     test.TestResourceLightInstanceHref(strconv.Itoa(i + 1)),
			Interfaces: []string{
				interfaces.OC_IF_RW,
				interfaces.OC_IF_BASELINE,
			},
			ResourceTypes: []string{
				lightResourceType,
			},
			Policy: &schema.Policy{
				BitMask: schema.Discoverable | schema.Observable,
			},
			InstanceID: int64(i),
		})
	}
	inputCbor, err := cbor.Encode(links)
	if err != nil {
		fmt.Printf("cannot encode resource links: %v", err)
		err = w.SetResponse(codes.Content, message.AppOcfCbor, bytes.NewReader([]byte{}))
		if err != nil {
			fmt.Printf(errSetResponseFmt, err)
		}
		return
	}
	err = w.SetResponse(codes.Content, message.AppOcfCbor, bytes.NewReader(inputCbor))
	if err != nil {
		fmt.Printf(errSetResponseFmt, err)
	}
}

func processGetDiscovery(w *responsewriter.ResponseWriter[*client.Conn], query string, deviceID string, numResources, resourceDataSize int) {
	if query == uri.InterfaceQueryKeyPrefix+interfaces.OC_IF_B {
		processBatchResourceLinks(w, deviceID, numResources, resourceDataSize)
		return
	}
	if query == uri.InterfaceQueryKeyPrefix+interfaces.OC_IF_LL {
		processGetResourceLinks(w, deviceID, numResources)
		return
	}
	err := w.SetResponse(codes.Content, message.AppOcfCbor, bytes.NewReader([]byte{}))
	if err != nil {
		fmt.Printf(errSetResponseFmt, err)
	}
}

func handleGET(deviceID string, numResources, resourceDataSize int, resp []byte, w *responsewriter.ResponseWriter[*client.Conn], r *pool.Message) error {
	var path string
	path, err := r.Path()
	if err != nil {
		fmt.Printf("cannot get path: %v", err)
		err = w.SetResponse(codes.Content, message.TextPlain, bytes.NewReader(resp))
		if err != nil {
			fmt.Printf(errSetResponseFmt, err)
		}
		return nil
	}
	if path == uri.ResourceDiscovery {
		q, err := r.Options().Queries()
		if err == nil {
			processGetDiscovery(w, q[0], deviceID, numResources, resourceDataSize)
			return nil
		}
	}
	return w.SetResponse(codes.Content, message.TextPlain, bytes.NewReader(resp))
}

func makeDeviceHandler(deviceID string, numResources, resourceDataSize int) func(w *responsewriter.ResponseWriter[*client.Conn], r *pool.Message) {
	return func(w *responsewriter.ResponseWriter[*client.Conn], r *pool.Message) {
		var err error
		resp := []byte("hello world")
		switch r.Code() {
		case codes.POST:
			err = w.SetResponse(codes.Changed, message.TextPlain, bytes.NewReader(resp))
		case codes.GET:
			err = handleGET(deviceID, numResources, resourceDataSize, resp, w, r)
		case codes.PUT:
			err = w.SetResponse(codes.Created, message.TextPlain, bytes.NewReader(resp))
		case codes.DELETE:
			err = w.SetResponse(codes.Deleted, message.TextPlain, bytes.NewReader(resp))
		}
		if err != nil {
			fmt.Printf(errSetResponseFmt, err)
		}
	}
}

func runDevice(ctx context.Context, signerCert []*x509.Certificate, signerKey *ecdsa.PrivateKey, deviceID string, numResources, resourceDataSize int) (*client.Conn, error) {
	co, err := connectDevice(ctx, signerCert, signerKey, deviceID, numResources, resourceDataSize, makeDeviceHandler(deviceID, numResources, resourceDataSize))
	if err != nil {
		return nil, err
	}
	r, err := signUpDevice(ctx, deviceID, co)
	if err != nil {
		co.Close()
		return nil, err
	}
	err = signInDevice(ctx, deviceID, co, r)
	if err != nil {
		co.Close()
		return nil, err
	}
	err = publishResources(ctx, deviceID, co, numResources)
	if err != nil {
		co.Close()
		return nil, err
	}
	return co, nil
}

func main() {
	numDevices := flag.Int("numDevices", 1, "number of devices")
	numResource := flag.Int("numResources", 1, "number of resources per device")
	resourceDataSize := flag.Int("resourceDataSize", 200, "size of resource data property in bytes")
	flag.Parse()
	signerCert, err := pkgX509.ReadX509(os.Getenv("TEST_ROOT_CA_CERT"))
	if err != nil {
		fmt.Printf("cannot load signer cert: %v", err)
		os.Exit(1)
	}
	signerKey, err := pkgX509.ReadPrivateKey(os.Getenv("TEST_ROOT_CA_KEY"))
	if err != nil {
		fmt.Printf("cannot load signer key: %v", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for i := range *numDevices {
		for {
			c, err := runDevice(ctx, signerCert, signerKey, test.GenerateDeviceIDbyIdx(i), *numResource, *resourceDataSize)
			if err == nil {
				defer c.Close()
				break
			}
			fmt.Printf("cannot run device: %v\n", err)
			time.Sleep(1 * time.Second)
			continue
		}
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	<-sigs
}

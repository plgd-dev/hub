package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/schema/device"
	capb "github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	grpcCloud "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/oauth-server/uri"
	"github.com/plgd-dev/kit/v2/codec/json"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func getServiceToken(ctx context.Context, authAddr string) (string, error) {
	reqBody := map[string]string{
		"grant_type":    string(service.AllowedGrantType_CLIENT_CREDENTIALS),
		uri.ClientIDKey: oauthTest.ClientTest,
		uri.AudienceKey: "test",
	}
	d, err := json.Encode(reqBody)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+authAddr+"/oauth/token", bytes.NewReader(d))
	if err != nil {
		return "", err
	}
	c := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	res, err := c.Do(request)
	if err != nil {
		return "", err
	}
	defer func() {
		if errC := res.Body.Close(); errC != nil {
			log.Fatalf("failed to close response body: %v", errC)
		}
	}()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("returns status code %v", res.StatusCode)
	}
	var body map[string]string
	err = json.ReadFrom(res.Body, &body)
	if err != nil {
		return "", err
	}
	token := body["access_token"]
	if token == "" {
		return "", errors.New("token not found in body")
	}
	return token, nil
}

func ownAndOnboard(ctx context.Context, c *OcfClient, deviceID, apn, authCode string) {
	fmt.Printf("own device %v\n", deviceID)
	newID, err := c.OwnDevice(ctx, deviceID)
	if err != nil {
		log.Fatalf("cannot own device %v: %v", deviceID, err)
	}
	err = c.OnboardDevice(ctx, newID, apn, authCode)
	if err != nil {
		err = c.DisownDevice(ctx, newID)
		if err != nil {
			log.Fatalf("cannot disown device %v: %v", newID, err)
		}
		log.Fatalf("cannot onboard device %v: %v", newID, err)
	}
}

func getAccessToken(ctx context.Context, authAddr string) string {
	accessToken, err := getServiceToken(ctx, authAddr)
	if err != nil {
		log.Fatalf("cannot get access token: %v", err)
	}
	return accessToken
}

// check if port is part of address, otherwise append ":443"
func getAddress(addr string) string {
	_, _, err := net.SplitHostPort(addr)
	if err != nil {
		if strings.Contains(err.Error(), "missing port in address") {
			return addr + ":443"
		}
	}
	return addr
}

func getHubConfiguration(ctx context.Context, grpcConn *grpc.ClientConn) *pb.HubConfigurationResponse {
	grpcClient := pb.NewGrpcGatewayClient(grpcConn)

	hubConfiguration, err := grpcClient.GetHubConfiguration(ctx, &pb.HubConfigurationRequest{})
	if err != nil {
		log.Fatalf("cannot get hub configuration: %v", err)
	}
	return hubConfiguration
}

func newOcfClient(ctx context.Context, grpcConn *grpc.ClientConn, hubConfiguration *pb.HubConfigurationResponse) *OcfClient {
	caClient := capb.NewCertificateAuthorityClient(grpcConn)
	c := new(OcfClient)
	err := c.Initialize(ctx, hubConfiguration, caClient)
	if err != nil {
		log.Fatalf("cannot initialize ocf client: %v", err)
	}
	return c
}

type deviceFilter = func(d client.DeviceDetails) bool

func getDevices(ctx context.Context, clt *OcfClient, timeout time.Duration, filter deviceFilter) []client.DeviceDetails {
	devices, err := clt.Discover(ctx, timeout)
	if err != nil {
		log.Fatalf("cannot device devices: %v", err)
	}
	filteredDevices := make([]client.DeviceDetails, 0, len(devices))
	for _, d := range devices {
		if !filter(d) {
			continue
		}
		filteredDevices = append(filteredDevices, d)
	}
	return filteredDevices
}

func getDeviceName(details interface{}) string {
	name := "unknown"
	if details != nil {
		if v, ok := details.(*device.Device); ok {
			name = v.Name
		}
	}
	return name
}

func main() {
	addr := flag.String("addr", "localhost:443", "address")
	authAddr := flag.String("authAddr", "", "auth address to get access token from mock oauth server")
	accessToken := flag.String("accessToken", "", "use directly access token without contacting mock oauth server")
	authCode := flag.String("authCode", "test", "use authorization code for registration device to the cloud")
	deviceID := flag.String("deviceId", "", "onboard the device")
	listDevices := flag.Bool("listDevices", false, "list devices which can be onboard to the cloud")
	discoverDuration := flag.Duration("discoverDuration", time.Second, "discover devices for X seconds")
	apn := flag.String("authorizationProvider", "plgd", "use authorization provider for registration device to the cloud")
	maxNum := flag.Int("maxNum", 1, "maximum number of devices which will be onboarded")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *discoverDuration+15*time.Second*(time.Duration(*maxNum+1)))
	defer cancel()
	if *authAddr == "" {
		*authAddr = *addr
	}
	if *accessToken == "" {
		*accessToken = getAccessToken(ctx, *authAddr)
	}

	*addr = getAddress(*addr)

	tlsCfg := tls.Config{
		InsecureSkipVerify: true,
	}
	grpcConn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(credentials.NewTLS(&tlsCfg)))
	if err != nil {
		panic(fmt.Errorf("cannot connect to grpc: %w", err))
	}
	defer func() {
		_ = grpcConn.Close()
	}()

	ctx = grpcCloud.CtxWithToken(ctx, *accessToken)

	hubConfiguration := getHubConfiguration(ctx, grpcConn)
	c := newOcfClient(ctx, grpcConn, hubConfiguration)

	if *deviceID != "" {
		ownAndOnboard(ctx, c, *deviceID, *apn, *authCode)
		return
	}

	filteredDevices := getDevices(ctx, c, *discoverDuration, func(d client.DeviceDetails) bool {
		return d.IsSecured && d.OwnershipStatus == client.OwnershipStatus_ReadyToBeOwned
	})
	fmt.Printf("found %v ready to be owned devices with discover duration %v\n", len(filteredDevices), *discoverDuration)
	for idx, d := range filteredDevices {
		if !*listDevices {
			ownAndOnboard(ctx, c, d.ID, *apn, *authCode)
			if idx == *maxNum-1 {
				return
			}
			continue
		}
		id := d.ID
		name := getDeviceName(d.Details)
		fmt.Printf("%v(%v)\n", name, id)
	}
}

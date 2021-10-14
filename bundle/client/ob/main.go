package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/plgd-dev/device/client"
	capb "github.com/plgd-dev/hub/certificate-authority/pb"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	grpcCloud "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	"github.com/plgd-dev/hub/test/oauth-server/uri"
	"github.com/plgd-dev/kit/v2/codec/json"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func getServiceToken(authAddr string) (string, error) {
	reqBody := map[string]string{
		"grant_type":    string(service.AllowedGrantType_CLIENT_CREDENTIALS),
		uri.ClientIDKey: oauthTest.ClientTest,
		uri.AudienceKey: "test",
	}
	d, err := json.Encode(reqBody)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequest(http.MethodPost, "https://"+authAddr+"/oauth/token", bytes.NewReader(d))
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
		if err := res.Body.Close(); err != nil {
			log.Fatalf("failed to close response body: %v", err)
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
		return "", fmt.Errorf("token not found in body")
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

func main() {
	addr := flag.String("addr", "localhost:443", "address")
	accessToken := flag.String("accessToken", "", "accessToken")
	authCode := flag.String("authCode", "test", "authCode")
	deviceID := flag.String("deviceId", "", "deviceId")
	discoverTimeout := flag.Duration("discoverTimeout", time.Second, "discoverTimeout")
	apn := flag.String("authorizationProvider", "plgd", "authorizationProvider")
	flag.Parse()

	if *accessToken == "" {
		var err error
		*accessToken, err = getServiceToken(*addr)
		if err != nil {
			log.Fatalf("cannot get access token")
		}
	}

	tlsCfg := tls.Config{
		InsecureSkipVerify: true,
	}

	grpcConn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(credentials.NewTLS(&tlsCfg)))
	if err != nil {
		log.Fatalf("cannot connect to grpc: %v", err)
	}
	defer func() {
		_ = grpcConn.Close()
	}()
	grpcClient := pb.NewGrpcGatewayClient(grpcConn)

	caClient := capb.NewCertificateAuthorityClient(grpcConn)
	ctx, cancel := context.WithTimeout(context.Background(), *discoverTimeout+60*time.Second)
	defer cancel()
	ctx = grpcCloud.CtxWithToken(ctx, *accessToken)

	c := new(OcfClient)
	err = c.Initialize(ctx, grpcClient, caClient)
	if err != nil {
		log.Fatalf("cannot initialize ocf client: %v", err)
	}

	if *deviceID != "" {
		ownAndOnboard(ctx, c, *deviceID, *apn, *authCode)
		return
	}

	devices, err := c.Discover(ctx, *discoverTimeout)
	if err != nil {
		log.Fatalf("cannot device devices: %v", err)
	}
	fmt.Printf("found %v devices with discover timeout %v\n", len(devices), *discoverTimeout)

	for _, d := range devices {
		if d.IsSecured && d.OwnershipStatus == client.OwnershipStatus_ReadyToBeOwned {
			ownAndOnboard(ctx, c, d.ID, *apn, *authCode)
			return
		}
	}

}

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/plgd-dev/go-coap/v3/message"
	pbGW "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/oauth-server/uri"
	"github.com/plgd-dev/kit/v2/codec/json"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func getServiceToken(authAddr string, tls *tls.Config) (string, error) {
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
			TLSClientConfig: tls,
		},
	}
	res, err := c.Do(request)
	if err != nil {
		return "", err
	}
	defer func() {
		if errC := res.Body.Close(); errC != nil {
			log.Errorf("failed to close response body: %w", errC)
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

func getAccessToken(authAddr string, tls *tls.Config) string {
	accesstoken, err := getServiceToken(authAddr, tls)
	if err != nil {
		log.Fatalf("cannot get access token: %v", err)
	}
	return accesstoken
}

func jsonEncodeError(err error) error {
	return fmt.Errorf("cannot encode resp to json: %w", err)
}

func deleteResource(ctx context.Context, client pbGW.GrpcGatewayClient, deviceID, href string) {
	delError := func(err error) {
		log.Fatalf("cannot delete resource: %v", err)
	}
	resp, err := client.DeleteResource(ctx, &pbGW.DeleteResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, href),
	})
	if err != nil {
		delError(err)
	}
	d, err := json.Encode(resp)
	if err != nil {
		delError(jsonEncodeError(err))
	}
	fmt.Println(string(d))
}

func updateResource(ctx context.Context, client pbGW.GrpcGatewayClient, deviceID, href string, contentFormat int) {
	updError := func(err error) {
		log.Fatalf("cannot update resource: %v", err)
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		updError(fmt.Errorf("cannot read data: %w", err))
	}
	resp, err := client.UpdateResource(ctx, &pbGW.UpdateResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, href),
		Content: &pbGW.Content{
			ContentType: message.MediaType(contentFormat).String(), //nolint:gosec
			Data:        data,
		},
	})
	if err != nil {
		updError(err)
	}
	d, err := json.Encode(resp)
	if err != nil {
		updError(jsonEncodeError(err))
	}
	fmt.Println(string(d))
}

func createResource(ctx context.Context, client pbGW.GrpcGatewayClient, deviceID, href string, contentFormat int) {
	createError := func(err error) {
		log.Fatalf("cannot create resource: %v", err)
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		createError(fmt.Errorf("cannot read data: %w", err))
	}
	resp, err := client.CreateResource(ctx, &pbGW.CreateResourceRequest{
		ResourceId: commands.NewResourceID(deviceID, href),
		Content: &pbGW.Content{
			ContentType: message.MediaType(contentFormat).String(), //nolint:gosec
			Data:        data,
		},
	})
	if err != nil {
		createError(err)
	}
	d, err := json.Encode(resp)
	if err != nil {
		createError(jsonEncodeError(err))
	}
	fmt.Println(string(d))
}

func getDevices(ctx context.Context, client pbGW.GrpcGatewayClient) {
	getError := func(err error) {
		log.Fatalf("cannot get devices: %v", err)
	}
	getClient, err := client.GetDevices(ctx, &pbGW.GetDevicesRequest{})
	if err != nil {
		getError(err)
	}
	devices := make([]*pbGW.Device, 0, 4)
	for {
		resp, err2 := getClient.Recv()
		if errors.Is(err2, io.EOF) {
			break
		}
		if err2 != nil {
			getError(fmt.Errorf("cannot recv device: %w", err2))
		}
		devices = append(devices, resp)
	}
	d, err := json.Encode(devices)
	if err != nil {
		getError(jsonEncodeError(err))
	}
	fmt.Println(string(d))
}

func getResource(ctx context.Context, client pbGW.GrpcGatewayClient, deviceID, href string) {
	getError := func(err error) {
		log.Fatalf("cannot get resource: %v", err)
	}
	var deviceIdFilter []string
	if deviceID != "" {
		deviceIdFilter = append(deviceIdFilter, deviceID)
	}
	var resourceIdFilter []*pbGW.ResourceIdFilter
	if href != "" {
		resourceIdFilter = append(resourceIdFilter, &pbGW.ResourceIdFilter{
			ResourceId: commands.NewResourceID(deviceID, href),
		})
	}
	getClient, err := client.GetResources(ctx, &pbGW.GetResourcesRequest{
		ResourceIdFilter: resourceIdFilter,
		DeviceIdFilter:   deviceIdFilter,
	})
	if err != nil {
		getError(fmt.Errorf("cannot retrieve values: %w", err))
	}
	resources := make([]*pbGW.Resource, 0, 4)
	for {
		resp, err2 := getClient.Recv()
		if errors.Is(err2, io.EOF) {
			break
		}
		if err2 != nil {
			getError(fmt.Errorf("cannot recv value: %w", err2))
		}
		resources = append(resources, resp)
	}
	d, err := json.Encode(resources)
	if err != nil {
		getError(jsonEncodeError(err))
	}
	fmt.Println(string(d))
}

func main() {
	addr := flag.String("addr", "localhost:443", "address")
	accesstoken := flag.String("accesstoken", "", "accesstoken")
	authAddr := flag.String("authaddr", "localhost:443", "authorization service address")
	deviceID := flag.String("deviceid", "", "deviceID")
	href := flag.String("href", "", "href")
	getOpt := flag.Bool("get", true, "get resources(default) filtered by deviceid and href")
	getDevicesOpt := flag.Bool("getdevices", false, "get devices")
	updateOpt := flag.Bool("update", false, "update resource, content is expected in stdin")
	deleteOpt := flag.Bool("delete", false, "delete resource")
	createOpt := flag.Bool("create", false, "create resource, content is expected in stdin")

	contentFormat := flag.Int("contentFormat", int(message.AppJSON), "contentFormat for update resource")

	flag.Parse()

	tlsCfg := tls.Config{
		InsecureSkipVerify: true,
	}
	if *accesstoken == "" {
		*accesstoken = getAccessToken(*authAddr, &tlsCfg)
	}

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(credentials.NewTLS(&tlsCfg)))
	if err != nil {
		log.Fatalf("Error dialing: %v", err)
	}

	ocfGW := pbGW.NewGrpcGatewayClient(conn)
	ctx := kitNetGrpc.CtxWithToken(context.Background(), *accesstoken)
	switch {
	case *deleteOpt:
		deleteResource(ctx, ocfGW, *deviceID, *href)
	case *updateOpt:
		updateResource(ctx, ocfGW, *deviceID, *href, *contentFormat)
	case *createOpt:
		createResource(ctx, ocfGW, *deviceID, *href, *contentFormat)
	case *getDevicesOpt:
		getDevices(ctx, ocfGW)
	case *getOpt:
		getResource(ctx, ocfGW, *deviceID, *href)
	default:
		log.Fatal("unknown command")
	}
}

package client_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"testing"

	"github.com/plgd-dev/hub/grpc-gateway/client"
	"github.com/plgd-dev/hub/pkg/net/grpc/server"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/plgd-dev/hub/test"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/plgd-dev/hub/grpc-gateway/pb"
	testCfg "github.com/plgd-dev/hub/test/config"
)

const (
	TestHref         = "test href"
	TestManufacturer = "Test Manufacturer"
)

var ClientTestCfg = client.Config{
	GatewayAddress: testCfg.GRPC_HOST,
}

func NewTestClient(t *testing.T) *client.Client {
	rootCAs := x509.NewCertPool()
	for _, c := range test.GetRootCertificateAuthorities(t) {
		rootCAs.AddCert(c)
	}
	tlsCfg := tls.Config{
		RootCAs: rootCAs,
	}
	c, err := client.NewFromConfig(&ClientTestCfg, &tlsCfg)
	require.NoError(t, err)
	return c
}

func NewGateway(addr string) (*server.Server, error) {
	s, err := server.NewServer(addr)
	if err != nil {
		return nil, err
	}
	h := gatewayHandler{}

	pb.RegisterGrpcGatewayServer(s.Server, &h)

	return s, nil
}

type gatewayHandler struct {
	pb.UnimplementedGrpcGatewayServer
	deviceID   string
	deviceName string
}

func (h *gatewayHandler) GetCloudConfiguration(context.Context, *pb.CloudConfigurationRequest) (*pb.CloudConfigurationResponse, error) {
	return &pb.CloudConfigurationResponse{
		CloudId: "abc",
	}, nil
}

func (h *gatewayHandler) GetDevices(req *pb.GetDevicesRequest, srv pb.GrpcGateway_GetDevicesServer) error {
	v := pb.Device{
		Id:   h.deviceID,
		Name: h.deviceName,
		Metadata: &pb.Device_Metadata{
			Status: &commands.ConnectionStatus{
				Value: commands.ConnectionStatus_ONLINE,
			},
		},
		ManufacturerName: []*pb.LocalizedString{{Value: TestManufacturer, Language: "en"}},
	}
	err := srv.Send(&v)
	if err != nil {
		return status.Errorf(status.Convert(err).Code(), "sending failed: %v", err)
	}
	return nil
}

func (h *gatewayHandler) GetResourceLinks(req *pb.GetResourceLinksRequest, srv pb.GrpcGateway_GetResourceLinksServer) error {
	err := srv.Send(&events.ResourceLinksPublished{
		DeviceId: h.deviceID,
		Resources: []*commands.Resource{
			{
				Href: "excluded", ResourceTypes: []string{schema.DeviceResourceType}, DeviceId: h.deviceID,
			},
			{
				Href: TestHref, ResourceTypes: []string{"x.com.test.type"}, DeviceId: h.deviceID,
			},
		},
	})
	if err != nil {
		return status.Errorf(status.Convert(err).Code(), "sending failed: %v", err)
	}
	return nil
}

func (h *gatewayHandler) GetResources(req *pb.GetResourcesRequest, srv pb.GrpcGateway_GetResourcesServer) error {
	err := sendResourceValue(srv, h.deviceID, schema.DeviceResourceType, schema.Device{
		ID:   h.deviceID,
		Name: h.deviceName,
	})
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	return nil
}

func (h *gatewayHandler) UpdateResourcesValues(context.Context, *pb.UpdateResourceRequest) (*pb.UpdateResourceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "not implemented")
}

func (h *gatewayHandler) SubscribeToEvents(pb.GrpcGateway_SubscribeToEventsServer) error {
	return status.Errorf(codes.Unimplemented, "not implemented")
}

func (h *gatewayHandler) GetResourceFromDevice(context.Context, *pb.GetResourceFromDeviceRequest) (*pb.GetResourceFromDeviceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "not implemented")
}

func (h *gatewayHandler) DeleteResource(context.Context, *pb.DeleteResourceRequest) (*pb.DeleteResourceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "not implemented")
}

func sendResourceValue(srv pb.GrpcGateway_GetResourcesServer, deviceId, resourceType string, v interface{}) error {
	c, err := cbor.Encode(v)
	if err != nil {
		return status.Errorf(codes.Internal, "%v", err)
	}
	rv := pb.Resource{
		Data: &events.ResourceChanged{
			ResourceId: &commands.ResourceId{DeviceId: deviceId},
			Content:    &commands.Content{ContentType: message.AppCBOR.String(), Data: c},
		},
		Types: []string{resourceType},
	}
	err = srv.Send(&rv)
	if err != nil {
		return status.Errorf(status.Convert(err).Code(), "sending failed: %v", err)
	}
	return nil
}

package client_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"testing"

	"github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/pkg/net/grpc/server"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/events"
	"github.com/plgd-dev/cloud/test"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/codec/cbor"
	"github.com/plgd-dev/sdk/schema"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	testCfg "github.com/plgd-dev/cloud/test/config"
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

func (h *gatewayHandler) GetClientConfiguration(context.Context, *pb.ClientConfigurationRequest) (*pb.ClientConfigurationResponse, error) {
	return &pb.ClientConfigurationResponse{
		CloudId: "abc",
	}, nil
}

func (h *gatewayHandler) GetDevices(req *pb.GetDevicesRequest, srv pb.GrpcGateway_GetDevicesServer) error {
	v := pb.Device{
		Id:               h.deviceID,
		Name:             h.deviceName,
		IsOnline:         true,
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

func (h *gatewayHandler) RetrieveResources(req *pb.RetrieveResourcesRequest, srv pb.GrpcGateway_RetrieveResourcesServer) error {
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

func (h *gatewayHandler) UpdateResourcesValues(context.Context, *pb.UpdateResourceRequest) (*events.ResourceUpdated, error) {
	return nil, status.Errorf(codes.Unimplemented, "not implemented")
}

func (h *gatewayHandler) SubscribeToEvents(pb.GrpcGateway_SubscribeToEventsServer) error {
	return status.Errorf(codes.Unimplemented, "not implemented")
}

func (h *gatewayHandler) RetrieveResourceFromDevice(context.Context, *pb.RetrieveResourceFromDeviceRequest) (*events.ResourceRetrieved, error) {
	return nil, status.Errorf(codes.Unimplemented, "not implemented")
}

func (h *gatewayHandler) DeleteResource(context.Context, *pb.DeleteResourceRequest) (*events.ResourceDeleted, error) {
	return nil, status.Errorf(codes.Unimplemented, "not implemented")
}

func sendResourceValue(srv pb.GrpcGateway_RetrieveResourcesServer, deviceId, resourceType string, v interface{}) error {
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

package client_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"testing"

	"github.com/go-ocf/cloud/grpc-gateway/client"
	"github.com/go-ocf/cloud/test"
	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/kit/codec/cbor"
	kit "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/sdk/schema"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-ocf/cloud/grpc-gateway/pb"
	testCfg "github.com/go-ocf/cloud/test/config"
)

const (
	TestHref         = "test href"
	TestManufacturer = "Test Manufacturer"
)

var ClientTestCfg = client.Config{
	GatewayAddress: testCfg.GRPC_HOST,
	AccessTokenURL: testCfg.AUTH_HOST,
}

func NewTestClient(t *testing.T) *client.Client {
	rootCAs := x509.NewCertPool()
	for _, c := range test.GetRootCertificateAuthorities(t) {
		rootCAs.AddCert(c)
	}
	tlsCfg := tls.Config{
		RootCAs: rootCAs,
	}
	c, err := client.NewClientFromConfig(&ClientTestCfg, &tlsCfg)
	require.NoError(t, err)
	return c
}

func NewGateway(addr string) (*kit.Server, error) {
	s, err := kit.NewServer(addr)
	if err != nil {
		return nil, err
	}
	h := gatewayHandler{}

	pb.RegisterGrpcGatewayServer(s.Server, &h)

	return s, nil
}

type gatewayHandler struct {
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
		ManufacturerName: []*pb.LocalizedString{&pb.LocalizedString{Value: TestManufacturer, Language: "en"}},
	}
	err := srv.Send(&v)
	if err != nil {
		return status.Errorf(status.Convert(err).Code(), "sending failed: %v", err)
	}
	return nil
}

func (h *gatewayHandler) GetResourceLinks(req *pb.GetResourceLinksRequest, srv pb.GrpcGateway_GetResourceLinksServer) error {
	err := srv.Send(&pb.ResourceLink{Href: "excluded", Types: []string{schema.DeviceResourceType}, DeviceId: h.deviceID})
	if err != nil {
		return status.Errorf(status.Convert(err).Code(), "sending failed: %v", err)
	}
	err = srv.Send(&pb.ResourceLink{Href: TestHref, Types: []string{"x.com.test.type"}, DeviceId: h.deviceID})
	if err != nil {
		return status.Errorf(status.Convert(err).Code(), "sending failed: %v", err)
	}
	return nil
}

func (h *gatewayHandler) RetrieveResourcesValues(req *pb.RetrieveResourcesValuesRequest, srv pb.GrpcGateway_RetrieveResourcesValuesServer) error {
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

func (h *gatewayHandler) UpdateResourcesValues(context.Context, *pb.UpdateResourceValuesRequest) (*pb.UpdateResourceValuesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "not implemented")
}

func (h *gatewayHandler) SubscribeForEvents(pb.GrpcGateway_SubscribeForEventsServer) error {
	return status.Errorf(codes.Unimplemented, "not implemented")
}

func (h *gatewayHandler) RetrieveResourceFromDevice(context.Context, *pb.RetrieveResourceFromDeviceRequest) (*pb.RetrieveResourceFromDeviceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "not implemented")
}

func sendResourceValue(srv pb.GrpcGateway_RetrieveResourcesValuesServer, deviceId, resourceType string, v interface{}) error {
	c, err := cbor.Encode(v)
	if err != nil {
		return status.Errorf(codes.Internal, "%v", err)
	}
	rv := pb.ResourceValue{
		ResourceId: &pb.ResourceId{DeviceId: deviceId},
		Types:      []string{resourceType},
		Content:    &pb.Content{ContentType: message.AppCBOR.String(), Data: c},
	}
	err = srv.Send(&rv)
	if err != nil {
		return status.Errorf(status.Convert(err).Code(), "sending failed: %v", err)
	}
	return nil
}

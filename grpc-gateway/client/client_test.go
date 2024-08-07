package client_test

import (
	"context"

	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/kit/v2/codec/cbor"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	TestHref         = "test href"
	TestManufacturer = "Test Manufacturer"
)

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

func (h *gatewayHandler) GetHubConfiguration(context.Context, *pb.HubConfigurationRequest) (*pb.HubConfigurationResponse, error) {
	return &pb.HubConfigurationResponse{
		Id: "abc",
	}, nil
}

func (h *gatewayHandler) GetDevices(_ *pb.GetDevicesRequest, srv pb.GrpcGateway_GetDevicesServer) error {
	v := pb.Device{
		Id:   h.deviceID,
		Name: h.deviceName,
		Metadata: &pb.Device_Metadata{
			Connection: &commands.Connection{
				Status:   commands.Connection_ONLINE,
				Protocol: test.StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME),
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

func (h *gatewayHandler) GetResourceLinks(_ *pb.GetResourceLinksRequest, srv pb.GrpcGateway_GetResourceLinksServer) error {
	err := srv.Send(&events.ResourceLinksPublished{
		DeviceId: h.deviceID,
		Resources: []*commands.Resource{
			{
				Href: "excluded", ResourceTypes: []string{device.ResourceType}, DeviceId: h.deviceID,
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

func (h *gatewayHandler) GetResources(_ *pb.GetResourcesRequest, srv pb.GrpcGateway_GetResourcesServer) error {
	err := sendResourceValue(srv, h.deviceID, device.ResourceType, device.Device{
		ID:   h.deviceID,
		Name: h.deviceName,
	})
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

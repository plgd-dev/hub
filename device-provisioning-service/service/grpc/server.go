package grpc

import (
	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store"
)

type DeviceProvisionServiceServer struct {
	store      store.Store
	ownerClaim string

	pb.UnimplementedDeviceProvisionServiceServer
}

func NewDeviceProvisionServiceServer(store store.Store, ownerClaim string) *DeviceProvisionServiceServer {
	return &DeviceProvisionServiceServer{
		store:      store,
		ownerClaim: ownerClaim,
	}
}

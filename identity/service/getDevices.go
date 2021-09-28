package service

import (
	"github.com/plgd-dev/cloud/identity/pb"
	"github.com/plgd-dev/cloud/identity/persistence"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// hasMatchDeviceID returns true if device id match filter.
// An empty deviceIdFilter matches all device ids.
func hasMatchDeviceID(deviceId string, deviceIdFilter map[string]bool) bool {
	if len(deviceIdFilter) == 0 {
		return true
	}
	if _, ok := deviceIdFilter[deviceId]; ok {
		return true
	}
	return false
}

func sendDevices(request *pb.GetDevicesRequest, srv pb.IdentityService_GetDevicesServer, createIter func() persistence.Iterator) error {
	deviceIdFilter := make(map[string]bool)
	for _, deviceID := range request.GetDeviceIdsFilter() {
		deviceIdFilter[deviceID] = true
	}
	ids := make([]string, 0, 16)
	var d persistence.AuthorizedDevice
	it := createIter()
	for it.Next(&d) {
		if hasMatchDeviceID(d.DeviceID, deviceIdFilter) {
			ids = append(ids, d.DeviceID)
		}
	}
	it.Close()
	if it.Err() != nil {
		return log.LogAndReturnError(status.Errorf(codes.Internal, "cannot get user devices: %v", it.Err()))
	}

	for _, deviceID := range ids {
		err := srv.Send(&pb.Device{DeviceId: deviceID})
		if err != nil {
			return log.LogAndReturnError(status.Errorf(status.Convert(err).Code(), "cannot get user devices: %v", err))
		}
	}
	return nil
}

// GetDevices returns a list of user's devices if the access token is valid.
func (s *Service) GetDevices(request *pb.GetDevicesRequest, srv pb.IdentityService_GetDevicesServer) error {
	tx := s.persistence.NewTransaction(srv.Context())
	defer tx.Close()

	owner, err := grpc.OwnerFromTokenMD(srv.Context(), s.ownerClaim)
	if err != nil {
		return log.LogAndReturnError(grpc.ForwardErrorf(codes.InvalidArgument, "cannot get owner devices: %v", err))
	}

	return sendDevices(request, srv, func() persistence.Iterator {
		return tx.RetrieveByOwner(owner)
	})
}

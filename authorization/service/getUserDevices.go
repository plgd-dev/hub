package service

import (
	"github.com/plgd-dev/cloud/authorization/persistence"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// hasMatchDeviceID returns true if device id match filter.
// An empty deviceIdsFilter matches all device ids.
func hasMatchDeviceID(deviceId string, deviceIdsFilter map[string]bool) bool {
	if len(deviceIdsFilter) == 0 {
		return true
	}
	if _, ok := deviceIdsFilter[deviceId]; ok {
		return true
	}
	return false
}

func logAndReturnError(err error) error {
	log.Errorf("%v", err)
	return err
}

func sendUserDevices(request *pb.GetUserDevicesRequest, srv pb.AuthorizationService_GetUserDevicesServer, createIter func() persistence.Iterator) error {
	deviceIdsFilter := make(map[string]bool)
	for _, deviceID := range request.GetDeviceIdsFilter() {
		deviceIdsFilter[deviceID] = true
	}
	ids := make([]string, 0, 16)
	var d persistence.AuthorizedDevice
	it := createIter()
	for it.Next(&d) {
		if hasMatchDeviceID(d.DeviceID, deviceIdsFilter) {
			ids = append(ids, d.DeviceID)
		}
	}
	it.Close()
	if it.Err() != nil {
		return logAndReturnError(status.Errorf(codes.Internal, "cannot get user devices: %v", it.Err()))
	}

	for _, deviceID := range ids {
		err := srv.Send(&pb.UserDevice{DeviceId: deviceID, UserId: d.Owner})
		if err != nil {
			return logAndReturnError(status.Errorf(status.Convert(err).Code(), "cannot get user devices: %v", err))
		}
	}
	return nil
}

// GetUserDevices returns a list of user's devices if the access token is valid.
func (s *Service) GetUserDevices(request *pb.GetUserDevicesRequest, srv pb.AuthorizationService_GetUserDevicesServer) error {
	tx := s.persistence.NewTransaction(srv.Context())
	defer tx.Close()

	userIdsFilter := request.GetUserIdsFilter()
	if len(userIdsFilter) == 0 {
		token, err := grpc_auth.AuthFromMD(srv.Context(), "bearer")
		if err != nil {
			return logAndReturnError(status.Errorf(codes.InvalidArgument, "cannot add device: %v", err))
		}
		owner, err := grpc.ParseOwnerFromJwtToken(s.ownerClaim, token)
		if err != nil {
			log.Debugf("cannot parse '%v' from jwt token: %v", s.ownerClaim, err)
		}
		if owner == "" {
			return logAndReturnError(status.Errorf(codes.InvalidArgument, "cannot get user devices: invalid userIdsFilter"))
		}
		if owner == serviceOwner {
			return sendUserDevices(request, srv, tx.RetrieveAll)
		} else {
			userIdsFilter = []string{owner}
		}
	}

	// auth0 ->

	for _, owner := range userIdsFilter {
		err := sendUserDevices(request, srv, func() persistence.Iterator {
			return tx.RetrieveByOwner(owner)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

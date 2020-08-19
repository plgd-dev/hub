package service

import (
	"fmt"

	"github.com/plgd-dev/cloud/authorization/persistence"

	jwt "github.com/dgrijalva/jwt-go"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/kit/log"
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

type claims struct {
	Subject string `json:"sub,omitempty"`
}

func (c *claims) Valid() error {
	return nil
}

func logAndReturnError(err error) error {
	log.Errorf("%v", err)
	return err
}

func parseSubFromJwtToken(rawJwtToken string) (string, error) {
	parser := &jwt.Parser{
		SkipClaimsValidation: true,
	}

	var claims claims
	_, _, err := parser.ParseUnverified(rawJwtToken, &claims)
	if err != nil {
		return "", fmt.Errorf("cannot get subject from jwt token: %w", err)
	}

	if claims.Subject != "" {
		return claims.Subject, nil
	}

	return "", fmt.Errorf("cannot get subject from jwt token: not found")
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
		userID, err := parseSubFromJwtToken(token)
		if err != nil {
			log.Debugf("cannot parse user from jwt token: %v", err)
		}
		if userID == "" {
			return logAndReturnError(status.Errorf(codes.InvalidArgument, "cannot get user devices: invalid userIdsFilter"))
		}
		userIdsFilter = []string{userID}
	}

	deviceIdsFilter := make(map[string]bool)
	for _, deviceID := range request.GetDeviceIdsFilter() {
		deviceIdsFilter[deviceID] = true
	}

	for _, userID := range userIdsFilter {
		var ids []string
		it := tx.RetrieveAll(userID)
		var d persistence.AuthorizedDevice
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
			err := srv.Send(&pb.UserDevice{DeviceId: deviceID, UserId: userID})
			if err != nil {
				return logAndReturnError(status.Errorf(status.Convert(err).Code(), "cannot get user devices: %v", err))
			}
		}
	}
	return nil
}

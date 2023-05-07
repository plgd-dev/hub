package service

import (
	"context"
	"fmt"
	"math"
	"net/url"

	"github.com/plgd-dev/go-coap/v3/message"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/hub/v2/coap-gateway/coapconv"
	"github.com/plgd-dev/hub/v2/identity-store/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
)

var (
	queryAccessToken = "accesstoken"
	queryDeviceID    = "di"
	queryUserID      = "uid" // optional because it is not defined in a current specification => it must be determined from the access token
)

type signOffData struct {
	deviceID    string
	userID      string
	accessToken string
}

func getSignOffDataFromQuery(req *mux.Message) (signOffData, error) {
	queries, _ := req.Options().Queries()
	// from QUERY: di, accesstoken, uid
	var data signOffData
	for _, query := range queries {
		values, err := url.ParseQuery(query)
		if err != nil {
			return signOffData{}, err
		}
		if deviceID := values.Get(queryDeviceID); deviceID != "" {
			data.deviceID = deviceID
		}
		if accessToken := values.Get(queryAccessToken); accessToken != "" {
			data.accessToken = accessToken
		}
		if userID := values.Get(queryUserID); userID != "" {
			data.userID = userID
		}
	}

	return data, nil
}

// Update empty values
func (s signOffData) updateSignOffDataFromAuthContext(client *session) signOffData {
	authCurrentCtx, err := client.GetAuthorizationContext()
	if err != nil {
		client.Debugf("auth context not available: %w", err)
		return s
	}

	if s.deviceID == "" {
		s.deviceID = authCurrentCtx.GetDeviceID()
	}
	if s.userID == "" {
		s.userID = authCurrentCtx.GetUserID()
	}
	if s.accessToken == "" {
		s.accessToken = authCurrentCtx.GetAccessToken()
	}
	return s
}

func (s signOffData) validateSignOffData() error {
	if s.deviceID == "" {
		return fmt.Errorf("invalid device id")
	}
	if s.userID == "" {
		return fmt.Errorf("invalid user id")
	}
	if s.accessToken == "" {
		return fmt.Errorf("invalid access token")
	}
	return nil
}

const errFmtSignOff = "cannot handle sign off: %w"

// Sign-off
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.account.swagger.json
func signOffHandler(req *mux.Message, client *session) (*pool.Message, error) {
	signOffData, err := getSignOffDataFromQuery(req)
	if err != nil {
		return nil, statusErrorf(coapCodes.BadOption, errFmtSignOff, err)
	}

	// we need to get signOffData because of blocking sign off, because client can close connection and clear auth context
	signOffData = signOffData.updateSignOffDataFromAuthContext(client)
	if err = signOffData.validateSignOffData(); err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, errFmtSignOff, err)
	}

	// we need to use sever context because of blocking sign off, because client can close connection
	ctx, cancel := context.WithTimeout(client.server.ctx, client.server.config.APIs.COAP.KeepAlive.Timeout)
	defer cancel()

	err = client.blockSignOff.Acquire(ctx, math.MaxInt64)
	if err != nil {
		return nil, statusErrorf(coapCodes.ServiceUnavailable, errFmtSignOff, fmt.Errorf("cannot acquire sign off lock: some commands are in progress"))
	}
	defer client.blockSignOff.Release(math.MaxInt64)

	jwtClaims, err := client.ValidateToken(ctx, signOffData.accessToken)
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, errFmtSignOff, err)
	}

	err = client.server.VerifyDeviceID(client.tlsDeviceID, jwtClaims)
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, errFmtSignOff, err)
	}

	if err := jwtClaims.ValidateOwnerClaim(client.server.config.APIs.COAP.Authorization.OwnerClaim, signOffData.userID); err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, errFmtSignOff, err)
	}

	deviceID, err := client.ResolveDeviceID(jwtClaims, signOffData.deviceID)
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, errFmtSignOff, err)
	}
	setDeviceIDToTracerSpan(req.Context(), deviceID)

	ctx = kitNetGrpc.CtxWithToken(ctx, signOffData.accessToken)
	deviceIds := []string{deviceID}
	respRA, err := client.server.raClient.DeleteDevices(ctx, &commands.DeleteDevicesRequest{
		DeviceIds: deviceIds,
	})
	if err != nil {
		return nil, statusErrorf(coapconv.GrpcErr2CoapCode(err, coapconv.Delete), errFmtSignOff, err)
	}
	if len(respRA.GetDeviceIds()) != 1 {
		client.Errorf("sign off error: failed to remove documents for device('%v') from eventstore", deviceID)
	}

	client.unsubscribeFromDeviceEvents()
	respIS, err := client.server.isClient.DeleteDevices(ctx, &pb.DeleteDevicesRequest{
		DeviceIds: deviceIds,
	})
	if err != nil {
		return nil, statusErrorf(coapconv.GrpcErr2CoapCode(err, coapconv.Delete), errFmtSignOff, err)
	}
	if len(respIS.GetDeviceIds()) != 1 {
		return nil, statusErrorf(coapCodes.BadRequest, errFmtSignOff, fmt.Errorf("cannot remove device %v from user", deviceID))
	}

	client.CleanUp(true)
	return client.createResponse(coapCodes.Deleted, req.Token(), message.TextPlain, nil), nil
}

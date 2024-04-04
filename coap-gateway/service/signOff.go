package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"

	"github.com/plgd-dev/go-coap/v3/message"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	pbCA "github.com/plgd-dev/hub/v2/certificate-authority/pb"
	"github.com/plgd-dev/hub/v2/coap-gateway/coapconv"
	"github.com/plgd-dev/hub/v2/identity-store/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
)

const (
	queryAccessTokenKey = "accesstoken"
	queryDeviceIDKey    = "di"
	queryUserIDKey      = "uid" // optional because it is not defined in a current specification => it must be determined from the access token
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
		if deviceID := values.Get(queryDeviceIDKey); deviceID != "" {
			data.deviceID = deviceID
		}
		if accessToken := values.Get(queryAccessTokenKey); accessToken != "" {
			data.accessToken = accessToken
		}
		if userID := values.Get(queryUserIDKey); userID != "" {
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
		return errors.New("invalid device id")
	}
	if s.userID == "" {
		return errors.New("invalid user id")
	}
	if s.accessToken == "" {
		return errors.New("invalid access token")
	}
	return nil
}

func getSignOffDataFromClaims(ctx context.Context, client *session, sod signOffData) (string, error) {
	jwtClaims, err := client.ValidateToken(ctx, sod.accessToken)
	if err != nil {
		return "", err
	}

	if err = jwtClaims.ValidateOwnerClaim(client.server.config.APIs.COAP.Authorization.OwnerClaim, sod.userID); err != nil {
		return "", err
	}

	deviceID, err := client.server.VerifyAndResolveDeviceID(client.tlsDeviceID, sod.deviceID, jwtClaims)
	if err != nil {
		return "", err
	}

	return deviceID, nil
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
		return nil, statusErrorf(coapCodes.ServiceUnavailable, errFmtSignOff, errors.New("cannot acquire sign off lock: some commands are in progress"))
	}
	defer client.blockSignOff.Release(math.MaxInt64)

	deviceID, err := getSignOffDataFromClaims(ctx, client, signOffData)
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
	// CertificateAuthority
	respCA, err := client.server.certificateAuthorityClient.DeleteSigningRecords(ctx, &pbCA.DeleteSigningRecordsRequest{
		DeviceIdFilter: deviceIds,
	})
	if err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, errFmtSignOff, fmt.Errorf("cannot delete certificates linked to devices %v from CertificateAuthority: %w", deviceIds, err))
	}
	client.getLogger().Debugf("certificate records(num: %v) linked to %v devices has been deleted", respCA.GetCount(), deviceIds)

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

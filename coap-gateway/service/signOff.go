package service

import (
	"context"
	"fmt"
	"net/url"

	"github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/coap-gateway/coapconv"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
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
	queries, _ := req.Options.Queries()
	//from QUERY: di, accesstoken, uid
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

/// Update empty values
func (s signOffData) updateSignOffDataFromAuthContext(client *Client) signOffData {
	authCurrentCtx, err := client.GetAuthorizationContext()
	if err != nil {
		log.Debugf("auth context not available: %w", err)
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

// Sign-off
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.account.swagger.json
func signOffHandler(req *mux.Message, client *Client) {
	logErrorAndCloseClient := func(err error, code coapCodes.Code) {
		client.logAndWriteErrorResponse(err, code, req.Token)
		if err := client.Close(); err != nil {
			log.Errorf("sign off error: %w", err)
		}
	}

	ctx, cancel := context.WithTimeout(client.server.ctx, client.server.config.APIs.COAP.KeepAlive.Timeout)
	defer cancel()

	signOffData, err := getSignOffDataFromQuery(req)
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign off: %w", err), coapCodes.BadOption)
		return
	}

	signOffData = signOffData.updateSignOffDataFromAuthContext(client)
	if err = signOffData.validateSignOffData(); err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign off: %w", err), coapCodes.BadRequest)
		return
	}

	jwtClaims, err := client.ValidateToken(req.Context, signOffData.accessToken)
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign off: %v", err), coapCodes.Unauthorized)
		return
	}

	err = client.server.VerifyDeviceID(client.tlsDeviceID, jwtClaims)
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign off: %w", err), coapCodes.Unauthorized)
		return
	}

	if err := jwtClaims.ValidateOwnerClaim(client.server.config.Clients.AuthServer.OwnerClaim, signOffData.userID); err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign off: %v", err), coapCodes.Unauthorized)
		return
	}

	deviceID := client.ResolveDeviceID(jwtClaims, signOffData.deviceID)

	ctx = kitNetGrpc.CtxWithToken(ctx, signOffData.accessToken)
	deviceIds := []string{deviceID}
	respRA, err := client.server.raClient.DeleteDevices(ctx, &commands.DeleteDevicesRequest{
		DeviceIds: deviceIds,
	})
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign off: %w", err), coapconv.GrpcErr2CoapCode(err, coapconv.Delete))
		return
	}
	if len(respRA.GetDeviceIds()) != 1 {
		log.Errorf("sign off error: failed to remove documents for device('%v') from eventstore", deviceID)
	}

	client.unsubscribeFromDeviceEvents()
	respAS, err := client.server.asClient.DeleteDevices(ctx, &pb.DeleteDevicesRequest{
		DeviceIds: deviceIds,
		UserId:    signOffData.userID,
	})
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign off: %w", err), coapconv.GrpcErr2CoapCode(err, coapconv.Delete))
		return
	}
	if len(respAS.GetDeviceIds()) != 1 {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign off: cannot remove device %v from user", deviceID), coapCodes.InternalServerError)
		return
	}

	client.CleanUp()
	client.sendResponse(coapCodes.Deleted, req.Token, message.TextPlain, nil)
}

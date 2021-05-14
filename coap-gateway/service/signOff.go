package service

import (
	"context"
	"fmt"
	"net/url"

	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/coap-gateway/coapconv"
	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"google.golang.org/grpc/status"
)

var (
	queryAccessToken = "accesstoken"
	queryDeviceID    = "di"
	queryUserID      = "uid" // optional because it is not defined in a current specification => it must be determined from the access token
)

func validateSignOff(deviceID, accessToken string) error {
	if deviceID == "" {
		return fmt.Errorf("invalid di('%v')", deviceID)
	}
	if accessToken == "" {
		return fmt.Errorf("invalid accesstoken('%v')", accessToken)
	}
	return nil
}

// Sign-off
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.account.swagger.json
func signOffHandler(req *mux.Message, client *Client) {
	//from QUERY: di, accesstoken
	var deviceID string
	var accessToken string
	var userID string

	ctx, cancel := context.WithTimeout(client.server.ctx, client.server.config.APIs.COAP.KeepAlive.Timeout)
	defer cancel()

	queries, _ := req.Options.Queries()
	for _, query := range queries {
		values, err := url.ParseQuery(query)
		if err != nil {
			client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign off: %w", err), coapCodes.BadOption, req.Token)
			return
		}
		if di := values.Get(queryDeviceID); di != "" {
			deviceID = di
		}

		if at := values.Get(queryAccessToken); at != "" {
			accessToken = at
		}

		if uid := values.Get(queryUserID); uid != "" {
			userID = uid
		}
	}
	authCurrentCtx, _ := client.GetAuthorizationContext()
	if userID == "" {
		userID = authCurrentCtx.GetUserID()
	}
	if deviceID == "" {
		deviceID = authCurrentCtx.GetDeviceID()
	}
	if accessToken == "" {
		accessToken = authCurrentCtx.GetAccessToken()
	}

	err := validateSignOff(deviceID, accessToken)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign off for %v: %w", deviceID, err), coapCodes.BadRequest, req.Token)
		return
	}
	_, err = client.server.asClient.SignOff(ctx, &pbAS.SignOffRequest{
		DeviceId:    deviceID,
		UserId:      userID,
		AccessToken: accessToken,
	})
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign off for %v: %w", deviceID, err), coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapconv.Delete), req.Token)
		return
	}
	client.CleanUp()
	client.sendResponse(coapCodes.Deleted, req.Token, message.TextPlain, nil)
}

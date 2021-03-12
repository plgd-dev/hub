package service

import (
	"context"
	"fmt"
	"io"
	"time"

	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/coap-gateway/coapconv"
	deviceStatus "github.com/plgd-dev/cloud/coap-gateway/schema/device/status"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/kit/codec/cbor"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/net/coap"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CoapSignInReq struct {
	DeviceID    string `json:"di"`
	UserID      string `json:"uid"`
	AccessToken string `json:"accesstoken"`
	Login       bool   `json:"login"`
}

type CoapSignInResp struct {
	ExpiresIn int64 `json:"expiresin"`
}

func registerObservationsForPublishedResources(ctx context.Context, client *Client, deviceID string) {
	getResourceLinksClient, err := client.server.rdClient.GetResourceLinks(ctx, &pb.GetResourceLinksRequest{
		DeviceIdsFilter: []string{deviceID},
	})
	if err != nil {
		if status.Convert(err).Code() == codes.NotFound {
			return
		}
		log.Errorf("signIn: cannot get resource links for the device %v: %v", deviceID, err)
		return
	}
	for {
		m, err := getResourceLinksClient.Recv()
		if err == io.EOF {
			break
		}
		if status.Convert(err).Code() == codes.NotFound {
			return
		}
		if err != nil {
			log.Errorf("signIn: cannot receive link for the device %v: %v", deviceID, err)
			return
		}
		resource := m.ToRAProto()
		client.observeResource(ctx, resource.GetResourceID(), resource.IsObservable(), true)
	}
}

// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.session.swagger.json
func signInPostHandler(req *mux.Message, client *Client, signIn CoapSignInReq) {
	resp, err := client.server.asClient.SignIn(req.Context, &pbAS.SignInRequest{
		DeviceId:    signIn.DeviceID,
		UserId:      signIn.UserID,
		AccessToken: signIn.AccessToken,
	})
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: %w", err), coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapconv.Update), req.Token)
		return
	}

	coapResp := CoapSignInResp{
		ExpiresIn: resp.ExpiresIn,
	}
	var expired time.Time
	if resp.ExpiresIn > 0 {
		expired = time.Now().Add(time.Second * time.Duration(resp.ExpiresIn))
	}

	accept := coap.GetAccept(req.Options)
	encode, err := coap.GetEncoder(accept)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: %w", err), coapCodes.InternalServerError, req.Token)
		return
	}
	out, err := encode(coapResp)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: %w", err), coapCodes.InternalServerError, req.Token)
		return
	}

	authCtx := authorizationContext{
		DeviceID:    signIn.DeviceID,
		UserID:      signIn.UserID,
		AccessToken: signIn.AccessToken,
		Expire:      expired,
	}
	serviceToken, err := client.server.oauthMgr.GetToken(req.Context)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot get service token: %w", err), coapCodes.InternalServerError, req.Token)
		client.Close()
		return
	}
	req.Context = kitNetGrpc.CtxWithUserID(kitNetGrpc.CtxWithToken(req.Context, serviceToken.AccessToken), authCtx.GetUserID())
	err = deviceStatus.Publish(req.Context, client.server.raClient, signIn.DeviceID, &commands.CommandMetadata{
		Sequence:     client.coapConn.Sequence(),
		ConnectionId: client.remoteAddrString(),
	})
	if err != nil {
		// Events from resources of device will be comes but device is offline. To recover cloud state, client need to reconnect to cloud.
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: cannot publish cloud device status: %w", err), coapCodes.InternalServerError, req.Token)
		client.Close()
		return
	}

	oldAuthCtx := client.SetAuthorizationContext(&authCtx)
	err = client.server.devicesStatusUpdater.Add(client)
	if err != nil {
		// Events from resources of device will be comes but device is offline. To recover cloud state, client need to reconnect to cloud.
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: cannot update cloud device status: %w", err), coapCodes.InternalServerError, req.Token)
		client.Close()
		return
	}

	newDevice := false

	switch {
	case oldAuthCtx.GetDeviceID() == "":
		newDevice = true
	case oldAuthCtx.GetDeviceID() != signIn.DeviceID || oldAuthCtx.GetUserID() != signIn.UserID:
		client.cancelResourceSubscriptions(true)
		client.cancelDeviceSubscriptions(req.Context)
		newDevice = true
	}

	if newDevice {
		h := NewDeviceSubscriptionHandlers(client, signIn.DeviceID)
		ctx := kitNetGrpc.CtxWithUserID(kitNetGrpc.CtxWithToken(client.server.ctx, serviceToken.AccessToken), signIn.UserID)
		cancelSubscription, err := client.server.subscribeToDevice(ctx, signIn.UserID, signIn.DeviceID, h)
		if err != nil {
			client.logAndWriteErrorResponse(fmt.Errorf("cannot create device %v pending subscription: %w", signIn.DeviceID, err), coapCodes.InternalServerError, req.Token)
			client.Close()
			return
		}
		if !client.storeDeviceSubscription(cancelSubscription) {
			cancelSubscription(req.Context)
		}
	}
	if expired.IsZero() {
		client.server.expirationClientCache.Set(signIn.DeviceID, nil, time.Millisecond)
	} else {
		client.server.expirationClientCache.Set(signIn.DeviceID, client, time.Second*time.Duration(resp.ExpiresIn))
	}
	client.sendResponse(coapCodes.Changed, req.Token, accept, out)

	// try to register observations to the device for published resources at the cloud.
	registerObservationsForPublishedResources(req.Context, client, signIn.DeviceID)
}

func signOutPostHandler(req *mux.Message, client *Client, signOut CoapSignInReq) {
	// fix for iotivity-classic
	authCurrentCtx, _ := client.GetAuthorizationContext()
	userID := signOut.UserID
	deviceID := signOut.DeviceID
	if userID == "" {
		userID = authCurrentCtx.GetUserID()
	}
	if deviceID == "" {
		deviceID = authCurrentCtx.GetDeviceID()
	}

	_, err := client.server.asClient.SignOut(req.Context, &pbAS.SignOutRequest{
		DeviceId:    deviceID,
		UserId:      userID,
		AccessToken: signOut.AccessToken,
	})
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign out: %w", err), coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapconv.Update), req.Token)
		client.Close()
		return
	}
	oldAuthCtx := client.CleanUp()
	if oldAuthCtx.GetDeviceID() != "" {
		serviceToken, err := client.server.oauthMgr.GetToken(req.Context)
		if err != nil {
			client.logAndWriteErrorResponse(fmt.Errorf("cannot get service token: %w", err), coapCodes.InternalServerError, req.Token)
			client.Close()
			return
		}
		ctx := kitNetGrpc.CtxWithToken(req.Context, serviceToken.AccessToken)
		client.server.expirationClientCache.Set(oldAuthCtx.GetDeviceID(), nil, time.Millisecond)
		req.Context = kitNetGrpc.CtxWithUserID(ctx, oldAuthCtx.GetUserID())
		err = deviceStatus.SetOffline(req.Context, client.server.raClient, oldAuthCtx.GetDeviceID(), &commands.CommandMetadata{
			Sequence:     client.coapConn.Sequence(),
			ConnectionId: client.remoteAddrString(),
		})
		if err != nil {
			// Device will be still reported as online and it can fix his state by next calls online, offline commands.
			log.Errorf("DeviceId %v: cannot handle sign out: cannot update cloud device status: %v", oldAuthCtx.GetDeviceID(), err)
			return
		}
	}

	client.sendResponse(coapCodes.Changed, req.Token, message.AppOcfCbor, []byte{0xA0}) // empty object
}

// Sign-in
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.session.swagger.json
func signInHandler(req *mux.Message, client *Client) {
	switch req.Code {
	case coapCodes.POST:
		var signIn CoapSignInReq
		err := cbor.ReadFrom(req.Body, &signIn)
		if err != nil {
			client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: %w", err), coapCodes.BadRequest, req.Token)
			return
		}
		switch signIn.Login {
		case true:
			signInPostHandler(req, client, signIn)
		default:
			signOutPostHandler(req, client, signIn)
		}
	default:
		client.logAndWriteErrorResponse(fmt.Errorf("forbidden request from %v", client.remoteAddrString()), coapCodes.Forbidden, req.Token)
	}
}

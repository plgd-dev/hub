package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"time"

	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/coap-gateway/coapconv"
	grpcgwClient "github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/go-coap/v2/message"
	coapCodes "github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/kit/codec/cbor"
	"golang.org/x/oauth2"
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

func (client *Client) registerObservationsForPublishedResourcesLocked(ctx context.Context, deviceID string) {
	getResourceLinksClient, err := client.server.rdClient.GetResourceLinks(ctx, &pb.GetResourceLinksRequest{
		DeviceIdFilter: []string{deviceID},
	})
	if err != nil {
		if status.Convert(err).Code() == codes.NotFound {
			return
		}
		log.Errorf("signIn: cannot get resource links for the device %v: %w", deviceID, err)
		return
	}
	resources := make([]*commands.Resource, 0, 8)
	for {
		m, err := getResourceLinksClient.Recv()
		if err == io.EOF {
			break
		}
		if status.Convert(err).Code() == codes.NotFound {
			return
		}
		if err != nil {
			log.Errorf("signIn: cannot receive link for the device %v: %w", deviceID, err)
			return
		}
		resources = append(resources, m.GetResources()...)

	}
	client.observeResourcesLocked(ctx, resources)
}

func (client *Client) loadShadowSynchronization(ctx context.Context, deviceID string) error {
	deviceMetadataClient, err := client.server.rdClient.GetDevicesMetadata(ctx, &pb.GetDevicesMetadataRequest{
		DeviceIdFilter: []string{deviceID},
	})
	if err != nil {
		if status.Convert(err).Code() == codes.NotFound {
			return nil
		}
		return fmt.Errorf("cannot get device(%v) metdata: %v", deviceID, err)
	}
	shadowSynchronization := commands.ShadowSynchronization_UNSET
	for {
		m, err := deviceMetadataClient.Recv()
		if err == io.EOF {
			break
		}
		if status.Convert(err).Code() == codes.NotFound {
			return nil
		}
		if err != nil {
			return fmt.Errorf("cannot get device(%v) metdata: %v", deviceID, err)
		}
		shadowSynchronization = m.GetShadowSynchronization()
	}
	client.observedResourcesLock.Lock()
	defer client.observedResourcesLock.Unlock()
	client.shadowSynchronization = shadowSynchronization
	return nil
}

func getValidUntilByOAuth(ctx context.Context, service *Service, signIn CoapSignInReq) (time.Time, error) {
	if err := checkReq(signIn); err != nil {
		return time.Time{}, fmt.Errorf("cannot handle sign in: %v", err)
	}

	oauthClient := service.provider.OAuth2.Client(ctx, &oauth2.Token{
		AccessToken: signIn.AccessToken,
		TokenType:   "bearer",
	})
	resp, err := oauthClient.Get(service.provider.OAuth2.Endpoint.AuthURL + "/userinfo")
	if err != nil {
		return time.Time{}, fmt.Errorf("failed oauth userinfo request: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Errorf("signIn: failed to close response body: %v", err)
		}
	}()

	var profile map[string]interface{}
	if err = json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return time.Time{}, fmt.Errorf("failed to decode userinfo request body: %v", err)
	}

	v, ok := profile[service.config.Clients.AuthServer.OwnerClaim]
	if !ok {
		return time.Time{}, fmt.Errorf("invalid userinfo response: %v", "ownerClaim not set")
	}
	userID, ok := v.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("invalid userinfo response: %v", "invalid ownerClaim value")
	}
	if userID != signIn.UserID {
		return time.Time{}, fmt.Errorf("invalid ownerClaim: %v", signIn.UserID)
	}

	v, ok = profile["exp"]
	if !ok {
		return time.Time{}, nil
	}

	exp, ok := v.(float64) // all integers are float64 in json
	if !ok {
		return time.Time{}, fmt.Errorf("invalid userinfo response: %v", "invalid exp value")
	}
	return time.Unix(int64(exp), 0), nil
}

func getContent(expiresIn int64, options message.Options) (message.MediaType, []byte, error) {
	coapResp := CoapSignInResp{
		ExpiresIn: expiresIn,
	}

	accept := coapconv.GetAccept(options)
	encode, err := coapconv.GetEncoder(accept)
	if err != nil {
		return 0, nil, fmt.Errorf("cannot handle sign in: %w", err)
	}
	out, err := encode(coapResp)
	if err != nil {
		return 0, nil, fmt.Errorf("cannot handle sign in: %w", err)

	}
	return accept, out, nil
}

func setNewDeviceSubscriber(req *mux.Message, client *Client, signIn CoapSignInReq) error {
	err := client.loadShadowSynchronization(req.Context, signIn.DeviceID)
	if err != nil {
		return fmt.Errorf("cannot load shadow synchronization for device %v: %w", signIn.DeviceID, err)
	}

	deviceSubscriber, err := grpcgwClient.NewDeviceSubscriber(client.GetContext, signIn.DeviceID, func() func() (when time.Time, err error) {
		var count uint64
		maxRand := client.server.config.APIs.COAP.KeepAlive.Timeout / 2
		if maxRand <= 0 {
			maxRand = time.Second * 10
		}
		return func() (when time.Time, err error) {
			count++
			r := rand.Int63n(int64(maxRand) / 2)
			next := time.Now().Add(client.server.config.APIs.COAP.KeepAlive.Timeout + time.Duration(r))
			log.Debugf("next iteration %v of retrying reconnect to grpc-client for deviceID %v will be at %v", count, signIn.DeviceID, next)
			return next, nil
		}
	}, client.server.rdClient, client.server.resourceSubscriber)
	if err != nil {
		return fmt.Errorf("cannot create device subscription for device %v: %w", signIn.DeviceID, err)
	}
	oldDeviceSubscriber := client.replaceDeviceSubscriber(deviceSubscriber)
	if oldDeviceSubscriber != nil {
		if err = oldDeviceSubscriber.Close(); err != nil {
			log.Errorf("failed to close replaced device subscriber: %v", err)
		}
	}
	h := grpcgwClient.NewDeviceSubscriptionHandlers(client)
	deviceSubscriber.SubscribeToPendingCommands(req.Context, h)
	return nil
}

// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.session.swagger.json
func signInPostHandler(req *mux.Message, client *Client, signIn CoapSignInReq) {
	ctx := context.WithValue(req.Context, oauth2.HTTPClient, client.server.provider.HTTPClient.HTTP())

	logErrorAndcloseClient := func(err error) {
		client.logAndWriteErrorResponse(err, coapCodes.InternalServerError, req.Token)
		if err := client.Close(); err != nil {
			log.Errorf("sign in error: %w", err)
		}
	}

	validUntil, err := getValidUntilByOAuth(ctx, client.server, signIn)
	if err != nil {
		logErrorAndcloseClient(err)
		return
	}

	expiresIn := validUntilToExpiresIn(validUntil)

	accept, out, err := getContent(expiresIn, req.Options)
	if err != nil {
		logErrorAndcloseClient(err)
		return
	}

	serviceToken, err := client.server.oauthMgr.GetToken(req.Context)
	if err != nil {
		logErrorAndcloseClient(fmt.Errorf("cannot get service token: %w", err))
		return
	}
	authCtx := authorizationContext{
		DeviceID:    signIn.DeviceID,
		UserID:      signIn.UserID,
		AccessToken: signIn.AccessToken,
		Expire:      validUntil,
	}
	req.Context = kitNetGrpc.CtxWithOwner(kitNetGrpc.CtxWithToken(req.Context, serviceToken.AccessToken), authCtx.GetUserID())

	oldAuthCtx := client.SetAuthorizationContext(&authCtx)
	err = client.server.devicesStatusUpdater.Add(client)
	if err != nil {
		// Events from resources of device will be comes but device is offline. To recover cloud state, client need to reconnect to cloud.
		logErrorAndcloseClient(fmt.Errorf("cannot handle sign in: cannot update cloud device status: %w", err))
		return
	}

	newDevice := false

	switch {
	case oldAuthCtx.GetDeviceID() == "":
		newDevice = true
	case oldAuthCtx.GetDeviceID() != signIn.DeviceID || oldAuthCtx.GetUserID() != signIn.UserID:
		client.cancelResourceSubscriptions(true)
		client.closeDeviceSubscriber()
		newDevice = true
		client.cleanObservedResources()
	}

	if newDevice {
		if err := setNewDeviceSubscriber(req, client, signIn); err != nil {
			logErrorAndcloseClient(err)
			return
		}
	}
	if validUntil.IsZero() {
		client.server.expirationClientCache.Set(signIn.DeviceID, nil, time.Millisecond)
	} else {
		client.server.expirationClientCache.Set(signIn.DeviceID, client, time.Second*time.Duration(expiresIn))
	}
	client.sendResponse(coapCodes.Changed, req.Token, accept, out)

	// try to register observations to the device for published resources at the cloud.
	client.server.taskQueue.Submit(func() {
		client.observedResourcesLock.Lock()
		defer client.observedResourcesLock.Unlock()
		if client.shadowSynchronization == commands.ShadowSynchronization_DISABLED {
			return
		}
		client.registerObservationsForPublishedResourcesLocked(req.Context, signIn.DeviceID)
	})
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
		req.Context = kitNetGrpc.CtxWithOwner(ctx, oldAuthCtx.GetUserID())

		_, err = client.server.raClient.UpdateDeviceMetadata(req.Context, &commands.UpdateDeviceMetadataRequest{
			DeviceId: oldAuthCtx.GetDeviceID(),
			Update: &commands.UpdateDeviceMetadataRequest_Status{
				Status: &commands.ConnectionStatus{
					Value: commands.ConnectionStatus_OFFLINE,
				},
			},
			CommandMetadata: &commands.CommandMetadata{
				Sequence:     client.coapConn.Sequence(),
				ConnectionId: client.remoteAddrString(),
			},
		})
		if err != nil {
			// Device will be still reported as online and it can fix his state by next calls online, offline commands.
			log.Errorf("DeviceId %v: cannot handle sign out: cannot update cloud device status: %w", oldAuthCtx.GetDeviceID(), err)
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

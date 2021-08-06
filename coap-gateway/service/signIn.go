package service

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"time"

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

/// Check that all required request fields are set
func (request CoapSignInReq) checkOAuthRequest() error {
	if request.UserID == "" {
		return fmt.Errorf("invalid user id")
	}
	if request.AccessToken == "" {
		return fmt.Errorf("invalid access token")
	}
	return nil
}

/// Update empty values
func (request CoapSignInReq) updateOAUthRequestIfEmpty(deviceID, userID, accessToken string) CoapSignInReq {
	if request.DeviceID == "" {
		request.DeviceID = deviceID
	}
	if request.UserID == "" {
		request.UserID = userID
	}
	if request.AccessToken == "" {
		request.AccessToken = accessToken
	}
	return request
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

/// Get data for sign in response
func getSignInContent(expiresIn int64, options message.Options) (message.MediaType, []byte, error) {
	coapResp := CoapSignInResp{
		ExpiresIn: expiresIn,
	}

	accept := coapconv.GetAccept(options)
	encode, err := coapconv.GetEncoder(accept)
	if err != nil {
		return 0, nil, err
	}
	out, err := encode(coapResp)
	if err != nil {
		return 0, nil, err

	}
	return accept, out, nil
}

func setNewDeviceSubscriber(ctx context.Context, client *Client, deviceID string) error {
	deviceSubscriber, err := grpcgwClient.NewDeviceSubscriber(client.GetContext, deviceID, func() func() (when time.Time, err error) {
		var count uint64
		maxRand := client.server.config.APIs.COAP.KeepAlive.Timeout / 2
		if maxRand <= 0 {
			maxRand = time.Second * 10
		}
		return func() (when time.Time, err error) {
			count++
			r := rand.Int63n(int64(maxRand) / 2)
			next := time.Now().Add(client.server.config.APIs.COAP.KeepAlive.Timeout + time.Duration(r))
			log.Debugf("next iteration %v of retrying reconnect to grpc-client for deviceID %v will be at %v", count, deviceID, next)
			return next, nil
		}
	}, client.server.rdClient, client.server.resourceSubscriber)
	if err != nil {
		return fmt.Errorf("cannot create device subscription for device %v: %w", deviceID, err)
	}
	oldDeviceSubscriber := client.replaceDeviceSubscriber(deviceSubscriber)
	if oldDeviceSubscriber != nil {
		if err = oldDeviceSubscriber.Close(); err != nil {
			log.Errorf("failed to close replaced device subscriber: %v", err)
		}
	}
	h := grpcgwClient.NewDeviceSubscriptionHandlers(client)
	deviceSubscriber.SubscribeToPendingCommands(ctx, h)
	return nil
}

// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.session.swagger.json
func signInPostHandler(req *mux.Message, client *Client, signIn CoapSignInReq) {
	logErrorAndCloseClient := func(err error, code coapCodes.Code) {
		if isTempError(err) {
			code = coapCodes.ServiceUnavailable
		}
		client.logAndWriteErrorResponse(err, code, req.Token)
		if err := client.Close(); err != nil {
			log.Errorf("sign in error: %w", err)
		}
	}

	if err := signIn.checkOAuthRequest(); err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign in: %v", err), coapCodes.BadRequest)
		return
	}

	jwtClaims, err := client.ValidateToken(req.Context, signIn.AccessToken)
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign in: %v", err), coapCodes.InternalServerError)
		return
	}

	if err := jwtClaims.validateOwnerClaim(client.server.config.Clients.AuthServer.OwnerClaim, signIn.UserID); err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign in: %v", err), coapCodes.InternalServerError)
		return
	}

	validUntil, err := jwtClaims.getExpirationTime()
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign in: %v", err), coapCodes.InternalServerError)
		return
	}

	expiresIn := validUntilToExpiresIn(validUntil)

	accept, out, err := getSignInContent(expiresIn, req.Options)
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign in: %v", err), coapCodes.InternalServerError)
		return
	}

	authCtx := authorizationContext{
		DeviceID:    signIn.DeviceID,
		UserID:      signIn.UserID,
		AccessToken: signIn.AccessToken,
		Expire:      validUntil,
	}
	req.Context = kitNetGrpc.CtxWithToken(req.Context, signIn.AccessToken)

	oldAuthCtx := client.SetAuthorizationContext(&authCtx)
	err = client.server.devicesStatusUpdater.Add(client)
	if err != nil {
		// Events from resources of device will be comes but device is offline. To recover cloud state, client need to reconnect to cloud.
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign in: cannot update cloud device status: %w", err), coapCodes.InternalServerError)
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
		if err := client.loadShadowSynchronization(req.Context, signIn.DeviceID); err != nil {
			logErrorAndCloseClient(fmt.Errorf("cannot load shadow synchronization for device %v: %w", signIn.DeviceID, err), coapCodes.InternalServerError)
			return
		}

		if err := setNewDeviceSubscriber(req.Context, client, signIn.DeviceID); err != nil {
			logErrorAndCloseClient(fmt.Errorf("cannot handle sign in: %v", err), coapCodes.InternalServerError)
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
	if err := client.server.taskQueue.Submit(func() {
		client.observedResourcesLock.Lock()
		defer client.observedResourcesLock.Unlock()
		if client.shadowSynchronization == commands.ShadowSynchronization_DISABLED {
			return
		}
		client.registerObservationsForPublishedResourcesLocked(req.Context, signIn.DeviceID)
	}); err != nil {
		log.Errorf("sign in error: failed to register resource observations for device %v: %v", signIn.DeviceID, err)
	}
}

func updateDeviceMetadata(req *mux.Message, client *Client) error {
	oldAuthCtx := client.CleanUp()
	if oldAuthCtx.GetDeviceID() != "" {
		ctx := kitNetGrpc.CtxWithToken(req.Context, oldAuthCtx.GetAccessToken())
		client.server.expirationClientCache.Set(oldAuthCtx.GetDeviceID(), nil, time.Millisecond)

		_, err := client.server.raClient.UpdateDeviceMetadata(ctx, &commands.UpdateDeviceMetadataRequest{
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
			return fmt.Errorf("DeviceId %v: cannot update cloud device status: %w", oldAuthCtx.GetDeviceID(), err)
		}
	}
	return nil
}

func signOutPostHandler(req *mux.Message, client *Client, signOut CoapSignInReq) {
	logErrorAndCloseClient := func(err error, code coapCodes.Code) {
		client.logAndWriteErrorResponse(err, code, req.Token)
		if err := client.Close(); err != nil {
			log.Errorf("sign out error: %w", err)
		}
	}

	if signOut.DeviceID == "" || signOut.UserID == "" || signOut.AccessToken == "" {
		authCurrentCtx, err := client.GetAuthorizationContext()
		if err != nil {
			logErrorAndCloseClient(fmt.Errorf("cannot handle sign out: %v", err), coapCodes.InternalServerError)
			return
		}
		signOut = signOut.updateOAUthRequestIfEmpty(authCurrentCtx.DeviceID, authCurrentCtx.UserID, authCurrentCtx.AccessToken)
	}

	if err := signOut.checkOAuthRequest(); err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign out: %v", err), coapCodes.BadRequest)
		return
	}

	jwtClaims, err := client.ValidateToken(req.Context, signOut.AccessToken)
	if err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign out: %v", err), coapCodes.InternalServerError)
		return
	}

	if err := jwtClaims.validateOwnerClaim(client.server.config.Clients.AuthServer.OwnerClaim, signOut.UserID); err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign out: %v", err), coapCodes.InternalServerError)
		return
	}

	if err := updateDeviceMetadata(req, client); err != nil {
		logErrorAndCloseClient(fmt.Errorf("cannot handle sign out: %v", err), coapCodes.InternalServerError)
		return
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

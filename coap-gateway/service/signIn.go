package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/plgd-dev/go-coap/v3/message"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/hub/v2/coap-gateway/coapconv"
	grpcgwClient "github.com/plgd-dev/hub/v2/grpc-gateway/client"
	"github.com/plgd-dev/hub/v2/identity-store/events"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/kit/v2/codec/cbor"
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

// Check that all required request fields are set
func (request CoapSignInReq) checkOAuthRequest() error {
	if request.DeviceID == "" {
		return errors.New("invalid device id")
	}
	if request.UserID == "" {
		return errors.New("invalid user id")
	}
	if request.AccessToken == "" {
		return errors.New("invalid access token")
	}
	return nil
}

// Update empty values
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

// Get data for sign in response
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

func setNewDeviceSubscriber(ctx context.Context, client *session, owner, deviceID string) error {
	getContext := func() (context.Context, context.CancelFunc) {
		return client.GetContext(), func() {
			// no-op
		}
	}

	deviceSubscriber, err := grpcgwClient.NewDeviceSubscriber(getContext, owner, deviceID,
		func() func() (when time.Time, err error) {
			var count uint64
			delayFn := pkgTime.GetRandomDelayGenerator(client.server.config.APIs.COAP.KeepAlive.Timeout / 4)
			return func() (when time.Time, err error) {
				count++
				next := time.Now().Add(client.server.config.APIs.COAP.KeepAlive.Timeout + delayFn())
				client.Debugf("next iteration %v of retrying reconnect to grpc-client will be at %v", count, next)
				return next, nil
			}
		}, client.server.rdClient, client.server.resourceSubscriber, client.server.tracerProvider)
	if err != nil {
		return fmt.Errorf("cannot create device subscription for device %v: %w", deviceID, err)
	}
	oldDeviceSubscriber := client.replaceDeviceSubscriber(deviceSubscriber)
	if oldDeviceSubscriber != nil {
		if err = oldDeviceSubscriber.Close(); err != nil {
			client.Errorf("failed to close replaced device subscriber: %v", err)
		}
	}
	h := grpcgwClient.NewDeviceSubscriptionHandlers(client)
	deviceSubscriber.SubscribeToPendingCommands(ctx, h)
	return nil
}

type updateType int

const (
	updateTypeNone    updateType = 0
	updateTypeNew     updateType = 1
	updateTypeChanged updateType = 2
)

func (c *session) updateAuthorizationContext(deviceID, userID, accessToken string, validUntil time.Time) updateType {
	authCtx := authorizationContext{
		DeviceID:    deviceID,
		UserID:      userID,
		AccessToken: accessToken,
		Expire:      validUntil,
	}
	oldAuthCtx := c.SetAuthorizationContext(&authCtx)

	if oldAuthCtx.GetDeviceID() == "" {
		return updateTypeNew
	}
	if oldAuthCtx.GetDeviceID() != deviceID || oldAuthCtx.GetUserID() != userID {
		return updateTypeChanged
	}
	return updateTypeNone
}

func (c *session) updateBySignInData(ctx context.Context, upd updateType, deviceId, owner string) (*commands.UpdateDeviceMetadataResponse, error) {
	if upd == updateTypeChanged {
		c.cancelResourceSubscriptions(true)
		if err := c.closeDeviceSubscriber(); err != nil {
			c.Errorf("failed to close previous device subscription: %w", err)
		}
		if err := c.closeDeviceObserver(c.Context()); err != nil {
			c.Errorf("failed to close previous device observer: %w", err)
		}
		c.unsubscribeFromDeviceEvents()
	}

	if upd != updateTypeNone {
		if err := setNewDeviceSubscriber(ctx, c, owner, deviceId); err != nil {
			return nil, fmt.Errorf("cannot set device subscriber: %w", err)
		}
	}

	if upd == updateTypeNew {
		resp, err := c.server.devicesStatusUpdater.UpdateOnlineStatus(ctx, c)
		if err != nil {
			return nil, fmt.Errorf("cannot update cloud device status: %w", err)
		}
		return resp, nil
	}

	return nil, nil
}

func subscribeToDeviceEvents(client *session, owner, deviceID string) error {
	if err := client.subscribeToDeviceEvents(owner, func(e *events.Event) {
		evt := e.GetDevicesUnregistered()
		if evt == nil {
			return
		}
		if evt.GetOwner() != owner {
			return
		}
		if !slices.Contains(evt.GetDeviceIds(), deviceID) {
			return
		}
		client.Close()
	}); err != nil {
		return fmt.Errorf("cannot subscribe to device events: %w", err)
	}
	return nil
}

func subscribeAndValidateDeviceAccess(ctx context.Context, client *session, owner, deviceID string, subscribe bool) (bool, error) {
	// subscribe to updates before checking cache, so when the device gets removed during sign in
	// the client will always be closed
	if subscribe {
		if err := subscribeToDeviceEvents(client, owner, deviceID); err != nil {
			return false, err
		}
	}

	return client.server.ownerCache.OwnsDevice(ctx, deviceID)
}

func signInError(err error) error {
	return fmt.Errorf("sign in error: %w", err)
}

func (c *session) resolveTwinEnabled(ctx context.Context, updateDeviceMetadataResp *commands.UpdateDeviceMetadataResponse) bool {
	twinEnabled := true
	if updateDeviceMetadataResp != nil {
		twinEnabled = updateDeviceMetadataResp.GetTwinEnabled()
	} else {
		deviceObs, ok, _ := c.getDeviceObserver(ctx)
		if ok {
			twinEnabled = deviceObs.GetTwinEnabled()
		}
	}
	return twinEnabled
}

func getSignInDataFromClaims(ctx context.Context, client *session, signIn CoapSignInReq) (string, time.Time, error) {
	jwtClaims, err := client.ValidateToken(ctx, signIn.AccessToken)
	if err != nil {
		return "", time.Time{}, err
	}

	if err = jwtClaims.ValidateOwnerClaim(client.server.config.APIs.COAP.Authorization.OwnerClaim, signIn.UserID); err != nil {
		return "", time.Time{}, err
	}

	deviceID, err := client.server.VerifyAndResolveDeviceID(client.tlsDeviceID, signIn.DeviceID, jwtClaims)
	if err != nil {
		return "", time.Time{}, err
	}

	expTime, _ := jwtClaims.GetExpirationTime()
	validUntil := time.Time{}
	if expTime != nil {
		if time.Until(expTime.Time) < 2*client.server.config.APIs.COAP.OwnerCacheExpiration {
			return "", time.Time{}, fmt.Errorf("access token will expire (%v) in less time than the interval for checking expiration (%v)", expTime.Time, 2*client.server.config.APIs.COAP.OwnerCacheExpiration)
		}
		// set expiration time before token expiration to allow sign off device.
		validUntil = expTime.Add(-2 * client.server.config.APIs.COAP.OwnerCacheExpiration)
	}

	return deviceID, validUntil, nil
}

const errFmtSignIn = "cannot handle sign in: %w"

// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.session.swagger.json
func signInPostHandler(req *mux.Message, client *session, signIn CoapSignInReq) (*pool.Message, error) {
	if err := signIn.checkOAuthRequest(); err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, errFmtSignIn, err)
	}

	deviceID, validUntil, err := getSignInDataFromClaims(req.Context(), client, signIn)
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, errFmtSignIn, err)
	}
	setDeviceIDToTracerSpan(req.Context(), deviceID)

	upd := client.updateAuthorizationContext(deviceID, signIn.UserID, signIn.AccessToken, validUntil)

	ctx := kitNetGrpc.CtxWithToken(kitNetGrpc.CtxWithIncomingToken(req.Context(), signIn.AccessToken), signIn.AccessToken)
	valid, err := subscribeAndValidateDeviceAccess(ctx, client, signIn.UserID, deviceID, upd != updateTypeNone)
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, errFmtSignIn, err)
	}
	if !valid {
		return nil, statusErrorf(coapCodes.Unauthorized, errFmtSignIn, fmt.Errorf("access to device('%s') denied", deviceID))
	}

	expiresIn := validUntilToExpiresIn(validUntil)
	accept, out, err := getSignInContent(expiresIn, req.Options())
	if err != nil {
		return nil, statusErrorf(coapCodes.ServiceUnavailable, errFmtSignIn, err)
	}

	updateDeviceMetadataResp, err := client.updateBySignInData(ctx, upd, deviceID, signIn.UserID)
	if err != nil {
		return nil, statusErrorf(coapCodes.ServiceUnavailable, errFmtSignIn, err)
	}
	twinEnabled := client.resolveTwinEnabled(ctx, updateDeviceMetadataResp)

	setExpirationClientCache(client.server.expirationClientCache, deviceID, client, validUntil)
	client.exchangeCache.Clear()
	client.refreshCache.Clear()

	x := struct {
		ctx         context.Context
		client      *session
		deviceID    string
		twinEnabled bool
	}{
		ctx:         ctx,
		client:      client,
		deviceID:    deviceID,
		twinEnabled: twinEnabled,
	}
	if err := client.server.taskQueue.Submit(func() {
		_, err := x.client.replaceDeviceObserverWithDeviceTwin(x.ctx, x.twinEnabled, false)
		if err != nil {
			x.client.Close()
			x.client.Errorf("%w", signInError(fmt.Errorf("failed to register resource observations for device %v: %w", x.deviceID, err)))
		}
	}); err != nil {
		return nil, statusErrorf(coapCodes.ServiceUnavailable, errFmtSignIn, fmt.Errorf("failed to register device observer: %w", err))
	}

	return client.createResponse(coapCodes.Changed, req.Token(), accept, out), nil
}

func updateDeviceMetadata(req *mux.Message, client *session) error {
	oldAuthCtx := client.CleanUp(true)
	if oldAuthCtx.GetDeviceID() != "" {
		ctx := kitNetGrpc.CtxWithToken(req.Context(), oldAuthCtx.GetAccessToken())
		client.server.expirationClientCache.Delete(oldAuthCtx.GetDeviceID())

		_, err := client.server.raClient.UpdateDeviceMetadata(ctx, &commands.UpdateDeviceMetadataRequest{
			DeviceId: oldAuthCtx.GetDeviceID(),
			Update: &commands.UpdateDeviceMetadataRequest_Connection{
				Connection: &commands.Connection{
					Status: commands.Connection_OFFLINE,
				},
			},
			CommandMetadata: &commands.CommandMetadata{
				Sequence:     client.coapConn.Sequence(),
				ConnectionId: client.RemoteAddr().String(),
			},
		})
		if err != nil {
			// Device will be still reported as online and it can fix his state by next calls online, offline commands.
			return fmt.Errorf("cannot update cloud device status: %w", err)
		}
	}
	return nil
}

func getSignOutDataFromClaims(ctx context.Context, client *session, signOut CoapSignInReq) (string, error) {
	jwtClaims, err := client.ValidateToken(ctx, signOut.AccessToken)
	if err != nil {
		return "", err
	}

	if err := jwtClaims.ValidateOwnerClaim(client.server.config.APIs.COAP.Authorization.OwnerClaim, signOut.UserID); err != nil {
		return "", err
	}

	return client.server.VerifyAndResolveDeviceID(client.tlsDeviceID, signOut.DeviceID, jwtClaims)
}

const errFmtSignOut = "cannot handle sign out: %w"

func signOutPostHandler(req *mux.Message, client *session, signOut CoapSignInReq) (*pool.Message, error) {
	if signOut.DeviceID == "" || signOut.UserID == "" || signOut.AccessToken == "" {
		authCurrentCtx, err := client.GetAuthorizationContext()
		if err != nil {
			return nil, statusErrorf(coapCodes.BadRequest, errFmtSignOut, err)
		}
		signOut = signOut.updateOAUthRequestIfEmpty(authCurrentCtx.DeviceID, authCurrentCtx.UserID, authCurrentCtx.AccessToken)
	}

	if err := signOut.checkOAuthRequest(); err != nil {
		return nil, statusErrorf(coapCodes.BadRequest, errFmtSignOut, err)
	}

	deviceID, err := getSignOutDataFromClaims(req.Context(), client, signOut)
	if err != nil {
		return nil, statusErrorf(coapCodes.Unauthorized, errFmtSignOut, err)
	}
	setDeviceIDToTracerSpan(req.Context(), deviceID)

	if err := updateDeviceMetadata(req, client); err != nil {
		return nil, statusErrorf(coapconv.GrpcErr2CoapCode(err, coapconv.Update), errFmtSignOut, err)
	}

	return client.createResponse(coapCodes.Changed, req.Token(), message.AppOcfCbor, []byte{0xA0}), nil // empty object
}

// Sign-in
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.session.swagger.json
func signInHandler(req *mux.Message, client *session) (*pool.Message, error) {
	switch req.Code() {
	case coapCodes.POST:
		var signIn CoapSignInReq
		err := cbor.ReadFrom(req.Body(), &signIn)
		if err != nil {
			return nil, statusErrorf(coapCodes.BadRequest, errFmtSignIn, fmt.Errorf("cannot decode body: %w", err))
		}
		switch signIn.Login {
		case true:
			return signInPostHandler(req, client, signIn)
		default:
			return signOutPostHandler(req, client, signIn)
		}
	default:
		return nil, statusErrorf(coapCodes.NotFound, "unsupported method %v", req.Code())
	}
}

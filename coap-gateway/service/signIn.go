package service

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	pbAS "github.com/go-ocf/cloud/authorization/pb"
	"github.com/go-ocf/cloud/coap-gateway/coapconv"
	grpcClient "github.com/go-ocf/cloud/grpc-gateway/client"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	"github.com/go-ocf/go-coap/v2/message"
	coapCodes "github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/mux"
	"github.com/go-ocf/kit/codec/cbor"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/net/coap"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
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

// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.session.swagger.json
func signInPostHandler(s mux.ResponseWriter, req *mux.Message, client *Client, signIn CoapSignInReq) {
	resp, err := client.server.asClient.SignIn(req.Context, &pbAS.SignInRequest{
		DeviceId:    signIn.DeviceID,
		UserId:      signIn.UserID,
		AccessToken: signIn.AccessToken,
	})
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: %w", err), coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.POST), req.Token)
		return
	}

	coapResp := CoapSignInResp{
		ExpiresIn: resp.ExpiresIn,
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

	authCtx := authCtx{
		AuthorizationContext: pbCQRS.AuthorizationContext{
			DeviceId: signIn.DeviceID,
		},
		UserID:      signIn.UserID,
		AccessToken: signIn.AccessToken,
		Expire:      time.Now().Add(time.Second * time.Duration(resp.ExpiresIn)),
	}
	serviceToken, err := client.server.oauthMgr.GetToken(req.Context)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot get service token: %w", err), coapCodes.InternalServerError, req.Token)
		client.Close()
		return
	}
	req.Context = kitNetGrpc.CtxWithUserID(kitNetGrpc.CtxWithToken(req.Context, serviceToken.AccessToken), authCtx.UserID)
	err = client.UpdateCloudDeviceStatus(req.Context, signIn.DeviceID, authCtx.AuthorizationContext, true)
	if err != nil {
		// Events from resources of device will be comes but device is offline. To recover cloud state, client need to reconnect to cloud.
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: cannot update cloud device status: %w", err), coapCodes.InternalServerError, req.Token)
		client.Close()
		return
	}

	oldAuthCtx := client.replaceAuthorizationContext(authCtx)
	newDevice := false

	switch {
	case oldAuthCtx.GetDeviceId() == "":
		newDevice = true
	case oldAuthCtx.GetDeviceId() != signIn.DeviceID || oldAuthCtx.UserID != signIn.UserID:
		client.cancelResourceSubscriptions(true)
		client.cancelDeviceSubscriptions(true)
		newDevice = true
	}

	if newDevice {
		h := deviceSubscriptionHandlers{
			onResourceUpdatePending: func(ctx context.Context, val *pb.Event_ResourceUpdatePending) error {
				return client.updateResource(ctx, val)
			},
			onResourceRetrievePending: func(ctx context.Context, val *pb.Event_ResourceRetrievePending) error {
				return client.retrieveResource(ctx, val)
			},
			onClose: func() {
				log.Debugf("device %v subscription(ResourceUpdatePending, ResourceRetrievePending) was closed", signIn.DeviceID)
			},
		}
		h.onError = func(err error) {
			log.Errorf("device %v subscription(ResourceUpdatePending, ResourceRetrievePending) ends with error: %v", signIn.DeviceID, err)
			if !strings.Contains(err.Error(), "transport is closing") {
				client.Close()
				return
			}
			client.deviceSubscriptions.Delete(pendingDeviceSubscriptionToken)
			for {
				log.Debugf("reconnect device %v subscription(ResourceUpdatePending, ResourceRetrievePending)")
				var devSub atomic.Value
				_, loaded := client.deviceSubscriptions.LoadOrStore(pendingDeviceSubscriptionToken, &devSub)
				if loaded {
					return
				}
				sub, err := grpcClient.NewDeviceSubscription(req.Context, signIn.DeviceID, &h, &h, client.server.rdClient)
				if err != nil {
					client.logAndWriteErrorResponse(fmt.Errorf("cannot create device %v pending subscription: %w", signIn.DeviceID, err), coapCodes.InternalServerError, req.Token)
					client.Close()
					return
				}
				if err == nil {
					devSub.Store(sub)
					return
				}
				client.deviceSubscriptions.Delete(pendingDeviceSubscriptionToken)
				if !strings.Contains(err.Error(), "connection refused") {
					client.Close()
					log.Errorf("device %v subscription(ResourceUpdatePending, ResourceRetrievePending) cannot reconnect: %v", signIn.DeviceID, err)
					return
				}
				select {
				case <-client.Context().Done():
					client.Close()
					return
				case <-time.After(client.server.ReconnectInterval):
				}
			}
		}
		var devSub atomic.Value
		_, loaded := client.deviceSubscriptions.LoadOrStore(pendingDeviceSubscriptionToken, &devSub)
		if loaded {
			client.logAndWriteErrorResponse(fmt.Errorf("cannot store device %v pending subscription: device subscription with token %v already exist", signIn.DeviceID, pendingDeviceSubscriptionToken), coapCodes.InternalServerError, req.Token)
			client.Close()
			return
		}
		sub, err := grpcClient.NewDeviceSubscription(req.Context, signIn.DeviceID, &h, &h, client.server.rdClient)
		if err != nil {
			client.logAndWriteErrorResponse(fmt.Errorf("cannot create device %v pending subscription: %w", signIn.DeviceID, err), coapCodes.InternalServerError, req.Token)
			client.Close()
			return
		}
		devSub.Store(sub)
	}
	client.server.expirationClientCache.Set(signIn.DeviceID, client, time.Second*time.Duration(resp.ExpiresIn))
	client.sendResponse(coapCodes.Changed, req.Token, accept, out)
}

func signOutPostHandler(s mux.ResponseWriter, req *mux.Message, client *Client, signOut CoapSignInReq) {
	_, err := client.server.asClient.SignOut(req.Context, &pbAS.SignOutRequest{
		DeviceId:    signOut.DeviceID,
		UserId:      signOut.UserID,
		AccessToken: signOut.AccessToken,
	})
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign out: %w", err), coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.POST), req.Token)
		client.Close()
		return
	}

	client.cancelResourceSubscriptions(true)
	client.cancelDeviceSubscriptions(true)
	oldAuthCtx := client.replaceAuthorizationContext(authCtx{})
	if oldAuthCtx.DeviceId != "" {
		client.server.expirationClientCache.Delete(oldAuthCtx.DeviceId)
		serviceToken, err := client.server.oauthMgr.GetToken(req.Context)
		if err != nil {
			client.logAndWriteErrorResponse(fmt.Errorf("cannot get service token: %w", err), coapCodes.InternalServerError, req.Token)
			client.Close()
			return
		}
		req.Context = kitNetGrpc.CtxWithUserID(kitNetGrpc.CtxWithToken(req.Context, serviceToken.AccessToken), oldAuthCtx.UserID)
		err = client.UpdateCloudDeviceStatus(req.Context, oldAuthCtx.DeviceId, oldAuthCtx.AuthorizationContext, false)
		if err != nil {
			// Device will be still reported as online and it can fix his state by next calls online, offline commands.
			log.Errorf("DeviceId %v: cannot handle sign out: cannot update cloud device status: %v", oldAuthCtx.GetDeviceId(), err)
			return
		}
	}

	client.sendResponse(coapCodes.Changed, req.Token, message.AppOcfCbor, []byte{0xA0}) // empty object
}

// Sign-in
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.session.swagger.json
func signInHandler(s mux.ResponseWriter, req *mux.Message, client *Client) {
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
			signInPostHandler(s, req, client, signIn)
		default:
			signOutPostHandler(s, req, client, signIn)
		}
	default:
		client.logAndWriteErrorResponse(fmt.Errorf("Forbidden request from %v", client.remoteAddrString()), coapCodes.Forbidden, req.Token)
	}
}

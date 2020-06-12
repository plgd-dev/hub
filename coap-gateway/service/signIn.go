package service

import (
	"context"
	"fmt"

	pbAS "github.com/go-ocf/cloud/authorization/pb"
	"github.com/go-ocf/cloud/coap-gateway/coapconv"
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
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: %v", err), coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.POST), req.Token)
		return
	}

	coapResp := CoapSignInResp{
		ExpiresIn: resp.ExpiresIn,
	}

	accept := coap.GetAccept(req.Options)
	encode, err := coap.GetEncoder(accept)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: %v", err), coapCodes.InternalServerError, req.Token)
		return
	}
	out, err := encode(coapResp)
	if err != nil {
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: %v", err), coapCodes.InternalServerError, req.Token)
		return
	}

	authCtx := authCtx{
		AuthorizationContext: pbCQRS.AuthorizationContext{
			DeviceId: signIn.DeviceID,
		},
		UserID:      signIn.UserID,
		AccessToken: signIn.AccessToken,
	}
	req.Context, err = client.server.ctxWithServiceToken(req.Context)
	if err != nil {
		client.logAndWriteErrorResponse(err, coapCodes.InternalServerError, req.Token)
		client.Close()
		return
	}

	req.Context = kitNetGrpc.CtxWithUserID(req.Context, authCtx.UserID)
	err = client.UpdateCloudDeviceStatus(req.Context, signIn.DeviceID, authCtx.AuthorizationContext, true)
	if err != nil {
		// Events from resources of device will be comes but device is offline. To recover cloud state, client need to reconnect to cloud.
		client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: cannot update cloud device status: %v", err), coapCodes.InternalServerError, req.Token)
		client.Close()
		return
	}

	oldDeviceID := client.storeAuthorizationContext(authCtx)
	newDevice := false

	switch {
	case oldDeviceID == "":
		newDevice = true
	case oldDeviceID != signIn.DeviceID:
		err := client.server.projection.Unregister(oldDeviceID)
		client.server.clientContainerByDeviceID.Remove(oldDeviceID)
		if err != nil {
			client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: %v", err), coapCodes.InternalServerError, req.Token)
			client.Close()
			return
		}
		newDevice = true
	}

	if newDevice {
		client.server.clientContainerByDeviceID.Add(signIn.DeviceID, client)
		loaded, err := client.server.projection.Register(context.Background(), signIn.DeviceID)
		if err != nil {
			client.server.projection.Unregister(signIn.DeviceID)
			client.server.clientContainerByDeviceID.Remove(signIn.DeviceID)

			client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: %v", err), coapCodes.InternalServerError, req.Token)
			client.Close()
			return
		}
		if !loaded {
			models := client.server.projection.Models(signIn.DeviceID, "")
			if len(models) == 0 {
				log.Errorf("cannot load models for deviceID %v", signIn.DeviceID)
			} else {
				for _, r := range models {
					r.(*resourceCtx).TriggerSignIn(req.Context)
				}
			}
		}
	}
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

	authCtxOld := client.loadAuthorizationContext()
	if authCtxOld.DeviceId != "" {
		req.Context, err = client.server.ctxWithServiceToken(req.Context)
		if err != nil {
			client.logAndWriteErrorResponse(fmt.Errorf("cannot get service token: %v", err), coapCodes.InternalServerError, req.Token)
			client.Close()
			return
		}
		req.Context = kitNetGrpc.CtxWithUserID(req.Context, authCtxOld.UserID)
		err = client.UpdateCloudDeviceStatus(req.Context, authCtxOld.DeviceId, authCtxOld.AuthorizationContext, false)
		if err != nil {
			// Device will be still reported as online and it can fix his state by next calls online, offline commands.
			log.Errorf("DeviceId %v: cannot handle sign out: cannot update cloud device status: %v", authCtxOld.DeviceId, err)
			return
		}

		client.storeAuthorizationContext(authCtx{})
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
			client.logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: %v", err), coapCodes.BadRequest, req.Token)
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

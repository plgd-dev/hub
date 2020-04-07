package service

import (
	"context"
	"errors"
	"fmt"

	pbAS "github.com/go-ocf/ocf-cloud/authorization/pb"
	"github.com/go-ocf/ocf-cloud/coap-gateway/coapconv"
	gocoap "github.com/go-ocf/go-coap"
	coapCodes "github.com/go-ocf/go-coap/codes"
	"github.com/go-ocf/kit/codec/cbor"
	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/net/coap"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	pbCQRS "github.com/go-ocf/ocf-cloud/resource-aggregate/pb"
	"google.golang.org/grpc/status"
)

type CoapSignInReq struct {
	DeviceId    string `json:"di"`
	UserId      string `json:"uid"`
	AccessToken string `json:"accesstoken"`
	Login       bool   `json:"login"`
}

type CoapSignInResp struct {
	ExpiresIn int64 `json:"expiresin"`
}

func validateSignIn(req CoapSignInReq) error {
	if req.DeviceId == "" {
		return errors.New("cannot sign in to auth server: invalid deviceId")
	}
	if req.AccessToken == "" {
		return errors.New("cannot sign in to auth server: invalid accessToken")
	}
	if req.UserId == "" {
		return errors.New("cannot sign in to auth server: invalid userId")
	}
	return nil
}

// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.session.swagger.json
func signInPostHandler(s gocoap.ResponseWriter, req *gocoap.Request, client *Client, signIn CoapSignInReq) {
	err := validateSignIn(signIn)
	if err != nil {
		if err != nil {
			logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: %v", err), s, client, coapCodes.BadRequest)
			return
		}
	}

	resp, err := client.server.asClient.SignIn(kitNetGrpc.CtxWithToken(req.Ctx, signIn.AccessToken), &pbAS.SignInRequest{
		DeviceId: signIn.DeviceId,
		UserId:   signIn.UserId,
	})
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: %v", err), s, client, coapconv.GrpcCode2CoapCode(status.Convert(err).Code(), coapCodes.POST))
		return
	}

	coapResp := CoapSignInResp{
		ExpiresIn: resp.ExpiresIn,
	}

	accept := coap.GetAccept(req.Msg)
	encode, err := coap.GetEncoder(accept)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: %v", err), s, client, coapCodes.InternalServerError)
		return
	}
	out, err := encode(coapResp)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: %v", err), s, client, coapCodes.InternalServerError)
		return
	}

	authCtx := authCtx{
		AuthorizationContext: pbCQRS.AuthorizationContext{
			DeviceId: signIn.DeviceId,
			UserId:   signIn.UserId,
		},
		AccessToken: signIn.AccessToken,
	}

	err = client.UpdateCloudDeviceStatus(kitNetGrpc.CtxWithToken(req.Ctx, signIn.AccessToken), signIn.DeviceId, authCtx.AuthorizationContext, true)
	if err != nil {
		// Events from resources of device will be comes but device is offline. To recover cloud state, client need to reconnect to cloud.
		logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: cannot update cloud device status: %v", err), s, client, coapCodes.InternalServerError)
		client.Close()
		return
	}

	oldDeviceId := client.storeAuthorizationContext(authCtx)
	newDevice := false

	switch {
	case oldDeviceId == "":
		newDevice = true
	case oldDeviceId != signIn.DeviceId:
		err := client.server.projection.Unregister(oldDeviceId)
		client.server.clientContainerByDeviceId.Remove(oldDeviceId)
		if err != nil {
			logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: %v", err), s, client, coapCodes.InternalServerError)
			client.Close()
			return
		}
		newDevice = true
	}

	if newDevice {
		client.server.clientContainerByDeviceId.Add(signIn.DeviceId, client)
		loaded, err := client.server.projection.Register(context.Background(), signIn.DeviceId)
		if err != nil {
			client.server.projection.Unregister(signIn.DeviceId)
			client.server.clientContainerByDeviceId.Remove(signIn.DeviceId)

			logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: %v", err), s, client, coapCodes.InternalServerError)
			client.Close()
			return
		}
		if !loaded {
			models := client.server.projection.Models(signIn.DeviceId, "")
			if len(models) == 0 {
				log.Errorf("cannot load models for deviceId %v", signIn.DeviceId)
			} else {
				for _, r := range models {
					r.(*resourceCtx).TriggerSignIn()
				}
			}
		}
	}
	sendResponse(s, client, coapCodes.Changed, accept, out)
}

func signOutPostHandler(s gocoap.ResponseWriter, req *gocoap.Request, client *Client) {
	authCtxOld := client.loadAuthorizationContext()

	if authCtxOld.DeviceId != "" {
		err := client.UpdateCloudDeviceStatus(kitNetGrpc.CtxWithToken(req.Ctx, authCtxOld.AccessToken), authCtxOld.DeviceId, authCtxOld.AuthorizationContext, false)
		if err != nil {
			// Device will be still reported as online and it can fix his state by next calls online, offline commands.
			log.Errorf("DeviceId %v: cannot handle sign out: cannot update cloud device status: %v", authCtxOld.DeviceId, err)
			return
		}

		client.storeAuthorizationContext(authCtx{})
	}

	sendResponse(s, client, coapCodes.Changed, coap.GetAccept(req.Msg), []byte{0xA0}) // empty object
}

// Sign-in
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.session.swagger.json
func signInHandler(s gocoap.ResponseWriter, req *gocoap.Request, client *Client) {
	switch req.Msg.Code() {
	case coapCodes.POST:
		var signIn CoapSignInReq
		err := cbor.Decode(req.Msg.Payload(), &signIn)
		if err != nil {
			logAndWriteErrorResponse(fmt.Errorf("cannot handle sign in: %v", err), s, client, coapCodes.BadRequest)
			return
		}
		switch signIn.Login {
		case true:
			signInPostHandler(s, req, client, signIn)
		default:
			signOutPostHandler(s, req, client)
		}
	default:
		logAndWriteErrorResponse(fmt.Errorf("Forbidden request from %v", req.Client.RemoteAddr()), s, client, coapCodes.Forbidden)
	}
}

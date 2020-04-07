package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-ocf/kit/log"
	pbCQRS "github.com/go-ocf/ocf-cloud/resource-aggregate/pb"
	pbDD "github.com/go-ocf/ocf-cloud/resource-directory/pb/device-directory"
	"github.com/valyala/fasthttp"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
)

type GetDevicesResponse struct {
	Devices map[string]*Device `json:"devices"`
}

type State struct {
	Id          string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	IsOnline    bool   `protobuf:"varint,2,opt,name=is_online,json=isOnline,proto3" json:"is_online,omitempty"`
	IsConnected bool   `protobuf:"varint,3,opt,name=is_connected,json=isConnected,proto3" json:"is_connected,omitempty"`
}

type Device struct {
	Id       string         `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Resource *pbDD.Resource `protobuf:"bytes,2,opt,name=resource" json:"resource,omitempty"`
	State    *State         `protobuf:"bytes,3,opt,name=state" json:"state,omitempty"`
}

func (r *RequestHandler) listDevices(ctx *fasthttp.RequestCtx, token, sub string) {
	log.Debugf("RequestHandler.listDevices start")
	t := time.Now()
	defer func() {
		log.Debugf("RequestHandler.listDevices takes %v", time.Since(t))
	}()

	getDevicesClient, err := r.ddClient.GetDevices(kitNetGrpc.CtxWithToken(context.Background(), token), &pbDD.GetDevicesRequest{
		AuthorizationContext: &pbCQRS.AuthorizationContext{
			UserId: sub,
		},
	})

	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot list device directory: %v", err), http.StatusBadRequest, ctx)
		return
	}
	defer getDevicesClient.CloseSend()

	devices := make(map[string]*Device)
	for {
		device, err := getDevicesClient.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logAndWriteErrorResponse(fmt.Errorf("cannot list device directory: %v", err), http.StatusBadRequest, ctx)
			return
		}
		devices[device.Id] = &Device{
			Id:       device.Id,
			Resource: device.Resource,
			State: &State{
				Id:       device.Id,
				IsOnline: device.IsOnline,
			},
		}
	}
	if len(devices) == 0 {
		logAndWriteErrorResponse(fmt.Errorf("cannot list device directory: not found"), http.StatusNotFound, ctx)
		return
	}

	writeJson(GetDevicesResponse{
		Devices: devices,
	}, fasthttp.StatusOK, ctx)
}

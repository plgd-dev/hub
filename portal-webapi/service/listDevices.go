package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	pbGRPC "github.com/go-ocf/cloud/grpc-gateway/pb"
	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/sdk/schema"
	"github.com/valyala/fasthttp"
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
	Id       string        `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Resource schema.Device `protobuf:"bytes,2,opt,name=resource" json:"resource,omitempty"`
	State    *State        `protobuf:"bytes,3,opt,name=state" json:"state,omitempty"`
}

func (r *RequestHandler) listDevices(ctx *fasthttp.RequestCtx, token, sub string) {
	log.Debugf("RequestHandler.listDevices start")
	t := time.Now()
	defer func() {
		log.Debugf("RequestHandler.listDevices takes %v", time.Since(t))
	}()

	getDevicesClient, err := r.rdClient.GetDevices(kitNetGrpc.CtxWithToken(context.Background(), token), &pbGRPC.GetDevicesRequest{})

	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot list device directory: %w", err), http.StatusBadRequest, ctx)
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
			logAndWriteErrorResponse(fmt.Errorf("cannot list device directory: %w", err), http.StatusBadRequest, ctx)
			return
		}
		devices[device.Id] = &Device{
			Id:       device.Id,
			Resource: device.ToSchema(),
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

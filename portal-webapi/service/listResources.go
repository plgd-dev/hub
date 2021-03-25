package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/sdk/schema"
	"github.com/valyala/fasthttp"
)

type DeviceResources struct {
	Resources map[string]schema.ResourceLink `json:"resources"`
}

type GetResourceLinksResponse struct {
	Devices map[string]*DeviceResources `json:"devices"`
}

func (r *RequestHandler) listResources(ctx *fasthttp.RequestCtx, token, sub string) {
	log.Debugf("RequestHandler.listResources start")
	t := time.Now()
	defer func() {
		log.Debugf("RequestHandler.listResources takes %v", time.Since(t))
	}()

	getResourceLinksClient, err := r.rdClient.GetResourceLinks(kitNetGrpc.CtxWithToken(context.Background(), token), &pbGRPC.GetResourceLinksRequest{})

	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot list resource directory: %w", err), http.StatusBadRequest, ctx)
		return
	}
	defer getResourceLinksClient.CloseSend()

	response := GetResourceLinksResponse{
		Devices: make(map[string]*DeviceResources),
	}
	for {
		resLink, err := getResourceLinksClient.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logAndWriteErrorResponse(fmt.Errorf("cannot list device directory: %w", err), http.StatusBadRequest, ctx)
			return
		}

		device, ok := response.Devices[resLink.GetDeviceId()]
		if !ok {
			device = &DeviceResources{
				Resources: make(map[string]schema.ResourceLink),
			}
			response.Devices[resLink.GetDeviceId()] = device
		}
		device.Resources["/"+resLink.GetDeviceId()+resLink.GetHref()] = resLink.ToSchema()
	}

	writeJson(response, fasthttp.StatusOK, ctx)
}

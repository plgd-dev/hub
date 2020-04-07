package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/kit/log"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRD "github.com/go-ocf/cloud/resource-directory/pb/resource-directory"
	"github.com/valyala/fasthttp"
)

type DeviceResources struct {
	Resources map[string]*pbRA.Resource `json:"resources"`
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

	getResourceLinksClient, err := r.rdClient.GetResourceLinks(kitNetGrpc.CtxWithToken(context.Background(), token), &pbRD.GetResourceLinksRequest{
		AuthorizationContext: &pbCQRS.AuthorizationContext{
			UserId:      sub,
		},
	})

	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot list resource directory: %v", err), http.StatusBadRequest, ctx)
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
			logAndWriteErrorResponse(fmt.Errorf("cannot list device directory: %v", err), http.StatusBadRequest, ctx)
			return
		}

		device, ok := response.Devices[resLink.Resource.DeviceId]
		if !ok {
			device = &DeviceResources{
				Resources: make(map[string]*pbRA.Resource),
			}
			response.Devices[resLink.Resource.DeviceId] = device
		}
		device.Resources[resLink.Resource.Id] = resLink.Resource
	}

	writeJson(response, fasthttp.StatusOK, ctx)
}

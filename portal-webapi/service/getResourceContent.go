package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ugorji/go/codec"

	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/log"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/valyala/fasthttp"
)

func parseResourceID(v string) (string, string) {
	if len(v) > 0 && v[0] == '/' {
		v = v[1:]
	}
	vals := strings.SplitN(v, "/", 2)
	if len(vals) < 2 {
		return v, ""
	}
	return vals[0], "/" + vals[1]
}

func (r *RequestHandler) getResourceContent(ctx *fasthttp.RequestCtx, token, sub string) {
	log.Debugf("RequestHandler.listResourceDirectory start")
	t := time.Now()
	defer func() {
		log.Debugf("RequestHandler.listResourceDirectory takes %v", time.Since(t))
	}()
	var resourceId string
	var ok bool

	if resourceId, ok = ctx.UserValue("resourceId").(string); !ok {
		logAndWriteErrorResponse(fmt.Errorf("cannot retrieve resource content: resourceId from uri"), http.StatusBadRequest, ctx)
		return
	}

	deviceID, href := parseResourceID(resourceId)

	retrieveResourcesValuesClient, err := r.rdClient.RetrieveResourcesValues(kitNetGrpc.CtxWithToken(context.Background(), token), &pbGRPC.RetrieveResourcesValuesRequest{
		ResourceIdsFilter: []*pbGRPC.ResourceId{
			{
				DeviceId: deviceID,
				Href:     href,
			},
		},
	})

	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot retrieve resource content: %w", err), http.StatusBadRequest, ctx)
		return
	}
	defer retrieveResourcesValuesClient.CloseSend()

	var m interface{}

	for {
		resourceValue, err := retrieveResourcesValuesClient.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logAndWriteErrorResponse(fmt.Errorf("cannot retrieve resource content: %w", err), http.StatusBadRequest, ctx)
			return
		}
		if resourceValue.GetResourceId().GetDeviceId() == deviceID && resourceValue.GetResourceId().GetHref() == href && resourceValue.Content != nil {
			switch resourceValue.Content.ContentType {
			case message.AppCBOR.String(), message.AppOcfCbor.String():
				err := codec.NewDecoderBytes(resourceValue.Content.Data, new(codec.CborHandle)).Decode(&m)
				if err != nil {
					logAndWriteErrorResponse(fmt.Errorf("cannot retrieve resource content: %w", err), http.StatusInternalServerError, ctx)
					return
				}
			case message.AppJSON.String():
				err := codec.NewDecoderBytes(resourceValue.Content.Data, new(codec.JsonHandle)).Decode(&m)
				if err != nil {
					logAndWriteErrorResponse(fmt.Errorf("cannot retrieve resource content: %w", err), http.StatusInternalServerError, ctx)
					return
				}
			case message.TextPlain.String():
				m = string(resourceValue.Content.Data)
			default:
				logAndWriteErrorResponse(fmt.Errorf("cannot retrieve resource content: cannot convert content-type '%v' to json", resourceValue.Content.ContentType), http.StatusInternalServerError, ctx)
				return
			}
			break
		}
	}
	if m == nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot retrieve resource content: not found"), http.StatusNotFound, ctx)
		return
	}

	writeJson(m, fasthttp.StatusOK, ctx)
}

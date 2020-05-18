package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ugorji/go/codec"

	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRS "github.com/go-ocf/cloud/resource-directory/pb/resource-shadow"
	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/valyala/fasthttp"
)

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

	retrieveResourcesValuesClient, err := r.rsClient.RetrieveResourcesValues(kitNetGrpc.CtxWithToken(context.Background(), token), &pbRS.RetrieveResourcesValuesRequest{
		AuthorizationContext: &pbCQRS.AuthorizationContext{
			UserId: sub,
		},
		ResourceIdsFilter: []string{resourceId},
	})

	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot retrieve resource content: %v", err), http.StatusBadRequest, ctx)
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
			logAndWriteErrorResponse(fmt.Errorf("cannot retrieve resource content: %v", err), http.StatusBadRequest, ctx)
			return
		}
		if resourceValue.ResourceId == resourceId && resourceValue.Content != nil {
			switch resourceValue.Content.ContentType {
			case message.AppCBOR.String(), message.AppOcfCbor.String():
				err := codec.NewDecoderBytes(resourceValue.Content.Data, new(codec.CborHandle)).Decode(&m)
				if err != nil {
					logAndWriteErrorResponse(fmt.Errorf("cannot retrieve resource content: %v", err), http.StatusInternalServerError, ctx)
					return
				}
			case message.AppJSON.String():
				err := codec.NewDecoderBytes(resourceValue.Content.Data, new(codec.JsonHandle)).Decode(&m)
				if err != nil {
					logAndWriteErrorResponse(fmt.Errorf("cannot retrieve resource content: %v", err), http.StatusInternalServerError, ctx)
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

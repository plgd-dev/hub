package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gofrs/uuid"

	"github.com/ugorji/go/codec"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs"
	pbCQRS "github.com/plgd-dev/cloud/resource-aggregate/pb"
	pbRA "github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/log"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/valyala/fasthttp"
)

func (r *RequestHandler) updateResourceContent(ctx *fasthttp.RequestCtx, token, sub string) {
	log.Debugf("RequestHandler.updateResourceContent start")
	t := time.Now()
	defer func() {
		log.Debugf("RequestHandler.updateResourceContent takes %v", time.Since(t))
	}()
	var resourceId string
	var ok bool

	if resourceId, ok = ctx.UserValue("resourceId").(string); !ok {
		logAndWriteErrorResponse(fmt.Errorf("cannot update resource content: resourceId from uri"), http.StatusBadRequest, ctx)
		return
	}
	deviceID, href := parseResourceID(resourceId)
	resourceId = cqrs.MakeResourceId(deviceID, href)

	if len(ctx.Request.Body()) == 0 {
		logAndWriteErrorResponse(fmt.Errorf("cannot update resource content: body is empty"), http.StatusBadRequest, ctx)
		return
	}

	var m interface{}

	err := codec.NewDecoderBytes(ctx.Request.Body(), new(codec.JsonHandle)).Decode(&m)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot update resource content - decode json: %w", err), http.StatusBadRequest, ctx)
		return
	}

	bw := bytes.NewBuffer(make([]byte, 0, 1024))
	err = codec.NewEncoder(bw, new(codec.CborHandle)).Encode(m)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot update resource content - encode cbor: %w", err), http.StatusInternalServerError, ctx)
		return
	}

	correlationIdUUID, err := uuid.NewV4()
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot update resource content - generate uuid: %w", err), http.StatusBadRequest, ctx)
		return
	}

	correlationId := correlationIdUUID.String()

	response, err := r.raClient.UpdateResource(kitNetGrpc.CtxWithToken(context.Background(), token), &pbRA.UpdateResourceRequest{
		ResourceId: resourceId,
		Content: &pbRA.Content{
			CoapContentFormat: int32(message.AppOcfCbor),
			ContentType:       message.AppOcfCbor.String(),
			Data:              bw.Bytes(),
		},
		CommandMetadata: &pbCQRS.CommandMetadata{
			ConnectionId: ctx.RemoteAddr().String(),
			Sequence:     ctx.ConnRequestNum(),
		},
		CorrelationId: correlationId,
	})

	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot update resource content: %w", err), http.StatusBadRequest, ctx)
		return
	}

	writeJson(response, fasthttp.StatusAccepted, ctx)
}

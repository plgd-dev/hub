package service

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-ocf/kit/log"
	"github.com/valyala/fasthttp"
)

func (r *RequestHandler) offboardDevice(ctx *fasthttp.RequestCtx, token, sub string) {
	log.Debugf("RequestHandler.offboardDevice start")
	t := time.Now()
	defer func() {
		log.Debugf("RequestHandler.offboardDevice takes %v", time.Since(t))
	}()

	logAndWriteErrorResponse(fmt.Errorf("cannot offboard device: not supported"), http.StatusNotImplemented, ctx)
	return

	/*

		var deviceId string
		var ok bool

		if deviceId, ok = ctx.UserValue("deviceId").(string); !ok {
			logAndWriteErrorResponse(fmt.Errorf("cannot offboard device: deviceId from uri"), http.StatusBadRequest, ctx)
			return
		}

		httpRequestCtx := httputil.AcquireRequestCtx()
		defer httputil.ReleaseRequestCtx(httpRequestCtx)

		request := commands.UnpublishResourceRequest{
			AuthorizationContext: &kitCqrsProtobuf.AuthorizationContext{
				UserId:      sub,
				AccessToken: token,
			},
			//ResourceId:           resourceId,
		}
		var response commands.UnpublishResourceResponse
		httpCode, err := httpRequestCtx.PostProto(r.server.client, getContentResourceShadowURI(r.server), &request, &response)

		if err != nil {
			logAndWriteErrorResponse(fmt.Errorf("cannot offboard device: %w", err), http.StatusBadRequest, ctx)
			return
		}

		if httpCode != fasthttp.StatusOK {
			logAndWriteErrorResponse(fmt.Errorf("ccannot offboard device: StatusCode(%v)", httpCode), httpCode, ctx)
			return
		}

		writeJson(response, ctx)
	*/
}

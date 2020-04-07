package service

import (
	"bytes"
	"fmt"

	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/ocf-cloud/portal-webapi/uri"
	"github.com/ugorji/go/codec"
	"github.com/valyala/fasthttp"

	router "github.com/buaazp/fasthttprouter"
	pbRA "github.com/go-ocf/ocf-cloud/resource-aggregate/pb"
	pbDD "github.com/go-ocf/ocf-cloud/resource-directory/pb/device-directory"
	pbRD "github.com/go-ocf/ocf-cloud/resource-directory/pb/resource-directory"
	pbRS "github.com/go-ocf/ocf-cloud/resource-directory/pb/resource-shadow"
)

//RequestHandler for handling incoming request
type RequestHandler struct {
	config   Config
	server   *Server
	raClient pbRA.ResourceAggregateClient
	rsClient pbRS.ResourceShadowClient
	rdClient pbRD.ResourceDirectoryClient
	ddClient pbDD.DeviceDirectoryClient
}

//NewRequestHandler factory for new RequestHandler
func NewRequestHandler(server *Server, raClient pbRA.ResourceAggregateClient, rsClient pbRS.ResourceShadowClient, rdClient pbRD.ResourceDirectoryClient, ddClient pbDD.DeviceDirectoryClient) *RequestHandler {
	return &RequestHandler{
		server:   server,
		raClient: raClient,
		rsClient: rsClient,
		rdClient: rdClient,
		ddClient: ddClient,
	}
}

func logAndWriteErrorResponse(err error, statusCode int, ctx *fasthttp.RequestCtx) {
	log.Errorf("%v", err)
	ctx.Response.SetBody([]byte(err.Error()))
	ctx.SetStatusCode(statusCode)
}

func toJson(v interface{}) ([]byte, error) {
	bw := bytes.NewBuffer(make([]byte, 0, 1024))
	h := &codec.JsonHandle{}
	h.BasicHandle.Canonical = true
	err := codec.NewEncoder(bw, h).Encode(v)
	if err != nil {
		return nil, fmt.Errorf("cannot convert to json: %v", err)
	}
	return bw.Bytes(), nil
}

func writeJson(v interface{}, statusCode int, ctx *fasthttp.RequestCtx) {
	body, err := toJson(v)
	if err != nil {
		err = fmt.Errorf("cannot write body: %v", err)
		logAndWriteErrorResponse(err, fasthttp.StatusInternalServerError, ctx)
		return
	}
	ctx.Response.Header.SetContentType("application/json")
	ctx.Response.SetBody(body)
	ctx.SetStatusCode(statusCode)
}

func validateRequest(ctx *fasthttp.RequestCtx, cbk func(ctx *fasthttp.RequestCtx, token, sub string)) {
	token, sub, err := parseAuth(ctx)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("invalid request: %v", err), fasthttp.StatusUnauthorized, ctx)
		return
	}
	cbk(ctx, token, sub)
}

//NewHTTP return router handle HTTP request
func NewHTTP(requestHandler *RequestHandler) *router.Router {
	router := router.New()
	router.GET(uri.Devices, func(ctx *fasthttp.RequestCtx) {
		validateRequest(ctx, requestHandler.listDevices)
	})
	router.DELETE(uri.Devices+"/:deviceId", func(ctx *fasthttp.RequestCtx) {
		validateRequest(ctx, requestHandler.offboardDevice)
	})
	router.GET(uri.Resources, func(ctx *fasthttp.RequestCtx) {
		validateRequest(ctx, requestHandler.listResources)
	})
	router.GET(uri.Resources+"/:resourceId", func(ctx *fasthttp.RequestCtx) {
		validateRequest(ctx, requestHandler.getResourceContent)
	})
	router.PUT(uri.Resources+"/:resourceId", func(ctx *fasthttp.RequestCtx) {
		validateRequest(ctx, requestHandler.updateResourceContent)
	})
	router.GET(uri.Healthcheck, requestHandler.healthcheck)

	return router
}

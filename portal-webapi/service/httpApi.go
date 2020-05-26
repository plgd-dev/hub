package service

import (
	"bytes"
	"fmt"

	"github.com/go-ocf/cloud/portal-webapi/uri"
	"github.com/go-ocf/kit/log"
	"github.com/ugorji/go/codec"
	"github.com/valyala/fasthttp"

	router "github.com/buaazp/fasthttprouter"
	pbGRPC "github.com/go-ocf/cloud/grpc-gateway/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
)

//RequestHandler for handling incoming request
type RequestHandler struct {
	config   Config
	server   *Server
	rdClient pbGRPC.GrpcGatewayClient
	raClient pbRA.ResourceAggregateClient
}

//NewRequestHandler factory for new RequestHandler
func NewRequestHandler(server *Server, raClient pbRA.ResourceAggregateClient, rdClient pbGRPC.GrpcGatewayClient) *RequestHandler {
	return &RequestHandler{
		server:   server,
		rdClient: rdClient,
		raClient: raClient,
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

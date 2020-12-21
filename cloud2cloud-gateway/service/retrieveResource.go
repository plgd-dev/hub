package service

import (
	"context"
	"fmt"
	"net/http"

	pbGRPC "github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetHttp "github.com/plgd-dev/kit/net/http"
)

func (rh *RequestHandler) RetrieveResourceBase(ctx context.Context, w http.ResponseWriter, resourceID pbGRPC.ResourceId, encoder responseWriterEncoderFunc) (int, error) {
	allResources, err := rh.RetrieveResourcesValues(ctx, []*pbGRPC.ResourceId{&resourceID}, nil)
	if err != nil {
		return kitNetHttp.ErrToStatusWithDef(err, http.StatusForbidden), err
	}

	for _, v := range allResources {
		if v[0].Status != pbGRPC.Status_OK {
			return statusToHttpStatus(v[0].Status), fmt.Errorf("device returns unexpected code %v", v[0].Status)
		}

		err = encoder(w, v[0].Representation, http.StatusOK)
		if err != nil {
			return http.StatusBadRequest, err
		}
		return http.StatusOK, nil
	}
	return http.StatusNotFound, err
}

func (rh *RequestHandler) RetrieveResourceWithContentQuery(ctx context.Context, w http.ResponseWriter, routeVars map[string]string, contentQuery string, encoder responseWriterEncoderFunc) (int, error) {
	switch contentQuery {
	case ContentQueryBaseValue:
		deviceID := routeVars[deviceIDKey]
		href := routeVars[HrefKey]
		code, err := rh.RetrieveResourceBase(ctx, w, pbGRPC.ResourceId{
			DeviceId: deviceID, Href: href}, encoder)
		if err != nil {
			err = fmt.Errorf("cannot retrieve resource(deviceID: %v, Href: %v): %w", deviceID, href, err)
		}
		return code, err

	}
	return http.StatusBadRequest, fmt.Errorf("invalid content query parameter")
}

func (rh *RequestHandler) RetrieveResource(w http.ResponseWriter, r *http.Request) {
	statusCode, err := retrieveWithCallback(w, r, rh.RetrieveResourceWithContentQuery)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot retrieve resource: %w", err), statusCode, w)
	}
}

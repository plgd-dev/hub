package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	pbGRPC "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
)

func (rh *RequestHandler) RetrieveResourceBase(ctx context.Context, w http.ResponseWriter, resourceID *commands.ResourceId, encoder responseWriterEncoderFunc) (int, error) {
	allResources, err := rh.RetrieveResources(ctx, []*pbGRPC.ResourceIdFilter{
		{
			ResourceId: resourceID,
		},
	}, nil)
	if err != nil {
		return kitNetHttp.ErrToStatusWithDef(err, http.StatusForbidden), err
	}

	for _, v := range allResources {
		if v[0].Status != commands.Status_OK {
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
	if contentQuery == ContentQueryBaseValue {
		deviceID := routeVars[deviceIDKey]
		href := routeVars[hrefKey]
		code, err := rh.RetrieveResourceBase(ctx, w, &commands.ResourceId{
			DeviceId: deviceID, Href: href,
		}, encoder)
		if err != nil {
			err = fmt.Errorf("cannot retrieve resource(deviceID: %v, Href: %v): %w", deviceID, href, err)
		}
		return code, err
	}
	return http.StatusBadRequest, errors.New("invalid content query parameter")
}

func (rh *RequestHandler) RetrieveResource(w http.ResponseWriter, r *http.Request) {
	encoder, err := getResponseWriterEncoder(strings.Split(r.Header.Get("Accept"), ","))
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot retrieve resource: %w", err), http.StatusBadRequest, w)
		return
	}

	statusCode, err := rh.RetrieveResourceWithContentQuery(r.Context(), w, mux.Vars(r), getContentQueryValue(r.URL), encoder)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot retrieve resource: %w", err), statusCode, w)
	}
}

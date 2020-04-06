package service

import (
	"context"
	"fmt"
	"net/http"

	kitNetHttp "github.com/go-ocf/kit/net/http"
	cqrsRA "github.com/go-ocf/ocf-cloud/resource-aggregate/cqrs"
	pbCQRS "github.com/go-ocf/ocf-cloud/resource-aggregate/pb"
)

func (rh *RequestHandler) RetrieveResourceBase(ctx context.Context, w http.ResponseWriter, resourceID string, encoder responseWriterEncoderFunc) (int, error) {
	allResources, err := rh.RetrieveResourcesValues(ctx, []string{resourceID}, nil, pbCQRS.AuthorizationContext{})
	if err != nil {
		return kitNetHttp.ErrToStatusWithDef(err, http.StatusForbidden), fmt.Errorf("cannot retrieve resource(%v) [all]: %w", resourceID, err)
	}

	for _, v := range allResources {
		if v[0].Status != pbCQRS.Status_OK {
			return statusToHttpStatus(v[0].Status), fmt.Errorf("cannot retrieve resource(%v) [all]: device returns code %v", resourceID, v[0].Status)
		}

		err = encoder(w, v[0].Representation)
		if err != nil {
			return http.StatusBadRequest, fmt.Errorf("cannot retrieve resource(%v) [all]: %w", resourceID, err)
		}
		return http.StatusOK, nil
	}
	return http.StatusNotFound, fmt.Errorf("cannot retrieve resource(%v) [all]: %w", resourceID, err)
}

func (rh *RequestHandler) RetrieveResourceWithContentQuery(ctx context.Context, w http.ResponseWriter, routeVars map[string]string, contentQuery string, encoder responseWriterEncoderFunc) (int, error) {
	switch contentQuery {
	case ContentQueryBaseValue:
		deviceID := routeVars[deviceIDKey]
		resourceID := routeVars[resourceLinkHrefKey]
		code, err := rh.RetrieveResourceBase(ctx, w, cqrsRA.MakeResourceId(deviceID, resourceID), encoder)
		if err != nil {
			err = fmt.Errorf("cannot retrieve resource(deviceID: %v, Href: %v): %w", deviceID, resourceID, err)
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

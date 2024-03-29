package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/plgd-dev/device/v2/schema/device"
	pbGRPC "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Status string

const (
	Status_ONLINE  Status = "online"
	Status_OFFLINE Status = "offline"
)

func toStatus(isOnline bool) Status {
	if isOnline {
		return "online"
	}
	return "offline"
}

type responseWriterEncoderFunc func(w http.ResponseWriter, v interface{}, status int) error

type Device struct {
	Device device.Device `json:"device"`
	Status Status        `json:"status"`
}

func (rh *RequestHandler) GetDevices(ctx context.Context, deviceIdFilter []string) ([]Device, error) {
	getDevicesClient, err := rh.gwClient.GetDevices(ctx, &pbGRPC.GetDevicesRequest{
		DeviceIdFilter: deviceIdFilter,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot get devices: %w", err)
	}
	defer func() {
		if err := getDevicesClient.CloseSend(); err != nil {
			log.Errorf("cannot close get devices client: %w", err)
		}
	}()

	devices := make([]Device, 0, 32)
	for {
		grpcDevice, err := getDevicesClient.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("cannot get devices: %w", err)
		}

		var d device.Device
		if err = unmarshalContent(grpcDevice.GetData().GetContent(), &d); err != nil {
			d = grpcDevice.ToSchema()
		}
		if len(d.Interfaces) == 0 {
			d.Interfaces = grpcDevice.GetInterfaces()
		}
		if len(d.ResourceTypes) == 0 {
			d.ResourceTypes = grpcDevice.GetTypes()
		}
		devices = append(devices, Device{
			Device: d,
			Status: toStatus(grpcDevice.GetMetadata().GetConnection().IsOnline()),
		})
	}
	if len(devices) == 0 {
		return nil, status.Errorf(codes.NotFound, "cannot get devices: not found")
	}
	return devices, nil
}

func retrieveDevicesError(tag string, err error) error {
	return fmt.Errorf("cannot retrieve all devices[%s]: %w", tag, err)
}

func (rh *RequestHandler) retrieveDevicesBase(ctx context.Context, w http.ResponseWriter, encoder responseWriterEncoderFunc) (int, error) {
	devices, err := rh.GetDevices(ctx, nil)
	if err != nil {
		return kitNetHttp.ErrToStatusWithDef(err, http.StatusForbidden), retrieveDevicesError("base", err)
	}
	resourceLink, err := rh.GetResourceLinks(ctx, nil)
	if err != nil {
		return kitNetHttp.ErrToStatusWithDef(err, http.StatusForbidden), retrieveDevicesError("base", err)
	}

	resp := make([]RetrieveDeviceWithLinksResponse, 0, 32)
	for _, dev := range devices {
		links, ok := resourceLink[dev.Device.ID]
		if ok {
			resp = append(resp, RetrieveDeviceWithLinksResponse{
				Device: dev,
				Links:  links,
			})
		}
	}

	err = encoder(w, resp, http.StatusOK)
	if err != nil {
		return http.StatusBadRequest, retrieveDevicesError("base", err)
	}
	return http.StatusOK, nil
}

func (rh *RequestHandler) retrieveDevicesAll(ctx context.Context, w http.ResponseWriter, encoder responseWriterEncoderFunc) (int, error) {
	devices, err := rh.GetDevices(ctx, nil)
	if err != nil {
		return kitNetHttp.ErrToStatusWithDef(err, http.StatusForbidden), retrieveDevicesError("all", err)
	}
	reps, err := rh.RetrieveResources(ctx, nil, nil)
	if err != nil {
		return kitNetHttp.ErrToStatusWithDef(err, http.StatusForbidden), retrieveDevicesError("all", err)
	}

	resp := make([]RetrieveDeviceContentAllResponse, 0, 32)
	for _, dev := range devices {
		devReps, ok := reps[dev.Device.ID]
		if ok {
			resp = append(resp, RetrieveDeviceContentAllResponse{
				Device: dev,
				Links:  devReps,
			})
		}
	}

	err = encoder(w, resp, http.StatusOK)
	if err != nil {
		return http.StatusBadRequest, retrieveDevicesError("all", err)
	}
	return http.StatusOK, nil
}

func (rh *RequestHandler) retrieveDevicesWithContentQuery(ctx context.Context, w http.ResponseWriter, contentQuery string, encoder responseWriterEncoderFunc) (statusCode int, err error) {
	switch contentQuery {
	case ContentQueryAllValue:
		statusCode, err = rh.retrieveDevicesAll(ctx, w, encoder)
	case ContentQueryBaseValue:
		statusCode, err = rh.retrieveDevicesBase(ctx, w, encoder)
	default:
		return http.StatusBadRequest, fmt.Errorf("invalid value '%v' of '%v' query parameter", contentQuery, ContentQuery)
	}
	if err != nil {
		statusCode = kitNetHttp.ErrToStatusWithDef(err, statusCode)
		if statusCode == http.StatusNotFound {
			// return's empty array
			errEnc := encoder(w, []interface{}{}, http.StatusOK)
			if errEnc == nil {
				return http.StatusOK, nil
			}
		}
	}
	return statusCode, err
}

func (rh *RequestHandler) RetrieveDevices(w http.ResponseWriter, r *http.Request) {
	encoder, err := getResponseWriterEncoder(strings.Split(r.Header.Get("Accept"), ","))
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot retrieve all devices: %w", err), http.StatusBadRequest, w)
		return
	}

	statusCode, err := rh.retrieveDevicesWithContentQuery(r.Context(), w, getContentQueryValue(r.URL), encoder)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot retrieve all devices: %w", err), statusCode, w)
	}
}

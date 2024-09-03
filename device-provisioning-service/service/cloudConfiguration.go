package service

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/device/v2/schema/cloud"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store"
	"github.com/plgd-dev/hub/v2/internal/math"
	"github.com/plgd-dev/kit/v2/codec/cbor"
)

func (RequestHandle) ProcessCloudConfiguration(ctx context.Context, req *mux.Message, session *Session, linkedHubs []*LinkedHub, group *EnrollmentGroup) (*pool.Message, error) {
	switch req.Code() {
	case coapCodes.POST:
		msg, deviceID, cloudCfg, err := postProvisionCloudConfiguration(ctx, req, session, linkedHubs, group)
		coapGateways := make([]*pb.CloudStatus_Gateway, 0, len(cloudCfg.Endpoints))
		selectedGateway := -1
		for idx, c := range cloudCfg.Endpoints {
			coapGateways = append(coapGateways, &pb.CloudStatus_Gateway{
				Uri: c.URI,
				Id:  c.ID,
			})
			if c.ID == cloudCfg.CloudID && c.URI == cloudCfg.URL {
				selectedGateway = idx
			}
		}
		session.updateProvisioningRecord(&store.ProvisioningRecord{
			DeviceId: deviceID,
			Cloud: &pb.CloudStatus{
				Status: &pb.ProvisionStatus{
					Date:         time.Now().UnixNano(),
					CoapCode:     toCoapCode(msg),
					ErrorMessage: toErrorStr(err),
				},
				Gateways:        coapGateways,
				ProviderName:    cloudCfg.AuthorizationProvider,
				SelectedGateway: math.CastTo[int32](selectedGateway),
			},
		})
		return msg, err
	default:
		return nil, statusErrorf(coapCodes.Forbidden, "unsupported command(%v)", req.Code())
	}
}

type ProvisionCloudConfigurationRequest struct {
	DeviceID        string         `json:"di"`
	SelectedGateway cloud.Endpoint `json:"selectedGateway"`
}

func postProvisionCloudConfiguration(ctx context.Context, req *mux.Message, session *Session, linkedHubs []*LinkedHub, group *EnrollmentGroup) (resp *pool.Message, deviceID string, cloudCfg cloud.ConfigurationUpdateRequest, err error) {
	if req.Body() == nil {
		return nil, "", cloudCfg, statusErrorf(coapCodes.BadRequest, "unable to parse cloud configuration request from empty body")
	}
	var provisionCloudConfigurationRequest ProvisionCloudConfigurationRequest
	err = cbor.ReadFrom(req.Body(), &provisionCloudConfigurationRequest)
	if err != nil {
		return nil, provisionCloudConfigurationRequest.DeviceID, cloudCfg, statusErrorf(coapCodes.BadRequest, "unable to parse cloud configuration request from body: %w", err)
	}
	resp, cloudCfg, err = provisionCloudConfiguration(ctx, req, session, linkedHubs, group, provisionCloudConfigurationRequest)
	return resp, provisionCloudConfigurationRequest.DeviceID, cloudCfg, err
}

func findSelectedLinkedHub(selectedHub cloud.Endpoint, linkedHubs []*LinkedHub) *LinkedHub {
	if selectedHub.URI == "" || selectedHub.ID == "" {
		return nil
	}
	for _, l := range linkedHubs {
		if l.cfg.GetHubId() == selectedHub.ID {
			for _, c := range l.cfg.GetGateways() {
				uri, _ := pb.ValidateCoapGatewayURI(c)
				if uri == selectedHub.URI {
					return l
				}
			}
		}
	}
	return nil
}

func findDefaultLinkedHub(linkedHubs []*LinkedHub) *LinkedHub {
	if len(linkedHubs) == 0 {
		return nil
	}
	return linkedHubs[0]
}

func findLinkedHub(selectedHub cloud.Endpoint, linkedHubs []*LinkedHub) (*LinkedHub, cloud.Endpoint, error) {
	linkedHub := findSelectedLinkedHub(selectedHub, linkedHubs)
	if linkedHub != nil {
		return linkedHub, selectedHub, nil
	}
	linkedHub = findDefaultLinkedHub(linkedHubs)
	if linkedHub != nil {
		var uri string
		var errs *multierror.Error
		for _, c := range linkedHub.cfg.GetGateways() {
			var err error
			uri, err = pb.ValidateCoapGatewayURI(c)
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("invalid coap gateway uri %v: %w", c, err))
			} else {
				break
			}
		}
		if uri != "" {
			return linkedHub, cloud.Endpoint{
				URI: uri,
				ID:  linkedHub.cfg.GetId(),
			}, nil
		}
		return nil, cloud.Endpoint{}, statusErrorf(coapCodes.BadRequest, "cannot find valid coap gateway: %w", errs.ErrorOrNil())
	}
	return nil, cloud.Endpoint{}, statusErrorf(coapCodes.BadRequest, "cannot find linked hub")
}

type cloudEndpoints []cloud.Endpoint

func (e cloudEndpoints) Deduplicate() cloudEndpoints {
	if len(e) == 0 {
		return e
	}
	deduplicated := make(cloudEndpoints, 0, len(e))
	m := make(map[uint64]struct{}, len(e))
	for _, v := range e {
		if _, ok := m[toCRC64([]byte(v.ID+v.URI))]; ok {
			continue
		}
		m[toCRC64([]byte(v.ID+v.URI))] = struct{}{}
		deduplicated = append(deduplicated, v)
	}
	return deduplicated
}

func provisionCloudConfiguration(ctx context.Context, req *mux.Message, session *Session, linkedHubs []*LinkedHub, group *EnrollmentGroup, provisionCloudConfigurationRequest ProvisionCloudConfigurationRequest) (*pool.Message, cloud.ConfigurationUpdateRequest, error) {
	linkedHub, selectedGateway, err := findLinkedHub(provisionCloudConfigurationRequest.SelectedGateway, linkedHubs)
	if err != nil {
		return nil, cloud.ConfigurationUpdateRequest{}, err
	}
	requiredClaims := map[string]interface{}{
		linkedHub.cfg.GetAuthorization().GetOwnerClaim(): group.Owner,
	}
	urlValues := map[string]string{
		linkedHub.cfg.GetAuthorization().GetOwnerClaim(): group.Owner,
	}
	if linkedHub.cfg.GetAuthorization().GetDeviceIdClaim() != "" {
		urlValues[linkedHub.cfg.GetAuthorization().GetDeviceIdClaim()] = provisionCloudConfigurationRequest.DeviceID
		requiredClaims[linkedHub.cfg.GetAuthorization().GetDeviceIdClaim()] = provisionCloudConfigurationRequest.DeviceID
	}
	token, err := linkedHub.GetTokenFromOAuth(ctx, urlValues, requiredClaims)
	if err != nil {
		err = processErrClaimNotFound(err, session.getLogger(), linkedHub, group)
		return nil, cloud.ConfigurationUpdateRequest{}, statusErrorf(coapCodes.BadRequest, "cannot get token for cloud configuration response: %w", err)
	}

	endpoints := make(cloudEndpoints, 0, len(linkedHubs))
	for _, l := range linkedHubs {
		for _, c := range l.cfg.GetGateways() {
			uri, errC := pb.ValidateCoapGatewayURI(c)
			if errC != nil {
				continue
			}
			endpoints = append(endpoints, cloud.Endpoint{
				URI: uri,
				ID:  l.cfg.GetId(),
			})
		}
	}
	endpoints = endpoints.Deduplicate()

	resp := cloud.ConfigurationUpdateRequest{
		AuthorizationProvider: linkedHub.cfg.GetAuthorization().GetProvider().GetName(),
		URL:                   selectedGateway.URI,
		CloudID:               selectedGateway.ID,
		AuthorizationCode:     token.AccessToken,
		Endpoints:             endpoints,
	}

	msgType, data, err := encodeResponse(resp, req.Options())
	if err != nil {
		return nil, cloud.ConfigurationUpdateRequest{}, statusErrorf(coapCodes.BadRequest, "cannot encode cloud configuration response: %w", err)
	}

	return session.createResponse(coapCodes.Changed, req.Token(), msgType, data), resp, nil
}

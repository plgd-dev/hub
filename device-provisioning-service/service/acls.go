package service

import (
	"context"
	"time"

	"github.com/plgd-dev/device/v2/schema/acl"
	"github.com/plgd-dev/device/v2/schema/cloud"
	"github.com/plgd-dev/device/v2/schema/configuration"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/doxm"
	"github.com/plgd-dev/device/v2/schema/maintenance"
	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/plgd-dev/device/v2/schema/plgdtime"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/device/v2/schema/sdi"
	"github.com/plgd-dev/device/v2/schema/softwareupdate"
	"github.com/plgd-dev/device/v2/schema/sp"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store"
	"github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/pkg/strings"
)

func toCoapCode(msg *pool.Message) int32 {
	if msg == nil {
		return 0
	}
	return int32(msg.Code())
}

func toErrorStr(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

func (RequestHandle) ProcessACLs(_ context.Context, req *mux.Message, session *Session, linkedHubs []*LinkedHub, _ *EnrollmentGroup) (*pool.Message, error) {
	switch req.Code() {
	case coapCodes.GET:
		msg, acls, err := provisionACLs(req, session, linkedHubs)
		session.updateProvisioningRecord(&store.ProvisioningRecord{
			Acl: &pb.ACLStatus{
				Status: &pb.ProvisionStatus{
					Date:         time.Now().UnixNano(),
					CoapCode:     toCoapCode(msg),
					ErrorMessage: toErrorStr(err),
				},
				AccessControlList: acls,
			},
		})
		return msg, err
	default:
		return nil, statusErrorf(coapCodes.Forbidden, "unsupported command(%v)", req.Code())
	}
}

func provisionACLs(req *mux.Message, session *Session, linkedHubs []*LinkedHub) (*pool.Message, []*pb.AccessControl, error) {
	var resp acl.UpdateRequest
	allowedHubResources := []acl.Resource{
		// allow to update device's name from hub
		{
			Interfaces: []string{"*"},
			Href:       configuration.ResourceURI,
		},
		// allow to update device from hub
		{
			Interfaces: []string{"*"},
			Href:       softwareupdate.ResourceURI,
		},
		// allow to update maintenance from hub
		{
			Interfaces: []string{"*"},
			Href:       maintenance.ResourceURI,
		},
		// allow to update time from hub
		{
			Interfaces: []string{"*"},
			Href:       plgdtime.ResourceURI,
		},
	}
	// allow to access all resources ordinal resources from hub
	allowedHubResources = append(allowedHubResources, acl.AllResources...)

	allowedOwnerResources := []acl.Resource{
		// allow access to cloud configuration resource
		{
			Interfaces: []string{"*"},
			Href:       cloud.ResourceURI,
		},
		// allow access security profile resource
		{
			Interfaces: []string{"*"},
			Href:       sp.ResourceURI,
		},
		{
			Interfaces: []string{"*"},
			Href:       plgdtime.ResourceURI,
		},
	}
	allowedOwnerResources = append(allowedOwnerResources, allowedHubResources...)

	resp.AccessControlList = []acl.AccessControl{
		// owner acls - allow user access via client application
		{
			Permission: acl.AllPermissions,
			Subject: acl.Subject{
				Subject_Device: &acl.Subject_Device{
					DeviceID: events.OwnerToUUID(session.enrollmentGroup.Owner),
				},
			},
			Resources: allowedOwnerResources,
			Tag:       DPSTag,
		},
		// custom ACLs
		{
			Permission: acl.Permission_READ,
			Subject: acl.Subject{
				Subject_Connection: &acl.Subject_Connection{
					Type: acl.ConnectionType_ANON_CLEAR,
				},
			},
			Resources: []acl.Resource{
				{
					Href:       device.ResourceURI,
					Interfaces: []string{"*"},
				},
				{
					Href:       platform.ResourceURI,
					Interfaces: []string{"*"},
				},
				{
					Href:       resources.ResourceURI,
					Interfaces: []string{"*"},
				},
				{
					Href:       sdi.ResourceURI,
					Interfaces: []string{"*"},
				},
				{
					Href:       doxm.ResourceURI,
					Interfaces: []string{"*"},
				},
			},
			Tag: DPSTag,
		},
	}
	hubIDs := make([]string, 0, len(linkedHubs))
	for _, linkedHub := range linkedHubs {
		hubIDs = append(hubIDs, linkedHub.cfg.GetHubId())
	}
	hubIDs = strings.UniqueStable(hubIDs)
	for _, id := range hubIDs {
		// hub acls
		resp.AccessControlList = append(resp.AccessControlList, acl.AccessControl{
			Permission: acl.AllPermissions,
			Subject: acl.Subject{
				Subject_Device: &acl.Subject_Device{
					DeviceID: id,
				},
			},
			Resources: allowedHubResources,
			Tag:       DPSTag,
		})
	}

	msgType, data, err := encodeResponse(resp, req.Options())
	if err != nil {
		return nil, nil, statusErrorf(coapCodes.BadRequest, "cannot encode ACLs response: %w", err)
	}

	return session.createResponse(coapCodes.Content, req.Token(), msgType, data), pb.DeviceAccessControlListToPb(resp.AccessControlList), nil
}

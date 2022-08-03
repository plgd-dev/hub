package subscription

import (
	isEvents "github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	kitStrings "github.com/plgd-dev/kit/v2/strings"
)

const (
	FilterBitmaskRegistrations                 = FilterBitmaskDeviceRegistered | FilterBitmaskDeviceUnregistered
	FilterBitmaskDevices                       = FilterBitmaskDeviceMetadata | FilterBitmaskDeviceResourceLinks | FilterBitmaskDeviceDeviceResourcesResource
	FilterBitmaskDeviceMetadata                = FilterBitmaskDeviceMetadataUpdatePending | FilterBitmaskDeviceMetadataUpdated
	FilterBitmaskDeviceResourceLinks           = FilterBitmaskResourcesPublished | FilterBitmaskResourcesUnpublished
	FilterBitmaskDeviceDeviceResourcesResource = FilterBitmaskResourceChanged |
		FilterBitmaskResourceCreatePending | FilterBitmaskResourceCreated |
		FilterBitmaskResourceDeletePending | FilterBitmaskResourceDeleted |
		FilterBitmaskResourceRetrievePending | FilterBitmaskResourceRetrieved |
		FilterBitmaskResourceUpdatePending | FilterBitmaskResourceUpdated
)

type subject struct {
	bitmask FilterBitmask
	subject string
}

var bitmaskToSubjectsTemplate = []subject{
	{bitmask: FilterBitmaskMax, subject: isEvents.PlgdOwnersOwner + ".>"},

	{bitmask: FilterBitmaskRegistrations, subject: isEvents.PlgdOwnersOwnerRegistrations + ".>"},
	{bitmask: FilterBitmaskDeviceRegistered, subject: isEvents.ToSubject(isEvents.PlgdOwnersOwnerRegistrationsEvent, isEvents.WithEventType(isEvents.DevicesRegisteredEvent))},
	{bitmask: FilterBitmaskDeviceUnregistered, subject: isEvents.ToSubject(isEvents.PlgdOwnersOwnerRegistrationsEvent, isEvents.WithEventType(isEvents.DevicesUnregisteredEvent))},

	{bitmask: FilterBitmaskDevices, subject: utils.PlgdOwnersOwnerDevices + ".>"},

	{bitmask: FilterBitmaskDeviceMetadata, subject: utils.PlgdOwnersOwnerDevicesDeviceMetadata + ".>"},
	{bitmask: FilterBitmaskDeviceMetadataUpdatePending, subject: isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceMetadataEvent, isEvents.WithEventType((&events.DeviceMetadataUpdatePending{}).EventType()))},
	{bitmask: FilterBitmaskDeviceMetadataUpdated, subject: isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceMetadataEvent, isEvents.WithEventType((&events.DeviceMetadataUpdated{}).EventType()))},

	{bitmask: FilterBitmaskDeviceResourceLinks, subject: utils.PlgdOwnersOwnerDevicesDeviceResourceLinks + ".>"},
	{bitmask: FilterBitmaskResourcesPublished, subject: isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourceLinksEvent, isEvents.WithEventType((&events.ResourceLinksPublished{}).EventType()))},
	{bitmask: FilterBitmaskResourcesUnpublished, subject: isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourceLinksEvent, isEvents.WithEventType((&events.ResourceLinksUnpublished{}).EventType()))},

	{bitmask: FilterBitmaskDeviceDeviceResourcesResource, subject: utils.PlgdOwnersOwnerDevicesDeviceResourcesResource + ".>"},
	{bitmask: FilterBitmaskResourceChanged, subject: isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent, isEvents.WithEventType((&events.ResourceChanged{}).EventType()))},
	{bitmask: FilterBitmaskResourceCreatePending, subject: isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent, isEvents.WithEventType((&events.ResourceCreatePending{}).EventType()))},
	{bitmask: FilterBitmaskResourceCreated, subject: isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent, isEvents.WithEventType((&events.ResourceCreated{}).EventType()))},
	{bitmask: FilterBitmaskResourceDeletePending, subject: isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent, isEvents.WithEventType((&events.ResourceDeletePending{}).EventType()))},
	{bitmask: FilterBitmaskResourceDeleted, subject: isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent, isEvents.WithEventType((&events.ResourceDeleted{}).EventType()))},
	{bitmask: FilterBitmaskResourceRetrievePending, subject: isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent, isEvents.WithEventType((&events.ResourceRetrievePending{}).EventType()))},
	{bitmask: FilterBitmaskResourceRetrieved, subject: isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent, isEvents.WithEventType((&events.ResourceRetrieved{}).EventType()))},
	{bitmask: FilterBitmaskResourceUpdatePending, subject: isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent, isEvents.WithEventType((&events.ResourceUpdatePending{}).EventType()))},
	{bitmask: FilterBitmaskResourceUpdated, subject: isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent, isEvents.WithEventType((&events.ResourceUpdated{}).EventType()))},
}

func ConvertToSubjects(owner string, filterDeviceIDs kitStrings.Set, filterResourceIDs kitStrings.Set, bitmask FilterBitmask) []string {
	var rawTemplates []string
	for _, s := range bitmaskToSubjectsTemplate {
		if s.bitmask&bitmask == s.bitmask {
			rawTemplates = append(rawTemplates, s.subject)
			bitmask &= ^(s.bitmask)
		}
	}

	intTemplates := make(map[string]bool)
	for _, rawTemplate := range rawTemplates {
		switch {
		case len(filterResourceIDs) > 0:
			for resID := range filterResourceIDs {
				intTemplates[isEvents.ToSubject(rawTemplate, isEvents.WithOwner(owner), utils.WithDeviceID("*"), utils.WithResourceId(resID))] = true
			}
		case len(filterDeviceIDs) > 0:
			for devID := range filterDeviceIDs {
				intTemplates[isEvents.ToSubject(rawTemplate, isEvents.WithOwner(owner), utils.WithDeviceID(devID), utils.WithResourceId("*"))] = true
			}
		default:
			intTemplates[isEvents.ToSubject(rawTemplate, isEvents.WithOwner(owner), utils.WithDeviceID("*"), utils.WithResourceId("*"))] = true
		}
	}

	if len(intTemplates) == 0 {
		return nil
	}
	templates := make([]string, 0, len(intTemplates))
	for template := range intTemplates {
		templates = append(templates, template)
	}
	return templates
}

package subscription

import (
	"github.com/google/uuid"
	isEvents "github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
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

func convertTemplateToSubjects(owner string, filters map[uuid.UUID]*commands.ResourceId, rawTemplate string, subjects map[string]bool) {
	if len(filters) == 0 {
		subjects[isEvents.ToSubject(rawTemplate, isEvents.WithOwner(owner), utils.WithDeviceID("*"), utils.WithHrefId("*"))] = true
		return
	}
	for _, v := range filters {
		deviceID := v.GetDeviceId()
		if deviceID == "" {
			deviceID = "*"
		}
		hrefID := v.GetHref()
		switch hrefID {
		case "":
			hrefID = "*"
		case "*":
		default:
			hrefID = utils.HrefToID(hrefID).String()
		}
		subjects[isEvents.ToSubject(rawTemplate, isEvents.WithOwner(owner), utils.WithDeviceID(deviceID), utils.WithHrefId(hrefID))] = true
	}
}

func ConvertToSubjects(owner string, filters map[uuid.UUID]*commands.ResourceId, bitmask FilterBitmask) []string {
	var rawTemplates []string
	for _, s := range bitmaskToSubjectsTemplate {
		if s.bitmask&bitmask == s.bitmask {
			rawTemplates = append(rawTemplates, s.subject)
			bitmask &= ^(s.bitmask)
		}
	}

	subjects := make(map[string]bool)
	for _, rawTemplate := range rawTemplates {
		convertTemplateToSubjects(owner, filters, rawTemplate, subjects)
	}

	if len(subjects) == 0 {
		return nil
	}
	templates := make([]string, 0, len(subjects))
	for template := range subjects {
		templates = append(templates, template)
	}
	return templates
}

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
	bitmask  FilterBitmask
	subjects []string
}

// resourceEventSubjects returns subjects for resource events
//
// If the Publisher submits evets with leading resource type then if a resource type is matched then the subjects will have "leadResourceType.$resourceType" suffix,
// othewise the subjects will not have the suffix; NATS does not support "a zero or more" wildcard, so we need to have two subjects - one with the suffix and one without
func resourceEventSubjects(eventType string) []string {
	subject := isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent, isEvents.WithEventType(eventType))
	return []string{subject, subject + "." + utils.LeadResourcePrefix + ".>"}
}

var bitmaskToSubjectsTemplate = []subject{
	{bitmask: FilterBitmaskMax, subjects: []string{isEvents.PlgdOwnersOwner + ".>"}},

	{bitmask: FilterBitmaskRegistrations, subjects: []string{isEvents.PlgdOwnersOwnerRegistrations + ".>"}},
	{bitmask: FilterBitmaskDeviceRegistered, subjects: []string{isEvents.ToSubject(isEvents.PlgdOwnersOwnerRegistrationsEvent, isEvents.WithEventType(isEvents.DevicesRegisteredEvent))}},
	{bitmask: FilterBitmaskDeviceUnregistered, subjects: []string{isEvents.ToSubject(isEvents.PlgdOwnersOwnerRegistrationsEvent, isEvents.WithEventType(isEvents.DevicesUnregisteredEvent))}},

	{bitmask: FilterBitmaskDevices, subjects: []string{utils.PlgdOwnersOwnerDevices + ".>"}},

	{bitmask: FilterBitmaskDeviceMetadata, subjects: []string{utils.PlgdOwnersOwnerDevicesDeviceMetadata + ".>"}},
	{bitmask: FilterBitmaskDeviceMetadataUpdatePending, subjects: []string{isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceMetadataEvent, isEvents.WithEventType((&events.DeviceMetadataUpdatePending{}).EventType()))}},
	{bitmask: FilterBitmaskDeviceMetadataUpdated, subjects: []string{isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceMetadataEvent, isEvents.WithEventType((&events.DeviceMetadataUpdated{}).EventType()))}},

	{bitmask: FilterBitmaskDeviceResourceLinks, subjects: []string{utils.PlgdOwnersOwnerDevicesDeviceResourceLinks + ".>"}},
	{bitmask: FilterBitmaskResourcesPublished, subjects: []string{isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourceLinksEvent, isEvents.WithEventType((&events.ResourceLinksPublished{}).EventType()))}},
	{bitmask: FilterBitmaskResourcesUnpublished, subjects: []string{isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourceLinksEvent, isEvents.WithEventType((&events.ResourceLinksUnpublished{}).EventType()))}},

	{bitmask: FilterBitmaskDeviceDeviceResourcesResource, subjects: []string{utils.PlgdOwnersOwnerDevicesDeviceResourcesResource + ".>"}},

	{bitmask: FilterBitmaskResourceChanged, subjects: resourceEventSubjects((&events.ResourceChanged{}).EventType())},
	{bitmask: FilterBitmaskResourceCreatePending, subjects: resourceEventSubjects((&events.ResourceCreatePending{}).EventType())},
	{bitmask: FilterBitmaskResourceCreated, subjects: resourceEventSubjects((&events.ResourceCreated{}).EventType())},
	{bitmask: FilterBitmaskResourceDeletePending, subjects: resourceEventSubjects((&events.ResourceDeletePending{}).EventType())},
	{bitmask: FilterBitmaskResourceDeleted, subjects: resourceEventSubjects((&events.ResourceDeleted{}).EventType())},
	{bitmask: FilterBitmaskResourceRetrievePending, subjects: resourceEventSubjects((&events.ResourceRetrievePending{}).EventType())},
	{bitmask: FilterBitmaskResourceRetrieved, subjects: resourceEventSubjects((&events.ResourceRetrieved{}).EventType())},
	{bitmask: FilterBitmaskResourceUpdatePending, subjects: resourceEventSubjects((&events.ResourceUpdatePending{}).EventType())},
	{bitmask: FilterBitmaskResourceUpdated, subjects: resourceEventSubjects((&events.ResourceUpdated{}).EventType())},
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
		hrefID := utils.GetSubjectHrefID(v.GetHref())
		subjects[isEvents.ToSubject(rawTemplate, isEvents.WithOwner(owner), utils.WithDeviceID(deviceID), utils.WithHrefId(hrefID))] = true
	}
}

func ConvertToSubjects(owner string, filters map[uuid.UUID]*commands.ResourceId, bitmask FilterBitmask) []string {
	var rawTemplates []string
	for _, s := range bitmaskToSubjectsTemplate {
		if s.bitmask&bitmask == s.bitmask {
			rawTemplates = append(rawTemplates, s.subjects...)
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

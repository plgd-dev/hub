package subscription

import (
	isEvents "github.com/plgd-dev/hub/v2/identity-store/events"
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
}

// for resource events the format of the subject varies depending on whether the leadResourceType filter is set or not
type resourceSubject struct {
	bitmask     FilterBitmask
	getSubjects func(leadRTFilter []string) []string
}

func subjectsForLeadResourceType(template string, leadRTFilter []string) []string {
	subjects := make([]string, 0, len(leadRTFilter))
	for _, leadRT := range leadRTFilter {
		subjects = append(subjects, isEvents.ToSubject(template, utils.WithLeadResourceType(leadRT)))
	}
	return subjects
}

func getSubjectsForEventType(eventType string) func(leadRTFilter []string) []string {
	return func(leadRTFilter []string) []string {
		if len(leadRTFilter) == 0 {
			// If the Publisher submits evets with leading resource type then if a resource type is matched then the subjects will have "leadResourceType.$resourceType" suffix,
			// othewise the subjects will not have the suffix; NATS does not support "a zero or more" wildcard, so we need to have two subjects - one with the suffix and one without
			subject := isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent, isEvents.WithEventType(eventType))
			isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEvent, isEvents.WithEventType(eventType))
			return []string{subject, subject + "." + utils.LeadResourcePrefix + ".>"}
		}
		template := isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEventLeadResourceType, isEvents.WithEventType(eventType))
		return subjectsForLeadResourceType(template, leadRTFilter)
	}
}

var bitmaskToResourceSubjectsTemplate = []resourceSubject{
	{bitmask: FilterBitmaskDeviceDeviceResourcesResource, getSubjects: func(leadRTFilter []string) []string {
		if len(leadRTFilter) == 0 {
			return []string{utils.PlgdOwnersOwnerDevicesDeviceResourcesResource + ".>"}
		}
		template := isEvents.ToSubject(utils.PlgdOwnersOwnerDevicesDeviceResourcesResourceEventLeadResourceType, isEvents.WithEventType("*"))
		return subjectsForLeadResourceType(template, leadRTFilter)
	}},
	{bitmask: FilterBitmaskResourceChanged, getSubjects: getSubjectsForEventType((&events.ResourceChanged{}).EventType())},
	{bitmask: FilterBitmaskResourceCreatePending, getSubjects: getSubjectsForEventType((&events.ResourceCreatePending{}).EventType())},
	{bitmask: FilterBitmaskResourceCreated, getSubjects: getSubjectsForEventType((&events.ResourceCreated{}).EventType())},
	{bitmask: FilterBitmaskResourceDeletePending, getSubjects: getSubjectsForEventType((&events.ResourceDeletePending{}).EventType())},
	{bitmask: FilterBitmaskResourceDeleted, getSubjects: getSubjectsForEventType((&events.ResourceDeleted{}).EventType())},
	{bitmask: FilterBitmaskResourceRetrievePending, getSubjects: getSubjectsForEventType((&events.ResourceRetrievePending{}).EventType())},
	{bitmask: FilterBitmaskResourceRetrieved, getSubjects: getSubjectsForEventType((&events.ResourceRetrieved{}).EventType())},
	{bitmask: FilterBitmaskResourceUpdatePending, getSubjects: getSubjectsForEventType((&events.ResourceUpdatePending{}).EventType())},
	{bitmask: FilterBitmaskResourceUpdated, getSubjects: getSubjectsForEventType((&events.ResourceUpdated{}).EventType())},
}

func convertTemplateToSubjects(owner string, sf SubjectFilters, rawTemplate string, subjects map[string]bool) {
	if len(sf.resourceFilters) == 0 {
		subjects[isEvents.ToSubject(rawTemplate, isEvents.WithOwner(owner), utils.WithDeviceID("*"), utils.WithHrefId("*"))] = true
		return
	}
	for _, v := range sf.resourceFilters {
		deviceID := v.GetDeviceId()
		if deviceID == "" {
			deviceID = "*"
		}
		hrefID := utils.GetSubjectHrefID(v.GetHref())
		subjects[isEvents.ToSubject(rawTemplate, isEvents.WithOwner(owner), utils.WithDeviceID(deviceID), utils.WithHrefId(hrefID))] = true
	}
}

func ConvertToSubjects(owner string, filters SubjectFilters, bitmask FilterBitmask) []string {
	var rawTemplates []string
	for _, s := range bitmaskToSubjectsTemplate {
		if s.bitmask&bitmask == s.bitmask {
			rawTemplates = append(rawTemplates, s.subject)
			bitmask &= ^(s.bitmask)
		}
	}
	for _, s := range bitmaskToResourceSubjectsTemplate {
		if s.bitmask&bitmask == s.bitmask {
			rawTemplates = append(rawTemplates, s.getSubjects(filters.leadResourceTypeFilter)...)
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

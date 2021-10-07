package events

const Registrations = "registrations"
const PlgdOwnersOwnerRegistrations = PlgdOwnersOwner + "." + Registrations
const PlgdOwnersOwnerRegistrationsEvent = PlgdOwnersOwnerRegistrations + ".{" + EventTypeKey + "}"

const DevicesRegisteredEvent = "devicesregistered"
const DevicesUnregisteredEvent = "devicesunregistered"

func GetRegistrationSubject(owner string) string {
	return ToSubject(PlgdOwnersOwnerRegistrations+".>", WithOwner(owner))
}

func GetDevicesRegisteredSubject(owner string) string {
	return ToSubject(PlgdOwnersOwnerRegistrationsEvent, WithOwner(owner), WithEventType(DevicesRegisteredEvent))
}

func GetDevicesUnregisteredSubject(owner string) string {
	return ToSubject(PlgdOwnersOwnerRegistrationsEvent, WithOwner(owner), WithEventType(DevicesUnregisteredEvent))
}

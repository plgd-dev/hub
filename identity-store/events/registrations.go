package events

const (
	Registrations                     = "registrations"
	PlgdOwnersOwnerRegistrations      = PlgdOwnersOwner + "." + Registrations
	PlgdOwnersOwnerRegistrationsEvent = PlgdOwnersOwnerRegistrations + ".{" + EventTypeKey + "}"
)

const (
	DevicesRegisteredEvent   = "devicesregistered"
	DevicesUnregisteredEvent = "devicesunregistered"
)

func GetRegistrationSubject(owner string) string {
	return ToSubject(PlgdOwnersOwnerRegistrations+".>", WithOwner(owner))
}

func GetDevicesRegisteredSubject(owner string) string {
	return ToSubject(PlgdOwnersOwnerRegistrationsEvent, WithOwner(owner), WithEventType(DevicesRegisteredEvent))
}

func GetDevicesUnregisteredSubject(owner string) string {
	return ToSubject(PlgdOwnersOwnerRegistrationsEvent, WithOwner(owner), WithEventType(DevicesUnregisteredEvent))
}

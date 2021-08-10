package events

func GetOwnerSubject(owner string) string {
	return "owners." + owner + ".>"
}

func GetDevicesRegisteredSubject(owner string) string {
	return "owners." + owner + ".devicesregistered"
}

func GetDevicesUnregisteredSubject(owner string) string {
	return "owners." + owner + ".devicesunregistered"
}

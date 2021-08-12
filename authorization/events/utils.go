package events

import "github.com/google/uuid"

func OwnerToUUID(owner string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(owner)).String()
}

func GetOwnerSubject(owner string) string {
	return "owners." + OwnerToUUID(owner) + ".>"
}

func GetDevicesRegisteredSubject(owner string) string {
	return "owners." + OwnerToUUID(owner) + ".devicesregistered"
}

func GetDevicesUnregisteredSubject(owner string) string {
	return "owners." + OwnerToUUID(owner) + ".devicesunregistered"
}

package events

import "github.com/google/uuid"

func OwnerToUUID(owner string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(owner)).String()
}

const OWNER_PREFIX = "owners"

func GetOwnerSubject(owner string) string {
	return OWNER_PREFIX + "." + OwnerToUUID(owner) + ".>"
}

func GetDevicesRegisteredSubject(owner string) string {
	return OWNER_PREFIX + "." + OwnerToUUID(owner) + ".devicesregistered"
}

func GetDevicesUnregisteredSubject(owner string) string {
	return OWNER_PREFIX + "." + OwnerToUUID(owner) + ".devicesunregistered"
}

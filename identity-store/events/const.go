package events

const OwnerIdKey = "ownerId"
const EventTypeKey = "eventType"

const Plgd = "plgd"
const PlgdOwners = Plgd + ".owners"
const PlgdOwnersOwner = PlgdOwners + ".{" + OwnerIdKey + "}"

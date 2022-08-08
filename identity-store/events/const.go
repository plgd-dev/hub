package events

const (
	OwnerIdKey   = "ownerId"
	EventTypeKey = "eventType"
)

const (
	Plgd            = "plgd"
	PlgdOwners      = Plgd + ".owners"
	PlgdOwnersOwner = PlgdOwners + ".{" + OwnerIdKey + "}"
)

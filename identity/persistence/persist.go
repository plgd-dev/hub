package persistence

// AuthorizedDevice comprises device's authorization details.
type AuthorizedDevice struct {
	DeviceID string `db:"device_id"`
	Owner    string `db:"owner"`
}

type Iterator interface {
	Err() error
	Next(v *AuthorizedDevice) bool
	Close()
}

type PersistenceTx interface {
	Retrieve(deviceID, owner string) (_ *AuthorizedDevice, ok bool, err error)
	RetrieveByDevice(deviceID string) (_ *AuthorizedDevice, ok bool, err error)
	RetrieveByOwner(owner string) Iterator
	RetrieveAll() Iterator
	Persist(d *AuthorizedDevice) error
	Delete(deviceID, owner string) error
	Close()
}

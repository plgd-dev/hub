package persistence

import (
	"time"
)

// AuthorizedDevice comprises device's authorization details.
type AuthorizedDevice struct {
	DeviceID     string    `db:"device_id"`
	UserID       string    `db:"user_id"`
	AccessToken  string    `db:"access_token"`
	RefreshToken string    `db:"refresh_token"`
	Expiry       time.Time `db:"expiry"`
}

type Iterator interface {
	Err() error
	Next(v *AuthorizedDevice) bool
	Close()
}

type PersistenceTx interface {
	Retrieve(deviceID, userID string) (_ *AuthorizedDevice, ok bool, err error)
	RetrieveByDevice(deviceID string) (_ *AuthorizedDevice, ok bool, err error)
	RetrieveAll(userID string) Iterator
	Persist(d *AuthorizedDevice) error
	Delete(deviceID, userID string) error
	Close()
}
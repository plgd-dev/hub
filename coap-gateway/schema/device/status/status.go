package status

import "time"

const ResourceType string = "x.cloud.device.status"
const Title string = "Device cloud status"

var Interfaces = []string{"oic.if.baseline"}
var ResourceTypes = []string{ResourceType}

type State uint8

const (
	State_Offline State = 0
	State_Online  State = 1
)

// Status is the resource published and maintained by plgd cloud.
// - signup: resource published
// - signin: content changed -> set state to online and set validUntil in unix timestap. (0 means infinite)
// - signout/close connection: content changed -> set state to offline
type Status struct {
	ResourceTypes []string `json:"rt"`
	Interfaces    []string `json:"if"`
	State         State    `json:"state"`
	ValidUntil    int64    `json:"validUntil"`
}

// IsOnline evaluate online state
func (s Status) IsOnline() bool {
	if s.State == State_Offline {
		return false
	}
	if s.ValidUntil <= 0 {
		return s.State == State_Online
	}
	return time.Now().Before(time.Unix(s.ValidUntil, 0))
}

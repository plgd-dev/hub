package events

type Device struct {
	ID string `json:"di"`
}

// https://github.com/openconnectivityfoundation/cloud-services/blob/master/swagger2.0/oic.r.cloudapiforcloudservices.swagger.json#/definitions/DevicesOnlineEvent
type DevicesOnline []Device

// https://github.com/openconnectivityfoundation/cloud-services/blob/master/swagger2.0/oic.r.cloudapiforcloudservices.swagger.json#/definitions/DevicesOfflineEvent
type DevicesOffline []Device

// https://github.com/openconnectivityfoundation/cloud-services/blob/master/swagger2.0/oic.r.cloudapiforcloudservices.swagger.json#/definitions/DevicesRegisteredEvent
type DevicesRegistered []Device

// https://github.com/openconnectivityfoundation/cloud-services/blob/master/swagger2.0/oic.r.cloudapiforcloudservices.swagger.json#/definitions/DevicesUnregisteredEvent
type DevicesUnregistered []Device

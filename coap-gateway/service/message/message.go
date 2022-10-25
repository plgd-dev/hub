package message

import (
	"fmt"
	"strings"

	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/hub/v2/coap-gateway/uri"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
)

// URIToDeviceIDHref convert uri to deviceID and href. Expected input "/api/v1/devices/{deviceID}/{Href}".
func URIToDeviceIDHref(msg *mux.Message) (deviceID, href string, err error) {
	wholePath, err := msg.Options().Path()
	if err != nil {
		return "", "", fmt.Errorf("cannot parse deviceID, href from uri: %w", err)
	}
	deviceIDHref := strings.TrimPrefix(wholePath, uri.ResourceRoute)
	if deviceIDHref[0] == '/' {
		deviceIDHref = deviceIDHref[1:]
	}
	r := commands.ResourceIdFromString(deviceIDHref)
	if r == nil {
		return "", "", fmt.Errorf("cannot parse deviceID, href from uri %v", wholePath)
	}
	return r.GetDeviceId(), r.GetHref(), nil
}

// Get resource interface from request query.
func GetResourceInterface(msg *mux.Message) string {
	queries, _ := msg.Options().Queries()
	for _, query := range queries {
		if strings.HasPrefix(query, "if=") {
			return strings.TrimLeft(query, "if=")
		}
	}
	return ""
}

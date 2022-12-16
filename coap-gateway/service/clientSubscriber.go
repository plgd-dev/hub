package service

import grpcClient "github.com/plgd-dev/hub/v2/grpc-gateway/client"

// Replace deviceSubscriber instance in the client.
func (c *session) replaceDeviceSubscriber(deviceSubscriber *grpcClient.DeviceSubscriber) *grpcClient.DeviceSubscriber {
	c.private.mutex.Lock()
	defer c.private.mutex.Unlock()
	s := c.private.deviceSubscriber
	c.private.deviceSubscriber = deviceSubscriber
	return s
}

// Replace the deviceSubscriber instance in the client with nil and clean up the previous instance.
func (c *session) closeDeviceSubscriber() error {
	deviceSubscriber := c.replaceDeviceSubscriber(nil)
	if deviceSubscriber != nil {
		return deviceSubscriber.Close()
	}
	return nil
}

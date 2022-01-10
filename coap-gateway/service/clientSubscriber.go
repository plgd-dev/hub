package service

import grpcClient "github.com/plgd-dev/hub/v2/grpc-gateway/client"

// Replace deviceSubscriber instance in the client.
func (client *Client) replaceDeviceSubscriber(deviceSubscriber *grpcClient.DeviceSubscriber) *grpcClient.DeviceSubscriber {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	c := client.deviceSubscriber
	client.deviceSubscriber = deviceSubscriber
	return c
}

// Replace the deviceSubscriber instance in the client with nil and clean up the previous instance.
func (client *Client) closeDeviceSubscriber() error {
	deviceSubscriber := client.replaceDeviceSubscriber(nil)
	if deviceSubscriber != nil {
		return deviceSubscriber.Close()
	}
	return nil
}

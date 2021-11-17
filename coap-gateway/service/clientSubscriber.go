package service

import grpcClient "github.com/plgd-dev/hub/grpc-gateway/client"

func (client *Client) replaceDeviceSubscriber(deviceSubscriber *grpcClient.DeviceSubscriber) *grpcClient.DeviceSubscriber {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	c := client.deviceSubscriber
	client.deviceSubscriber = deviceSubscriber
	return c
}

func (client *Client) closeDeviceSubscriber() error {
	deviceSubscriber := client.replaceDeviceSubscriber(nil)
	if deviceSubscriber != nil {
		return deviceSubscriber.Close()
	}
	return nil
}

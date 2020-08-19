package client

import kitNetCoap "github.com/plgd-dev/kit/net/coap"

// WithInterface updates/gets resource with interface directly from a device.
func WithInterface(resourceInterface string) ResourceInterfaceOption {
	return ResourceInterfaceOption{
		resourceInterface: resourceInterface,
	}
}

// WithSkipShadow gets resource directly from a device without using interface for client client.
func WithSkipShadow() SkipShadowOption {
	return SkipShadowOption{}
}

type ResourceInterfaceOption struct {
	resourceInterface string
}

func (r ResourceInterfaceOption) applyOnGet(opts getOptions) getOptions {
	if r.resourceInterface != "" {
		opts.resourceInterface = r.resourceInterface
	}
	return opts
}

func (r ResourceInterfaceOption) applyOnUpdate(opts updateOptions) updateOptions {
	if r.resourceInterface != "" {
		opts.resourceInterface = r.resourceInterface
	}
	return opts
}

type SkipShadowOption struct {
}

func (r SkipShadowOption) applyOnGet(opts getOptions) getOptions {
	opts.skipShadow = true
	return opts
}

// GetOption option definition.
type GetOption = interface {
	applyOnGet(opts getOptions) getOptions
}

type getOptions struct {
	resourceInterface string
	skipShadow        bool
	codec             kitNetCoap.Codec
}

type updateOptions struct {
	resourceInterface string
	codec             kitNetCoap.Codec
}

// UpdateOption option definition.
type UpdateOption = interface {
	applyOnUpdate(opts updateOptions) updateOptions
}

// UpdateOption option definition.
type GetDevicesOption = interface {
	applyOnGetDevices(opts getDevicesOptions) getDevicesOptions
}

func WithDeviceIDs(deviceIDs ...string) DeviceIDsOption {
	return DeviceIDsOption{
		deviceIDs: deviceIDs,
	}
}

type DeviceIDsOption struct {
	deviceIDs []string
}

type getDevicesOptions struct {
	deviceIDs     []string
	resourceTypes []string
}

func (r DeviceIDsOption) applyOnGetDevices(opts getDevicesOptions) getDevicesOptions {
	opts.deviceIDs = r.deviceIDs
	return opts
}

func WithResourceTypes(resourceTypes ...string) ResourceTypesOption {
	return ResourceTypesOption{
		resourceTypes: resourceTypes,
	}
}

type ResourceTypesOption struct {
	resourceTypes []string
}

func (r ResourceTypesOption) applyOnGetDevices(opts getDevicesOptions) getDevicesOptions {
	opts.resourceTypes = r.resourceTypes
	return opts
}

func WithCodec(codec kitNetCoap.Codec) CodecOption {
	return CodecOption{
		codec: codec,
	}
}

type CodecOption struct {
	codec kitNetCoap.Codec
}

func (r CodecOption) applyOnGet(opts getOptions) getOptions {
	opts.codec = r.codec
	return opts
}

func (r CodecOption) applyOnUpdate(opts updateOptions) updateOptions {
	opts.codec = r.codec
	return opts
}

func (r CodecOption) applyOnObserve(opts observeOptions) observeOptions {
	opts.codec = r.codec
	return opts
}

type observeOptions struct {
	codec kitNetCoap.Codec
}

// ObserveOption option definition.
type ObserveOption = interface {
	applyOnObserve(opts observeOptions) observeOptions
}

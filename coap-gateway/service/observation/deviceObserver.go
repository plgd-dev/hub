package observation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/schema/resources"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/hub/v2/coap-gateway/resource"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgStrings "github.com/plgd-dev/hub/v2/pkg/strings"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ObservationType int

type GrpcGatewayClient interface {
	GetDevicesMetadata(ctx context.Context, in *pb.GetDevicesMetadataRequest, opts ...grpc.CallOption) (pb.GrpcGateway_GetDevicesMetadataClient, error)
	GetResourceLinks(ctx context.Context, in *pb.GetResourceLinksRequest, opts ...grpc.CallOption) (pb.GrpcGateway_GetResourceLinksClient, error)
}

type ResourceAggregateClient interface {
	UnpublishResourceLinks(ctx context.Context, in *commands.UnpublishResourceLinksRequest, opts ...grpc.CallOption) (*commands.UnpublishResourceLinksResponse, error)
}

const (
	ObservationType_Detect      ObservationType = 0 // default, detect if /oic/res is observable using GET method, if not fallback to per resource observations
	ObservationType_PerDevice   ObservationType = 1 // single /oic/res observation
	ObservationType_PerResource ObservationType = 2 // fallback, observation of every published resource
)

// DeviceObserver is a type that sets up resources observation for a single device.
type DeviceObserver struct {
	deviceID              string
	observationType       ObservationType
	shadowSynchronization commands.ShadowSynchronization
	rdClient              GrpcGatewayClient
	resourcesObserver     *resourcesObserver
	logger                log.Logger
}

type DeviceObserverConfig struct {
	observationType          ObservationType
	shadowSynchronization    commands.ShadowSynchronization
	shadowSynchronizationSet bool
	logger                   log.Logger
}

type ClientConn interface {
	Get(ctx context.Context, path string, opts ...message.Option) (*pool.Message, error)
	Observe(ctx context.Context, path string, observeFunc func(req *pool.Message), opts ...message.Option) (Observation, error)
	ReleaseMessage(m *pool.Message)
	RemoteAddr() net.Addr
}

type Option interface {
	Apply(o *DeviceObserverConfig)
}

// Force observationType
type ObservationTypeOpt struct {
	observationType ObservationType
}

func (o ObservationTypeOpt) Apply(opts *DeviceObserverConfig) {
	opts.observationType = o.observationType
}

func WithObservationType(observationType ObservationType) ObservationTypeOpt {
	return ObservationTypeOpt{
		observationType: observationType,
	}
}

// Force shadowSynchronization value
type ShadowSynchronizationOpt struct {
	shadowSynchronization commands.ShadowSynchronization
}

func (o ShadowSynchronizationOpt) Apply(opts *DeviceObserverConfig) {
	opts.shadowSynchronization = o.shadowSynchronization
	opts.shadowSynchronizationSet = true
}

func WithShadowSynchronization(shadowSynchronization commands.ShadowSynchronization) ShadowSynchronizationOpt {
	return ShadowSynchronizationOpt{
		shadowSynchronization: shadowSynchronization,
	}
}

// Set logger option
type LoggerOpt struct {
	logger log.Logger
}

func WithLogger(logger log.Logger) LoggerOpt {
	return LoggerOpt{
		logger: logger,
	}
}

func (o LoggerOpt) Apply(opts *DeviceObserverConfig) {
	opts.logger = o.logger
}

func prepareSetupDeviceObserver(ctx context.Context, deviceID string, coapConn ClientConn, rdClient GrpcGatewayClient, raClient ResourceAggregateClient, cfg DeviceObserverConfig) (DeviceObserverConfig, []*commands.Resource, error) {
	links, sequence, err := GetResourceLinks(ctx, coapConn, resources.ResourceURI)
	switch {
	case err == nil:
		if cfg.observationType == ObservationType_Detect {
			cfg.observationType, err = detectObservationType(links)
			if err != nil {
				cfg.logger.Errorf("%w", err)
				cfg.observationType = ObservationType_PerDevice
			}
		}
	case errors.Is(err, context.Canceled):
		// the connection has been closed
		return cfg, nil, err
	default:
		cfg.logger.Debugf("cannot to get resource links from the device(%v): %w", deviceID, err)
		// the device doesn't have a /oic/res resource or a timeout has occurred
		if cfg.observationType == ObservationType_Detect {
			cfg.observationType = ObservationType_PerDevice
		}
	}

	published, err := getPublishedResources(ctx, rdClient, deviceID)
	if err != nil {
		return cfg, nil, err
	}

	published, err = cleanUpPublishedResources(ctx, raClient, deviceID, coapConn.RemoteAddr().String(), sequence, published, links)
	if err != nil {
		return cfg, nil, err
	}
	return cfg, published, nil
}

// Create new deviceObserver with given settings
func NewDeviceObserver(ctx context.Context, deviceID string, coapConn ClientConn, rdClient GrpcGatewayClient, raClient ResourceAggregateClient, callbacks ResourcesObserverCallbacks, opts ...Option) (*DeviceObserver, error) {
	createError := func(err error) error {
		return fmt.Errorf("cannot create device observer: %w", err)
	}
	if deviceID == "" {
		return nil, createError(emptyDeviceIDError())
	}

	cfg := DeviceObserverConfig{
		logger: log.Get(),
	}
	for _, o := range opts {
		o.Apply(&cfg)
	}

	if !cfg.shadowSynchronizationSet {
		shadowSynchronization, err := loadShadowSynchronization(ctx, rdClient, deviceID)
		if err != nil {
			return nil, createError(err)
		}
		cfg.shadowSynchronization = shadowSynchronization
	}

	if cfg.shadowSynchronization == commands.ShadowSynchronization_DISABLED {
		return &DeviceObserver{
			deviceID:              deviceID,
			shadowSynchronization: commands.ShadowSynchronization_DISABLED,
		}, nil
	}

	cfg, published, err := prepareSetupDeviceObserver(ctx, deviceID, coapConn, rdClient, raClient, cfg)
	if err != nil {
		return nil, createError(err)
	}

	if cfg.observationType == ObservationType_PerDevice {
		resourcesObserver, err := createDiscoveryResourceObserver(ctx, deviceID, coapConn, callbacks, cfg.logger)
		if err == nil {
			return &DeviceObserver{
				deviceID:              deviceID,
				observationType:       ObservationType_PerDevice,
				shadowSynchronization: cfg.shadowSynchronization,
				rdClient:              rdClient,
				resourcesObserver:     resourcesObserver,
				logger:                cfg.logger,
			}, nil
		}
		cfg.logger.Debugf("NewDeviceObserverWithResourceShadow: failed to create /oic/res observation for device(%v): %v", deviceID, err)
	}

	resourcesObserver, err := createPublishedResourcesObserver(ctx, deviceID, coapConn, callbacks, published, cfg.logger)
	if err != nil {
		return nil, createError(err)
	}
	return &DeviceObserver{
		deviceID:              deviceID,
		observationType:       ObservationType_PerResource,
		shadowSynchronization: cfg.shadowSynchronization,
		rdClient:              rdClient,
		resourcesObserver:     resourcesObserver,
		logger:                cfg.logger,
	}, nil
}

func emptyDeviceIDError() error {
	return fmt.Errorf("empty deviceID")
}

func IsDiscoveryResourceObservable(links schema.ResourceLinks) (bool, error) {
	if len(links) == 0 {
		return false, fmt.Errorf("no links")
	}
	resourceHref := resources.ResourceURI
	observeInterface := interfaces.OC_IF_B
	res, ok := links.GetResourceLink(resourceHref)
	if !ok {
		return false, fmt.Errorf("resourceLink for href(%v) not found", resourceHref)
	}

	observable := res.Policy.BitMask.Has(schema.Observable)
	if !observable {
		return observable, nil
	}

	return pkgStrings.Contains(res.Interfaces, observeInterface), nil
}

func detectObservationType(links schema.ResourceLinks) (ObservationType, error) {
	ok, err := IsDiscoveryResourceObservable(links)
	if err != nil {
		return ObservationType_PerDevice, fmt.Errorf("cannot determine whether /oic/res is observable: %w", err)
	}
	if ok {
		return ObservationType_PerDevice, nil
	}
	return ObservationType_PerResource, nil
}

// Retrieve device metadata and get ShadowSynchronization value.
func loadShadowSynchronization(ctx context.Context, rdClient GrpcGatewayClient, deviceID string) (commands.ShadowSynchronization, error) {
	metadataError := func(err error) error {
		return fmt.Errorf("cannot get device(%v) metadata: %w", deviceID, err)
	}
	if deviceID == "" {
		return commands.ShadowSynchronization_UNSET, metadataError(fmt.Errorf("invalid deviceID"))
	}
	deviceMetadataClient, err := rdClient.GetDevicesMetadata(ctx, &pb.GetDevicesMetadataRequest{
		DeviceIdFilter: []string{deviceID},
	})
	if err != nil {
		if status.Convert(err).Code() == codes.NotFound {
			return commands.ShadowSynchronization_UNSET, nil
		}
		return commands.ShadowSynchronization_UNSET, metadataError(err)
	}
	shadowSynchronization := commands.ShadowSynchronization_UNSET
	for {
		m, err := deviceMetadataClient.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if status.Convert(err).Code() == codes.NotFound {
			return commands.ShadowSynchronization_UNSET, nil
		}
		if err != nil {
			return commands.ShadowSynchronization_UNSET, metadataError(err)
		}
		shadowSynchronization = m.GetShadowSynchronization()
	}
	return shadowSynchronization, nil
}

// Create observer with a single observation for /oic/res resource.
func createDiscoveryResourceObserver(ctx context.Context, deviceID string, coapConn ClientConn, callbacks ResourcesObserverCallbacks, logger log.Logger) (*resourcesObserver, error) {
	resourcesObserver := newResourcesObserver(deviceID, coapConn, callbacks, logger)
	err := resourcesObserver.addResource(ctx, &commands.Resource{
		DeviceId: resourcesObserver.deviceID,
		Href:     resources.ResourceURI,
		Policy:   &commands.Policy{BitFlags: int32(schema.Observable)},
	}, interfaces.OC_IF_B)
	if err != nil {
		resourcesObserver.CleanObservedResources(ctx)
		return nil, err
	}
	return resourcesObserver, nil
}

// Create observer with a single observations for all published resources.
func createPublishedResourcesObserver(ctx context.Context, deviceID string, coapConn ClientConn, callbacks ResourcesObserverCallbacks, published []*commands.Resource, logger log.Logger) (*resourcesObserver, error) {
	resourcesObserver := newResourcesObserver(deviceID, coapConn, callbacks, logger)

	err := resourcesObserver.addResources(ctx, published)
	if err != nil {
		resourcesObserver.CleanObservedResources(ctx)
		return nil, err
	}
	return resourcesObserver, nil
}

func (d *DeviceObserver) GetDeviceID() string {
	return d.deviceID
}

func (d *DeviceObserver) GetObservationType() ObservationType {
	return d.observationType
}

func (d *DeviceObserver) GetShadowSynchronization() commands.ShadowSynchronization {
	return d.shadowSynchronization
}

// Get list of observed resources for device.
func (d *DeviceObserver) GetResources() ([]*commands.ResourceId, error) {
	getResourcesError := func(err error) error {
		return fmt.Errorf("cannot get observed resources: %w", err)
	}
	if d.shadowSynchronization == commands.ShadowSynchronization_DISABLED {
		return nil, nil
	}
	if d.resourcesObserver == nil {
		return nil, getResourcesError(fmt.Errorf("resources observer is nil"))
	}
	return d.resourcesObserver.getResources(), nil
}

// Remove all observations.
func (d *DeviceObserver) Clean(ctx context.Context) {
	if d.shadowSynchronization == commands.ShadowSynchronization_DISABLED {
		return
	}
	d.resourcesObserver.CleanObservedResources(ctx)
}

// Retrieve resources published for device.
func getPublishedResources(ctx context.Context, rdClient GrpcGatewayClient, deviceID string) ([]*commands.Resource, error) {
	resourceLinksError := func(err error) error {
		return fmt.Errorf("cannot get resource links for device(%v): %w", deviceID, err)
	}
	if deviceID == "" {
		return nil, resourceLinksError(emptyDeviceIDError())
	}
	getResourceLinksClient, err := rdClient.GetResourceLinks(ctx, &pb.GetResourceLinksRequest{
		DeviceIdFilter: []string{deviceID},
	})
	if err != nil {
		if status.Convert(err).Code() == codes.NotFound {
			return nil, nil
		}
		return nil, resourceLinksError(err)
	}
	resources := make([]*commands.Resource, 0, 8)
	for {
		m, err := getResourceLinksClient.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if status.Convert(err).Code() == codes.NotFound {
			return nil, nil
		}
		if err != nil {
			return nil, resourceLinksError(err)
		}
		resources = append(resources, m.GetResources()...)
	}
	return resources, nil
}

func diffResources(publishedResources commands.Resources, deviceResources schema.ResourceLinks) (validResources []*commands.Resource, toUnpublishResourceInstanceIds []int64) {
	validResources = make([]*commands.Resource, 0, len(publishedResources))
	toUnpublishResourceInstanceIds = make([]int64, 0, len(publishedResources))
	publishedResources.Sort()
	deviceResources.Sort()
	var j int
	for _, res := range publishedResources {
		for j < len(deviceResources) && deviceResources[j].Href < res.GetHref() {
			j++
		}
		if j >= len(deviceResources) {
			break
		}
		if deviceResources[j].Href == res.GetHref() {
			validResources = append(validResources, res)
		} else {
			toUnpublishResourceInstanceIds = append(toUnpublishResourceInstanceIds, resource.GetInstanceID(res.GetHref()))
		}
	}
	return validResources, toUnpublishResourceInstanceIds
}

func cleanUpPublishedResources(ctx context.Context, raClient ResourceAggregateClient, deviceID, connectionID string, sequence uint64, publishedResources commands.Resources, deviceResources schema.ResourceLinks) ([]*commands.Resource, error) {
	if len(publishedResources) == 0 {
		return nil, nil
	}
	if len(deviceResources) == 0 {
		return publishedResources, nil
	}

	validResources, toUnpublishResourceInstanceIds := diffResources(publishedResources, deviceResources)

	for _, res := range publishedResources {
		if _, ok := deviceResources.GetResourceLink(res.GetHref()); ok {
			validResources = append(validResources, res)
		} else {
			toUnpublishResourceInstanceIds = append(toUnpublishResourceInstanceIds, resource.GetInstanceID(res.GetHref()))
		}
	}
	if len(toUnpublishResourceInstanceIds) > 0 {
		_, err := raClient.UnpublishResourceLinks(ctx, &commands.UnpublishResourceLinksRequest{
			InstanceIds: toUnpublishResourceInstanceIds,
			DeviceId:    deviceID,
			CommandMetadata: &commands.CommandMetadata{
				ConnectionId: connectionID,
				Sequence:     sequence,
			},
		})
		if err != nil {
			return nil, err
		}
	}

	return validResources, nil
}

// Add observation of published resources.
//
// Function does nothing if device shadow is disabled or /oic/res observation type (ObservationType_PerDevice)
// is active. Only if observation per published resource (ObservationType_PerResource) is active does the
// function try to add the given resources to active observations.
func (d *DeviceObserver) AddPublishedResources(ctx context.Context, resources []*commands.Resource) error {
	if d.shadowSynchronization == commands.ShadowSynchronization_DISABLED {
		d.logger.Debugf("add published resources skipped: device shadow disabled")
		return nil
	}
	if d.observationType == ObservationType_PerDevice {
		d.logger.Debugf("add published resources skipped: /oic/res observation active")
		return nil
	}
	return d.resourcesObserver.addResources(ctx, resources)
}

// Remove observation of published resources.
//
// Function does nothing if device shadow is disabled or /oic/res observation type (ObservationType_PerDevice)
// is active. Only if observation per published resource (ObservationType_PerResource) is active does the
// function try to cancel the observations of given resources.
func (d *DeviceObserver) RemovePublishedResources(ctx context.Context, resourceHrefs []string) {
	if d.shadowSynchronization == commands.ShadowSynchronization_DISABLED {
		d.logger.Debugf("remove published resources skipped: device shadow disabled")
		return
	}
	if d.observationType == ObservationType_PerDevice {
		d.logger.Debugf("remove published resources skipped: /oic/res observation active")
		return
	}
	d.resourcesObserver.cancelResourcesObservations(ctx, resourceHrefs)
}

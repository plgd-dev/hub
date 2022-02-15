package observation

import (
	"context"
	"fmt"
	"io"

	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/schema/resources"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/go-coap/v2/tcp/message/pool"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ObservationType int

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
	rdClient              pb.GrpcGatewayClient
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
	Observe(ctx context.Context, path string, observeFunc func(req *pool.Message), opts ...message.Option) (*tcp.Observation, error)
	ReleaseMessage(m *pool.Message)
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

// Create new deviceObserver with given settings
func NewDeviceObserver(ctx context.Context, deviceID string, coapConn ClientConn, rdClient pb.GrpcGatewayClient, callbacks ResourcesObserverCallbacks, opts ...Option) (*DeviceObserver, error) {
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

	var err error
	if cfg.observationType == ObservationType_Detect {
		cfg.observationType, err = detectObservationType(ctx, coapConn)
		if err != nil {
			cfg.logger.Errorf("%w", err)
			cfg.observationType = ObservationType_PerDevice
		}
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

	resourcesObserver, err := createPublishedResourcesObserver(ctx, rdClient, deviceID, coapConn, callbacks, cfg.logger)
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

func isDiscoveryResourceObservable(ctx context.Context, coapConn ClientConn) (bool, error) {
	return IsResourceObservableWithInterface(ctx, coapConn, resources.ResourceURI, resources.ResourceType, interfaces.OC_IF_B)
}

func detectObservationType(ctx context.Context, coapConn ClientConn) (ObservationType, error) {
	ok, err := isDiscoveryResourceObservable(ctx, coapConn)
	if err != nil {
		return ObservationType_PerDevice, fmt.Errorf("cannot determine whether /oic/res is observable: %w", err)
	}
	if ok {
		return ObservationType_PerDevice, nil
	}
	return ObservationType_PerResource, nil
}

// Retrieve device metadata and get ShadowSynchronization value.
func loadShadowSynchronization(ctx context.Context, rdClient pb.GrpcGatewayClient, deviceID string) (commands.ShadowSynchronization, error) {
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
		if err == io.EOF {
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
func createPublishedResourcesObserver(ctx context.Context, rdClient pb.GrpcGatewayClient, deviceID string, coapConn ClientConn, callbacks ResourcesObserverCallbacks, logger log.Logger) (*resourcesObserver, error) {
	resourcesObserver := newResourcesObserver(deviceID, coapConn, callbacks, logger)

	published, err := getPublishedResources(ctx, rdClient, deviceID)
	if err != nil {
		return nil, err
	}

	err = resourcesObserver.addResources(ctx, published)
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
func getPublishedResources(ctx context.Context, rdClient pb.GrpcGatewayClient, deviceID string) ([]*commands.Resource, error) {
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
		if err == io.EOF {
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

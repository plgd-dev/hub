package observation

import (
	"context"
	"fmt"
	"io"

	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/schema/resources"
	"github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	"github.com/plgd-dev/hub/pkg/log"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ObservationType int

const (
	ObservationType_PerDevice   ObservationType = 0 // default, single /oic/res observation
	ObservationType_PerResource ObservationType = 1 // fallback, observation of every published resource
)

// DeviceObserver is a type that sets up resources observation for a single device.
type DeviceObserver struct {
	deviceID              string
	observationType       ObservationType
	shadowSynchronization commands.ShadowSynchronization
	rdClient              pb.GrpcGatewayClient
	resourcesObserver     *resourcesObserver
}

func NewDeviceObserverWithResourceShadow(ctx context.Context, deviceID string, shadowSynchronization commands.ShadowSynchronization, coapConn *tcp.ClientConn, rdClient pb.GrpcGatewayClient, onObserveResource OnObserveResource, onGetResourceContent OnGetResourceContent) (*DeviceObserver, error) {
	createError := func(err error) error {
		return fmt.Errorf("cannot create device observer: %w", err)
	}
	if deviceID == "" {
		return nil, createError(fmt.Errorf("empty deviceID"))
	}
	if shadowSynchronization == commands.ShadowSynchronization_DISABLED {
		return &DeviceObserver{
			deviceID:              deviceID,
			shadowSynchronization: commands.ShadowSynchronization_DISABLED,
		}, nil
	}

	// resourcesObserver, err := createDiscoveryResourceObserver(ctx, deviceID, coapConn, onObserveResource, onGetResourceContent)
	// if err == nil && resourcesObserver != nil {
	// 	return &DeviceObserver{
	// 		deviceID:              deviceID,
	// 		observationType:       ObservationType_PerDevice,
	// 		shadowSynchronization: shadowSynchronization,
	// 		rdClient:              rdClient,
	// 		resourcesObserver:     resourcesObserver,
	// 	}, nil
	// }
	// log.Debugf("NewDeviceObserverWithResourceShadow: failed to create /oic/res observation for device(%v): %v", deviceID, err)

	resourcesObserver, err := createPublishedResourcesObserver(ctx, rdClient, deviceID, coapConn, onObserveResource, onGetResourceContent)
	if err != nil {
		return nil, createError(err)
	}

	return &DeviceObserver{
		deviceID:              deviceID,
		observationType:       ObservationType_PerResource,
		shadowSynchronization: shadowSynchronization,
		rdClient:              rdClient,
		resourcesObserver:     resourcesObserver,
	}, nil
}

func NewDeviceObserver(ctx context.Context, deviceID string, coapConn *tcp.ClientConn, rdClient pb.GrpcGatewayClient, onObserveResource OnObserveResource, onGetResourceContent OnGetResourceContent) (*DeviceObserver, error) {
	createError := func(err error) error {
		return fmt.Errorf("cannot create device observer: %w", err)
	}
	if deviceID == "" {
		return nil, createError(fmt.Errorf("empty deviceID"))
	}
	shadowSynchronization, err := loadShadowSynchronization(ctx, rdClient, deviceID)
	if err != nil {
		return nil, createError(err)
	}
	return NewDeviceObserverWithResourceShadow(ctx, deviceID, shadowSynchronization, coapConn, rdClient, onObserveResource, onGetResourceContent)
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
//
// Return value is a future which will contain bool value signifying whether the observation
// was successfully opened or not.
func createDiscoveryResourceObserver(ctx context.Context, deviceID string, coapConn *tcp.ClientConn, onObserveResource OnObserveResource,
	onGetResourceContent OnGetResourceContent) (*resourcesObserver, error) {
	resourcesObserver := newResourcesObserver(deviceID, coapConn, onObserveResource, onGetResourceContent)
	opened, err := resourcesObserver.addAndWaitForResource(ctx, &commands.Resource{
		DeviceId: resourcesObserver.deviceID,
		Href:     resources.ResourceURI,
		Policy:   &commands.Policy{BitFlags: int32(schema.Observable)},
	}, interfaces.OC_IF_B)
	if err != nil {
		resourcesObserver.CleanObservedResources(ctx)
		return nil, err
	}
	if !opened {
		return nil, nil
	}
	return resourcesObserver, nil
}

func createPublishedResourcesObserver(ctx context.Context, rdClient pb.GrpcGatewayClient, deviceID string, coapConn *tcp.ClientConn, onObserveResource OnObserveResource,
	onGetResourceContent OnGetResourceContent) (*resourcesObserver, error) {
	resourcesObserver := newResourcesObserver(deviceID, coapConn, onObserveResource, onGetResourceContent)

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
func (d *DeviceObserver) GetResources(deviceID string, instanceIDs []int64) ([]*commands.ResourceId, error) {
	if deviceID != d.deviceID {
		return nil, fmt.Errorf("cannot get observed resources: invalid deviceID")
	}
	if d.resourcesObserver == nil {
		return nil, fmt.Errorf("cannot get observed resources: resources observer is nil")
	}
	return d.resourcesObserver.getResources(instanceIDs), nil
}

func (d *DeviceObserver) Clean(ctx context.Context) {
	d.resourcesObserver.CleanObservedResources(ctx)
}

// Retrieve resources published for device.
func getPublishedResources(ctx context.Context, rdClient pb.GrpcGatewayClient, deviceID string) ([]*commands.Resource, error) {
	resourceLinksError := func(err error) error {
		return fmt.Errorf("cannot get resource links for device(%v): %w", deviceID, err)
	}
	if deviceID == "" {
		return nil, resourceLinksError(fmt.Errorf("empty deviceID"))
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
		log.Debugf("add published resources skipped, device shadow disabled")
		return nil
	}
	if d.observationType == ObservationType_PerDevice {
		log.Debugf("add published resources skipped, /oic/res observation active")
		return nil
	}
	return d.resourcesObserver.addResources(ctx, resources)
}

// Remove observation of published resources.
//
// Function does nothing if device shadow is disabled or /oic/res observation type (ObservationType_PerDevice)
// is active. Only if observation per published resource (ObservationType_PerResource) is active does the
// function try to add the given resources to active observations.
func (d *DeviceObserver) RemovePublishedResources(ctx context.Context, resourceHrefs []string) error {
	if d.shadowSynchronization == commands.ShadowSynchronization_DISABLED {
		log.Debugf("add published resources skipped, device shadow disabled")
		return nil
	}
	if d.observationType == ObservationType_PerDevice {
		log.Debugf("add published resources skipped, /oic/res observation active")
		return nil
	}
	d.resourcesObserver.cancelResourcesObservations(ctx, resourceHrefs)
	return nil
}
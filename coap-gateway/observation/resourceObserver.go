package observation

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/plgd-dev/go-coap/v2/tcp"
	"github.com/plgd-dev/hub/coap-gateway/resource"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	"github.com/plgd-dev/hub/pkg/log"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type OnAddObservedResource = func(ctx context.Context, deviceID string, instanceID int64, obs *ObservedResource)

type ResourceObserver struct {
	lock                  sync.Mutex
	resources             map[string]map[int64]*ObservedResource // [deviceID][instanceID]
	shadowSynchronization commands.ShadowSynchronization
	rdClient              pb.GrpcGatewayClient
	onAddResource         OnAddObservedResource
}

func NewResourceObserver(rdClient pb.GrpcGatewayClient, onAddResource OnAddObservedResource) *ResourceObserver {
	return &ResourceObserver{
		resources:     make(map[string]map[int64]*ObservedResource),
		rdClient:      rdClient,
		onAddResource: onAddResource,
	}
}

func (o *ResourceObserver) addResourceLocked(ctx context.Context, res *commands.Resource) *ObservedResource {
	resID := res.GetResourceID()
	log.Debugf("observation of resource %v requested", resID)
	instanceID := resource.GetInstanceID(resID.GetHref())
	deviceID := resID.GetDeviceId()
	href := resID.GetHref()
	if _, ok := o.resources[deviceID]; !ok {
		o.resources[deviceID] = make(map[int64]*ObservedResource)
	}
	if _, ok := o.resources[deviceID][instanceID]; ok {
		return nil
	}
	obsRes := NewObservedResource(href, res.IsObservable())
	o.resources[deviceID][instanceID] = obsRes
	if o.onAddResource != nil {
		o.onAddResource(ctx, deviceID, instanceID, obsRes)
	}
	return obsRes
}

func (o *ResourceObserver) AddResource(ctx context.Context, res *commands.Resource) *ObservedResource {
	o.lock.Lock()
	defer o.lock.Unlock()
	return o.addResourceLocked(ctx, res)
}

func (o *ResourceObserver) addResourcesLocked(ctx context.Context, resources []*commands.Resource) []*ObservedResource {
	obsRes := make([]*ObservedResource, 0, len(resources))
	for _, resource := range resources {
		obsRes = append(obsRes, o.addResourceLocked(ctx, resource))
	}
	return obsRes
}

func (o *ResourceObserver) AddResources(ctx context.Context, resources []*commands.Resource) []*ObservedResource {
	o.lock.Lock()
	defer o.lock.Unlock()
	return o.addResourcesLocked(ctx, resources)
}

func (o *ResourceObserver) GetTrackedResources(deviceID string, instanceIDs []int64) []commands.ResourceId {
	getAllDeviceIDMatches := len(instanceIDs) == 0
	matches := make([]commands.ResourceId, 0, 16)

	o.lock.Lock()
	defer o.lock.Unlock()

	if deviceResourcesMap, ok := o.resources[deviceID]; ok {
		if getAllDeviceIDMatches {
			for _, value := range deviceResourcesMap {
				matches = append(matches, commands.ResourceId{
					DeviceId: deviceID,
					Href:     value.Href(),
				})
			}
		} else {
			for _, instanceID := range instanceIDs {
				if resource, ok := deviceResourcesMap[instanceID]; ok {
					matches = append(matches, commands.ResourceId{
						DeviceId: deviceID,
						Href:     resource.Href(),
					})
				}
			}
		}
	}

	return matches
}

func (o *ResourceObserver) removeResourceLocked(deviceID string, instanceID int64) {
	if device, ok := o.resources[deviceID]; ok {
		delete(device, instanceID)
		if len(o.resources[deviceID]) == 0 {
			delete(o.resources, deviceID)
		}
	}
}

func (o *ResourceObserver) popObservationLocked(deviceID string, instanceID int64) *tcp.Observation {
	log.Debugf("remove published resource ocf://%v/%v", deviceID, instanceID)

	if device, ok := o.resources[deviceID]; ok {
		if res, ok := device[instanceID]; ok {
			return res.PopObservation()
		}
	}
	return nil
}

func (o *ResourceObserver) popTrackedObservations(hrefs []string) []*tcp.Observation {
	observations := make([]*tcp.Observation, 0, 32)

	o.lock.Lock()
	defer o.lock.Unlock()

	for _, href := range hrefs {
		var instanceID int64
		var deviceID string
		for devID, devs := range o.resources {
			for insID, r := range devs {
				if r.Href() == href {
					instanceID = insID
					deviceID = devID
					break
				}
			}
		}

		obs := o.popObservationLocked(deviceID, instanceID)
		if obs != nil {
			observations = append(observations, obs)
		}
		o.removeResourceLocked(deviceID, instanceID)
		log.Debugf("canceling observation on resource %v%v", deviceID, href)
	}
	return observations
}

func (o *ResourceObserver) CancelResourcesObservations(ctx context.Context, hrefs []string) {
	observations := o.popTrackedObservations(hrefs)
	for _, obs := range observations {
		if err := obs.Cancel(ctx); err != nil {
			log.Debugf("cannot cancel resource observation: %w", err)
		}
	}
}

func (o *ResourceObserver) popObservedResourcesLocked() map[string]map[int64]*ObservedResource {
	observations := o.resources
	o.resources = make(map[string]map[int64]*ObservedResource)
	return observations
}

// cleanObservedResourcesLocked remove all device pbRA observation requested by cloud.
func (o *ResourceObserver) cleanObservedResourcesLocked(ctx context.Context) {
	for _, resources := range o.popObservedResourcesLocked() {
		for k, obs := range resources {
			if v := obs.PopObservation(); v != nil {
				if err := v.Cancel(ctx); err != nil {
					log.Errorf("cannot cancel resource('%v') observation: %w", k, err)
				}
			}
		}
	}
}

// cleanObservedResources remove all device pbRA observations requested by cloud under lock.
func (o *ResourceObserver) CleanObservedResources(ctx context.Context) {
	o.lock.Lock()
	defer o.lock.Unlock()
	o.cleanObservedResourcesLocked(ctx)
}

func (o *ResourceObserver) LoadShadowSynchronization(ctx context.Context, deviceID string) error {
	deviceMetadataClient, err := o.rdClient.GetDevicesMetadata(ctx, &pb.GetDevicesMetadataRequest{
		DeviceIdFilter: []string{deviceID},
	})
	if err != nil {
		if status.Convert(err).Code() == codes.NotFound {
			return nil
		}
		return fmt.Errorf("cannot get device(%v) metdata: %v", deviceID, err)
	}
	shadowSynchronization := commands.ShadowSynchronization_UNSET
	for {
		m, err := deviceMetadataClient.Recv()
		if err == io.EOF {
			break
		}
		if status.Convert(err).Code() == codes.NotFound {
			return nil
		}
		if err != nil {
			return fmt.Errorf("cannot get device(%v) metdata: %v", deviceID, err)
		}
		shadowSynchronization = m.GetShadowSynchronization()
	}
	o.lock.Lock()
	defer o.lock.Unlock()
	o.shadowSynchronization = shadowSynchronization
	return nil
}

func (o *ResourceObserver) SetShadowSynchronization(ctx context.Context, deviceID string, shadowSynchronization commands.ShadowSynchronization) commands.ShadowSynchronization {
	o.lock.Lock()
	defer o.lock.Unlock()
	if o.shadowSynchronization == shadowSynchronization {
		return o.shadowSynchronization
	}
	previous := o.shadowSynchronization

	o.shadowSynchronization = shadowSynchronization
	if shadowSynchronization == commands.ShadowSynchronization_DISABLED {
		// ctx = client.coapConn.Context()
		o.cleanObservedResourcesLocked(ctx)
	} else {
		o.registerObservationsForPublishedResourcesLocked(ctx, deviceID)
	}
	return previous
}

func (o *ResourceObserver) registerObservationsForPublishedResourcesLocked(ctx context.Context, deviceID string) {
	getResourceLinksClient, err := o.rdClient.GetResourceLinks(ctx, &pb.GetResourceLinksRequest{
		DeviceIdFilter: []string{deviceID},
	})
	if err != nil {
		if status.Convert(err).Code() == codes.NotFound {
			return
		}
		log.Errorf("signIn: cannot get resource links for the device %v: %w", deviceID, err)
		return
	}
	resources := make([]*commands.Resource, 0, 8)
	for {
		m, err := getResourceLinksClient.Recv()
		if err == io.EOF {
			break
		}
		if status.Convert(err).Code() == codes.NotFound {
			return
		}
		if err != nil {
			log.Errorf("signIn: cannot receive link for the device %v: %w", deviceID, err)
			return
		}
		resources = append(resources, m.GetResources()...)

	}
	o.addResourcesLocked(ctx, resources)
}

func (o *ResourceObserver) RegisterObservationsForPublishedResource(ctx context.Context, deviceID string) {
	o.lock.Lock()
	defer o.lock.Unlock()
	if o.shadowSynchronization == commands.ShadowSynchronization_DISABLED {
		return
	}
	o.registerObservationsForPublishedResourcesLocked(ctx, deviceID)
}

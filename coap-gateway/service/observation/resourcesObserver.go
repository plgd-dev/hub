package observation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
)

type (
	OnObserveResource                 = func(ctx context.Context, deviceID, resourceHref string, batch bool, notification *pool.Message) error
	OnGetResourceContent              = func(ctx context.Context, deviceID, resourceHref string, notification *pool.Message) error
	UpdateShadowSynchronizationStatus = func(ctx context.Context, deviceID string, status commands.ShadowSynchronizationStatus_Status, t time.Time) error
)

type ResourcesObserverCallbacks struct {
	OnObserveResource                 OnObserveResource
	OnGetResourceContent              OnGetResourceContent
	UpdateShadowSynchronizationStatus UpdateShadowSynchronizationStatus
}

func MakeResourcesObserverCallbacks(onObserveResource OnObserveResource, onGetResourceContent OnGetResourceContent, updateShadowSynchronizationStatus UpdateShadowSynchronizationStatus) ResourcesObserverCallbacks {
	return ResourcesObserverCallbacks{
		OnObserveResource:                 onObserveResource,
		OnGetResourceContent:              onGetResourceContent,
		UpdateShadowSynchronizationStatus: updateShadowSynchronizationStatus,
	}
}

// ResourcesObserver is a thread-safe type that handles observation of resources belonging to
// a single device.
//
// The resource observer keeps track of observed resources to avoid multiple observation of the
// same resource. Each new unique observation fires an event:
//   - If the resource is observable then the connection to COAP-GW (coapConn) is used to
//     register for observations from COAP-GW. Observation notifications are handled by the
//     onObserveResource callback.
//   - If the resource is not observable then a GET request is sent to COAP-GW to receive
//     the content of the resource and the response is handled by the onGetResourceContent
//     callback.
type resourcesObserver struct {
	lock      sync.Mutex
	deviceID  string
	resources observedResources
	coapConn  ClientConn
	callbacks ResourcesObserverCallbacks
	logger    log.Logger
}

// Create new Resource Observer.
//
// All arguments (coapConn, onObserveResource and onGetResourceContent) must be non-nil,
// otherwise the function will panic.
func newResourcesObserver(deviceID string, coapConn ClientConn, callbacks ResourcesObserverCallbacks, logger log.Logger) *resourcesObserver {
	fatalCannotCreate := func(err error) {
		log.Fatal("cannot create resource observer: %v", err)
	}
	if deviceID == "" {
		fatalCannotCreate(emptyDeviceIDError())
	}
	if coapConn == nil {
		fatalCannotCreate(fmt.Errorf("invalid coap-gateway connection"))
	}
	if callbacks.OnObserveResource == nil {
		fatalCannotCreate(fmt.Errorf("invalid onObserveResource callback"))
	}
	if callbacks.OnGetResourceContent == nil {
		fatalCannotCreate(fmt.Errorf("invalid onGetResourceContent callback"))
	}
	return &resourcesObserver{
		deviceID:  deviceID,
		resources: make(observedResources, 0, 1),
		coapConn:  coapConn,
		callbacks: callbacks,
		logger:    logger,
	}
}

// Add resource to observer with given interface and wait for initialization message.
func (o *resourcesObserver) addResource(ctx context.Context, res *commands.Resource, obsInterface string) error {
	o.lock.Lock()
	defer o.lock.Unlock()
	obs, err := o.addResourceLocked(res, obsInterface)
	if err == nil && obs != nil {
		o.notifyAboutStartShadowSynchronizationStatus(ctx)
		err = o.performObservationLocked([]*observedResource{obs})
		if err != nil {
			o.resources, _ = o.resources.removeByHref(obs.Href())
			defer o.notifyAboutFinishShadowSynchronizationStatus(ctx)
		}
	}
	return err
}

func (o *resourcesObserver) isSynchronizedLocked() bool {
	for _, res := range o.resources {
		if !res.synced.Load() {
			return false
		}
	}
	return true
}

func (o *resourcesObserver) resourceHasBeenSynchronized(ctx context.Context, href string) {
	if o.setSynchronizedAtResource(href) {
		o.notifyAboutFinishShadowSynchronizationStatus(ctx)
	}
}

// returns true if all resources are observed and synchronized
func (o *resourcesObserver) setSynchronizedAtResource(href string) bool {
	o.lock.Lock()
	defer o.lock.Unlock()
	i := o.resources.search(href)
	synced := false
	if i < len(o.resources) && o.resources[i].Equals(href) {
		synced = o.resources[i].synced.CompareAndSwap(false, true)
	} else {
		return false
	}
	if !synced {
		return false
	}
	if o.isSynchronizedLocked() {
		return true
	}
	return false
}

func (o *resourcesObserver) addResourceLocked(res *commands.Resource, obsInterface string) (*observedResource, error) {
	resID := res.GetResourceID()
	addObservationError := func(err error) error {
		return fmt.Errorf("cannot add resource observation: %w", err)
	}
	if o.deviceID == "" {
		return nil, addObservationError(emptyDeviceIDError())
	}
	if o.deviceID != resID.GetDeviceId() {
		return nil, addObservationError(fmt.Errorf("invalid deviceID(%v)", resID.GetDeviceId()))
	}
	href := resID.GetHref()
	if o.resources.contains(href) {
		return nil, nil
	}
	obsRes := NewObservedResource(href, obsInterface, res.IsObservable())
	o.resources = o.resources.insert(obsRes)
	return obsRes, nil
}

// Handle given resource.
//
// For observable resources subscribe to observations, for unobservable resources retrieve
// their content.
func (o *resourcesObserver) handleResource(ctx context.Context, obsRes *observedResource) error {
	if obsRes.Href() == commands.StatusHref {
		return nil
	}

	if obsRes.isObservable {
		obs, err := o.observeResource(ctx, obsRes)
		if err != nil {
			return err
		}
		obsRes.SetObservation(obs)
		return nil
	}
	return o.getResourceContent(ctx, obsRes.Href())
}

// Register to COAP-GW resource observation for given resource
func (o *resourcesObserver) observeResource(ctx context.Context, obsRes *observedResource) (Observation, error) {
	cannotObserveResourceError := func(deviceID, href string, err error) error {
		return fmt.Errorf("cannot observe resource /%v%v: %w", deviceID, href, err)
	}
	if o.deviceID == "" {
		return nil, cannotObserveResourceError(o.deviceID, obsRes.Href(), emptyDeviceIDError())
	}

	var opts []message.Option
	if obsRes.Interface() != "" {
		opts = append(opts, message.Option{
			ID:    message.URIQuery,
			Value: []byte("if=" + obsRes.Interface()),
		})
	}

	batchObservation := obsRes.resInterface == interfaces.OC_IF_B
	returnIfNonObservable := batchObservation && obsRes.Href() == resources.ResourceURI

	obs, err := o.coapConn.Observe(ctx, obsRes.Href(), func(msg *pool.Message) {
		if returnIfNonObservable {
			if _, errObs := msg.Observe(); errObs != nil {
				o.logger.Debugf("href: %v not observable err: %v", obsRes.Href(), errObs)
				return
			}
		}

		if err2 := o.callbacks.OnObserveResource(ctx, o.deviceID, obsRes.Href(), batchObservation, msg); err2 != nil {
			_ = o.logger.LogAndReturnError(cannotObserveResourceError(o.deviceID, obsRes.Href(), err2))
			return
		}
	}, opts...)
	if err != nil {
		return nil, cannotObserveResourceError(o.deviceID, obsRes.Href(), err)
	}
	if obs.Canceled() {
		return nil, cannotObserveResourceError(o.deviceID, obsRes.Href(), fmt.Errorf("resource not observable"))
	}
	return obs, nil
}

// Request resource content form COAP-GW
func (o *resourcesObserver) getResourceContent(ctx context.Context, href string) error {
	cannotGetResourceError := func(deviceID, href string, err error) error {
		return fmt.Errorf("cannot get resource /%v%v content: %w", deviceID, href, err)
	}
	if o.deviceID == "" {
		return cannotGetResourceError(o.deviceID, href, emptyDeviceIDError())
	}
	resp, err := o.coapConn.Get(ctx, href)
	if err != nil {
		return cannotGetResourceError(o.deviceID, href, err)
	}
	defer func() {
		if !resp.IsHijacked() {
			o.coapConn.ReleaseMessage(resp)
		}
	}()
	if err := o.callbacks.OnGetResourceContent(ctx, o.deviceID, href, resp); err != nil {
		return cannotGetResourceError(o.deviceID, href, err)
	}
	return nil
}

func (o *resourcesObserver) notifyAboutStartShadowSynchronizationStatus(ctx context.Context) {
	err := o.callbacks.UpdateShadowSynchronizationStatus(ctx, o.deviceID, commands.ShadowSynchronizationStatus_STARTED, time.Now())
	if err != nil {
		o.logger.Debugf("cannot update shadow synchronization status to finish: %v", err)
	}
}

func (o *resourcesObserver) notifyAboutFinishShadowSynchronizationStatus(ctx context.Context) {
	err := o.callbacks.UpdateShadowSynchronizationStatus(ctx, o.deviceID, commands.ShadowSynchronizationStatus_FINISHED, time.Now())
	if err != nil {
		o.logger.Debugf("cannot update shadow synchronization status to start: %v", err)
	}
}

func (o *resourcesObserver) performObservationLocked(obs []*observedResource) error {
	var errors []error
	for _, obsRes := range obs {
		if err := o.handleResource(context.Background(), obsRes); err != nil {
			o.resources, _ = o.resources.removeByHref(obsRes.Href())
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot perform observation: %v", errors)
	}
	return nil
}

// Add multiple resources to observer.
func (o *resourcesObserver) addResources(ctx context.Context, resources []*commands.Resource) error {
	o.lock.Lock()
	observedResources, err := o.addResourcesLocked(resources)
	if len(observedResources) > 0 {
		o.notifyAboutStartShadowSynchronizationStatus(ctx)
		err2 := o.performObservationLocked(observedResources)
		if err2 != nil {
			if err == nil {
				err = err2
			} else {
				err = fmt.Errorf("[%w, %v]", err, err2)
			}
			if o.isSynchronizedLocked() {
				defer o.notifyAboutFinishShadowSynchronizationStatus(ctx)
			}
		}
	}
	o.lock.Unlock()
	return err
}

func (o *resourcesObserver) addResourcesLocked(resources []*commands.Resource) ([]*observedResource, error) {
	var errors []error
	observedResources := make([]*observedResource, 0, len(resources))
	for _, resource := range resources {
		observedResource, err := o.addResourceLocked(resource, "")
		if err != nil {
			errors = append(errors, err)
		} else if observedResource != nil {
			observedResources = append(observedResources, observedResource)
		}
	}
	if len(errors) > 0 {
		return observedResources, fmt.Errorf("cannot add resources to observe: %v", errors)
	}
	return observedResources, nil
}

// Get list of observable and non-observable resources added to resourcesObserver.
func (o *resourcesObserver) getResources() []*commands.ResourceId {
	matches := make([]*commands.ResourceId, 0, 16)
	o.lock.Lock()
	defer o.lock.Unlock()
	for _, value := range o.resources {
		matches = append(matches, &commands.ResourceId{
			DeviceId: o.deviceID,
			Href:     value.Href(),
		})
	}
	return matches
}

// Cancel observation of given resources.
func (o *resourcesObserver) cancelResourcesObservations(ctx context.Context, hrefs []string) {
	observations := o.popTrackedObservations(hrefs)
	for _, obs := range observations {
		if err := obs.Cancel(ctx); err != nil {
			o.logger.Debugf("cannot cancel resource observation: %v", err)
		}
	}
}

func (o *resourcesObserver) popTrackedObservations(hrefs []string) []Observation {
	observations := make([]Observation, 0, 32)
	o.lock.Lock()
	defer o.lock.Unlock()
	newResources, delResources := o.resources.removeByHref(hrefs...)
	if len(delResources) == 0 {
		return nil
	}
	for _, res := range delResources {
		obs := res.PopObservation()
		if obs == nil {
			continue
		}
		o.logger.Debugf("canceling observation on resource(/%v%v)", o.deviceID, res.Href())
		observations = append(observations, obs)
	}
	o.resources = newResources
	return observations
}

// Remove all observations.
func (o *resourcesObserver) CleanObservedResources(ctx context.Context) {
	o.lock.Lock()
	defer o.lock.Unlock()
	o.cleanObservedResourcesLocked(ctx)
}

func (o *resourcesObserver) cleanObservedResourcesLocked(ctx context.Context) {
	observedResources := o.popObservedResourcesLocked()
	for _, obs := range observedResources {
		if v := obs.PopObservation(); v != nil {
			if err := v.Cancel(ctx); err != nil {
				o.logger.Errorf("cannot cancel resource('/%v%v') observation: %w", o.deviceID, obs.Href(), err)
			}
		}
	}
}

func (o *resourcesObserver) popObservedResourcesLocked() observedResources {
	observations := o.resources
	o.resources = make(observedResources, 0, 1)
	return observations
}

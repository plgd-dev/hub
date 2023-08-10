package observation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
)

type (
	OnObserveResource         = func(ctx context.Context, deviceID, resourceHref string, batch bool, notification *pool.Message) error
	OnGetResourceContent      = func(ctx context.Context, deviceID, resourceHref string, notification *pool.Message) error
	UpdateTwinSynchronization = func(ctx context.Context, deviceID string, status commands.TwinSynchronization_State, t time.Time) error
)

type ResourcesObserverCallbacks struct {
	OnObserveResource         OnObserveResource
	OnGetResourceContent      OnGetResourceContent
	UpdateTwinSynchronization UpdateTwinSynchronization
}

func MakeResourcesObserverCallbacks(onObserveResource OnObserveResource, onGetResourceContent OnGetResourceContent, updateTwinSynchronization UpdateTwinSynchronization) ResourcesObserverCallbacks {
	return ResourcesObserverCallbacks{
		OnObserveResource:         onObserveResource,
		OnGetResourceContent:      onGetResourceContent,
		UpdateTwinSynchronization: updateTwinSynchronization,
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
	callbacks ResourcesObserverCallbacks
	coapConn  ClientConn
	logger    log.Logger
	deviceID  string
	private   struct { // guarded by lock
		resources observedResources
		lock      sync.Mutex
	}
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
		deviceID: deviceID,
		private: struct {
			resources observedResources
			lock      sync.Mutex
		}{resources: make(observedResources, 0, 1)},
		coapConn:  coapConn,
		callbacks: callbacks,
		logger:    logger,
	}
}

// Add resource to observer with given interface and wait for initialization message.
func (o *resourcesObserver) addResource(ctx context.Context, res *commands.Resource, obsInterface string, etags [][]byte) error {
	o.private.lock.Lock()
	defer o.private.lock.Unlock()
	obs, err := o.addResourceLocked(res, obsInterface)
	if err == nil && obs != nil {
		o.notifyAboutStartTwinSynchronization(ctx)
		err = o.performObservationLocked([]*observedResourceWithETags{{observedResource: obs, etags: etags}})
		if err != nil {
			o.private.resources, _ = o.private.resources.removeByHref(obs.Href())
			defer o.notifyAboutFinishTwinSynchronization(ctx)
		}
	}
	return err
}

func (o *resourcesObserver) isSynchronizedLocked() bool {
	for _, res := range o.private.resources {
		if !res.synced.Load() {
			return false
		}
	}
	return true
}

func (o *resourcesObserver) resourceHasBeenSynchronized(ctx context.Context, href string) {
	if o.setSynchronizedAtResource(href) {
		o.notifyAboutFinishTwinSynchronization(ctx)
	}
}

// returns true if all resources are observed and synchronized
func (o *resourcesObserver) setSynchronizedAtResource(href string) bool {
	o.private.lock.Lock()
	defer o.private.lock.Unlock()
	i := o.private.resources.search(href)
	synced := false
	if i < len(o.private.resources) && o.private.resources[i].Equals(href) {
		synced = o.private.resources[i].synced.CompareAndSwap(false, true)
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
	if o.private.resources.contains(href) {
		return nil, nil
	}
	obsRes := newObservedResource(href, obsInterface, res.IsObservable())
	o.private.resources = o.private.resources.insert(obsRes)
	return obsRes, nil
}

// Handle given resource.
//
// For observable resources subscribe to observations, for unobservable resources retrieve
// their content.
func (o *resourcesObserver) handleResource(ctx context.Context, obsRes *observedResource, etags [][]byte) error {
	if obsRes.Href() == commands.StatusHref {
		return nil
	}

	if obsRes.isObservable {
		obs, err := o.observeResource(ctx, obsRes, etags)
		if err != nil {
			return err
		}
		obsRes.SetObservation(obs)
		return nil
	}
	var etag []byte
	if len(etags) > 0 {
		etag = etags[0]
	}
	return o.getResourceContent(ctx, obsRes.Href(), etag)
}

// Register to COAP-GW resource observation for given resource
func (o *resourcesObserver) observeResource(ctx context.Context, obsRes *observedResource, etags [][]byte) (Observation, error) {
	cannotObserveResourceError := func(deviceID, href string, err error) error {
		return fmt.Errorf("cannot observe resource /%v%v: %w", deviceID, href, err)
	}
	if o.deviceID == "" {
		return nil, cannotObserveResourceError(o.deviceID, obsRes.Href(), emptyDeviceIDError())
	}

	opts := obsRes.toCoapOptions(etags)
	batchObservation := obsRes.isBatchObservation()
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
func (o *resourcesObserver) getResourceContent(ctx context.Context, href string, etag []byte) error {
	cannotGetResourceError := func(deviceID, href string, err error) error {
		return fmt.Errorf("cannot get resource /%v%v content: %w", deviceID, href, err)
	}
	if o.deviceID == "" {
		return cannotGetResourceError(o.deviceID, href, emptyDeviceIDError())
	}
	opts := make([]message.Option, 0, 1)
	if etag != nil {
		// we use only first etag
		opts = append(opts, message.Option{
			ID:    message.ETag,
			Value: etag,
		})
	}
	resp, err := o.coapConn.Get(ctx, href, opts...)
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

func (o *resourcesObserver) notifyAboutStartTwinSynchronization(ctx context.Context) {
	err := o.callbacks.UpdateTwinSynchronization(ctx, o.deviceID, commands.TwinSynchronization_SYNCING, time.Now())
	if err != nil {
		o.logger.Debugf("cannot update twin synchronization to finish: %v", err)
	}
}

func (o *resourcesObserver) notifyAboutFinishTwinSynchronization(ctx context.Context) {
	err := o.callbacks.UpdateTwinSynchronization(ctx, o.deviceID, commands.TwinSynchronization_IN_SYNC, time.Now())
	if err != nil {
		o.logger.Debugf("cannot update twin synchronization to start: %v", err)
	}
}

type observedResourceWithETags struct {
	*observedResource
	etags [][]byte
}

func (o *resourcesObserver) performObservationLocked(obs []*observedResourceWithETags) error {
	var errors *multierror.Error
	for _, obsRes := range obs {
		if err := o.handleResource(context.Background(), obsRes.observedResource, obsRes.etags); err != nil {
			o.private.resources, _ = o.private.resources.removeByHref(obsRes.Href())
			errors = multierror.Append(errors, err)
		}
	}
	if errors.ErrorOrNil() != nil {
		return fmt.Errorf("cannot perform observation: %w", errors)
	}
	return nil
}

// Add multiple resources to observer.
func (o *resourcesObserver) addResources(ctx context.Context, resources []*commands.Resource) error {
	o.private.lock.Lock()
	observedResources, err := o.addResourcesLocked(resources)
	var errors *multierror.Error
	if err != nil {
		errors = multierror.Append(errors, err)
	}
	if len(observedResources) > 0 {
		o.notifyAboutStartTwinSynchronization(ctx)
		err2 := o.performObservationLocked(observedResources)
		if err2 != nil {
			errors = multierror.Append(errors, err2)
			if o.isSynchronizedLocked() {
				defer o.notifyAboutFinishTwinSynchronization(ctx)
			}
		}
	}
	o.private.lock.Unlock()
	return errors.ErrorOrNil()
}

func (o *resourcesObserver) addResourcesLocked(resources []*commands.Resource) ([]*observedResourceWithETags, error) {
	var errors *multierror.Error
	observedResources := make([]*observedResourceWithETags, 0, len(resources))
	for _, resource := range resources {
		observedResource, err := o.addResourceLocked(resource, "")
		if err != nil {
			errors = multierror.Append(errors, err)
		} else if observedResource != nil {
			observedResources = append(observedResources, &observedResourceWithETags{observedResource: observedResource})
		}
	}
	if errors.ErrorOrNil() != nil {
		return observedResources, fmt.Errorf("cannot add resources to observe: %w", errors)
	}
	return observedResources, nil
}

// Get list of observable and non-observable resources added to resourcesObserver.
func (o *resourcesObserver) getResources() []*commands.ResourceId {
	matches := make([]*commands.ResourceId, 0, 16)
	o.private.lock.Lock()
	defer o.private.lock.Unlock()
	for _, value := range o.private.resources {
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
	o.private.lock.Lock()
	defer o.private.lock.Unlock()
	newResources, delResources := o.private.resources.removeByHref(hrefs...)
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
	o.private.resources = newResources
	return observations
}

// Remove all observations.
func (o *resourcesObserver) CleanObservedResources(ctx context.Context) {
	o.private.lock.Lock()
	defer o.private.lock.Unlock()
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
	observations := o.private.resources
	o.private.resources = make(observedResources, 0, 1)
	return observations
}

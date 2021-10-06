package service

import (
	"fmt"
	"sync"

	"github.com/plgd-dev/cloud/v2/cloud2cloud-connector/store"
	kitSync "github.com/plgd-dev/kit/v2/sync"
)

type ResourceData struct {
	mutex        sync.Mutex
	isSubscribed bool
	subscription Subscription
}

func NewResourceData() *ResourceData {
	return &ResourceData{}
}

func (d *ResourceData) LoadOrCreate(sub Subscription) (Subscription, bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.isSubscribed {
		return d.subscription, true
	}
	d.isSubscribed = true
	d.subscription = sub
	return sub, false
}

func (d *ResourceData) PullOut(sub Subscription) (Subscription, bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if !d.isSubscribed {
		return Subscription{}, false
	}
	if d.subscription.ID != sub.ID {
		return Subscription{}, false
	}
	sub = d.subscription
	d.isSubscribed = false
	d.subscription = Subscription{}
	return sub, true
}

func (d *ResourceData) Subscription() (Subscription, bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.subscription, d.isSubscribed
}

func (d *ResourceData) Dump() interface{} {
	out := make(map[interface{}]interface{})
	if sub, ok := d.Subscription(); ok {
		out["subscription"] = sub
	}
	return out
}

func (d *ResourceData) DumpTasks(linkedCloud store.LinkedCloud, linkedAccount store.LinkedAccount, deviceID, href string) []Task {
	out := make([]Task, 0, 32)
	_, ok := d.Subscription()
	if !ok {
		out = append(out, Task{
			taskType:      TaskType_SubscribeToResource,
			linkedCloud:   linkedCloud,
			linkedAccount: linkedAccount,
			deviceID:      deviceID,
			href:          href,
		})
	}
	return out
}

type DeviceData struct {
	resources *kitSync.Map

	mutex        sync.Mutex
	isSubscribed bool
	subscription Subscription
}

func NewDeviceData() *DeviceData {
	return &DeviceData{
		resources: kitSync.NewMap(),
	}
}

func (d *DeviceData) LoadOrCreate(sub Subscription) (Subscription, bool) {
	if sub.Type == Type_Device {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.isSubscribed {
			return d.subscription, true
		}
		d.isSubscribed = true
		d.subscription = sub
		return sub, false
	}
	resourceI, _ := d.resources.LoadOrStoreWithFunc(sub.Href, nil, func() interface{} {
		return NewResourceData()
	})
	resource := resourceI.(*ResourceData)
	return resource.LoadOrCreate(sub)
}

func (d *DeviceData) PullOut(sub Subscription) (Subscription, bool) {
	if sub.Type == Type_Device {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if !d.isSubscribed {
			return Subscription{}, false
		}
		if d.subscription.ID != sub.ID {
			return Subscription{}, false
		}
		sub = d.subscription
		d.isSubscribed = false
		d.subscription = Subscription{}
		return sub, true
	}
	resourceI, ok := d.resources.Load(sub.Href)
	if !ok {
		return Subscription{}, false
	}
	resource := resourceI.(*ResourceData)
	return resource.PullOut(sub)
}

func (d *DeviceData) Subscription() (Subscription, bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.subscription, d.isSubscribed
}

func (d *DeviceData) Dump() interface{} {
	out := make(map[interface{}]interface{})
	resources := make(map[interface{}]interface{})
	for key, resource := range d.DumpResources() {
		resources[key] = resource.Dump()
	}
	if sub, ok := d.Subscription(); ok {
		out["subscription"] = sub
	}
	if len(resources) > 0 {
		out["resources"] = resources
	}
	return out
}

func (d *DeviceData) DumpResources() map[string]*ResourceData {
	out := make(map[string]*ResourceData)
	d.resources.Range(func(key, resourceI interface{}) bool {
		out[key.(string)] = resourceI.(*ResourceData)
		return true
	})
	return out
}

func (d *DeviceData) DumpTasks(linkedCloud store.LinkedCloud, linkedAccount store.LinkedAccount, deviceID string) []Task {
	out := make([]Task, 0, 32)
	if !linkedCloud.SupportedSubscriptionsEvents.NeedPullDevice() && !linkedCloud.SupportedSubscriptionsEvents.StaticDeviceEvents {
		_, ok := d.Subscription()
		if !ok {
			out = append(out, Task{
				taskType:      TaskType_SubscribeToDevice,
				linkedCloud:   linkedCloud,
				linkedAccount: linkedAccount,
				deviceID:      deviceID,
			})
		}
	}
	if linkedCloud.SupportedSubscriptionsEvents.NeedPullResources() {
		return out
	}
	for href, resource := range d.DumpResources() {
		v := resource.DumpTasks(linkedCloud, linkedAccount, deviceID, href)
		if len(v) > 0 {
			out = append(out, v...)
		}
	}
	return out
}

type LinkedAccountData struct {
	devices *kitSync.Map

	mutex         sync.Mutex
	linkedAccount store.LinkedAccount
	isSubscribed  bool
	subscription  Subscription
}

func NewLinkedAccountData(linkedAccount store.LinkedAccount) *LinkedAccountData {
	return &LinkedAccountData{
		linkedAccount: linkedAccount,
		devices:       kitSync.NewMap(),
	}
}

func (d *LinkedAccountData) LoadOrCreate(sub Subscription) (Subscription, bool) {
	if sub.Type == Type_Devices {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if d.isSubscribed {
			return d.subscription, true
		}
		d.isSubscribed = true
		d.subscription = sub
		return sub, false
	}
	deviceI, _ := d.devices.LoadOrStoreWithFunc(sub.DeviceID, nil, func() interface{} {
		return NewDeviceData()
	})
	device := deviceI.(*DeviceData)
	return device.LoadOrCreate(sub)
}

func (d *LinkedAccountData) PullOut(sub Subscription) (Subscription, bool) {
	if sub.Type == Type_Devices {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		if !d.isSubscribed {
			return Subscription{}, false
		}
		if d.subscription.ID != sub.ID {
			return Subscription{}, false
		}
		sub = d.subscription
		d.isSubscribed = false
		d.subscription = Subscription{}
		return sub, true
	}
	deviceI, ok := d.devices.Load(sub.DeviceID)
	if !ok {
		return Subscription{}, false
	}
	device := deviceI.(*DeviceData)
	return device.PullOut(sub)
}

func (d *LinkedAccountData) Subscription() (Subscription, bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.subscription, d.isSubscribed
}

func (d *LinkedAccountData) LinkedAccount() store.LinkedAccount {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.linkedAccount
}

func (d *LinkedAccountData) Dump() interface{} {
	out := make(map[interface{}]interface{})
	devs := make(map[interface{}]interface{})
	for key, device := range d.DumpDevices() {
		devs[key] = device.Dump()
	}
	if sub, ok := d.Subscription(); ok {
		out["subscription"] = sub
	}
	if len(devs) > 0 {
		out["devices"] = devs
	}
	out["account"] = d.LinkedAccount()
	return out
}

func (d *LinkedAccountData) DumpDevices() map[string]*DeviceData {
	out := make(map[string]*DeviceData)
	d.devices.Range(func(key, deviceI interface{}) bool {
		out[key.(string)] = deviceI.(*DeviceData)
		return true
	})
	return out
}

func (d *LinkedAccountData) DumpTasks(linkedCloud store.LinkedCloud) []Task {
	out := make([]Task, 0, 32)
	linkedAccount := d.LinkedAccount()
	if !linkedCloud.SupportedSubscriptionsEvents.NeedPullDevices() {
		_, ok := d.Subscription()
		if !ok {
			out = append(out, Task{
				taskType:      TaskType_SubscribeToDevices,
				linkedCloud:   linkedCloud,
				linkedAccount: linkedAccount,
			})
		}
	}
	if linkedCloud.SupportedSubscriptionsEvents.NeedPullDevice() && linkedCloud.SupportedSubscriptionsEvents.NeedPullResources() {
		return out
	}
	for deviceID, device := range d.DumpDevices() {
		v := device.DumpTasks(linkedCloud, linkedAccount, deviceID)
		if len(v) > 0 {
			out = append(out, v...)
		}
	}
	return out
}

type CloudData struct {
	linkedAccounts *kitSync.Map
	linkedCloud    store.LinkedCloud
}

func NewCloudData(linkedCloud store.LinkedCloud) *CloudData {
	return &CloudData{
		linkedCloud:    linkedCloud,
		linkedAccounts: kitSync.NewMap(),
	}
}

func (d *CloudData) Dump() interface{} {
	out := make(map[interface{}]interface{})
	accs := make(map[interface{}]interface{})
	for key, linkedAccount := range d.DumpLinkedAccounts() {
		accs[key] = linkedAccount.Dump()
	}
	if len(accs) > 0 {
		out["accounts"] = accs
	}
	out["cloud"] = d.linkedCloud
	return out
}

func (d *CloudData) DumpLinkedAccounts() map[string]*LinkedAccountData {
	out := make(map[string]*LinkedAccountData)
	d.linkedAccounts.Range(func(key, linkedAccountI interface{}) bool {
		out[key.(string)] = linkedAccountI.(*LinkedAccountData)
		return true
	})
	return out
}

func (d *CloudData) DumpTasks() []Task {
	out := make([]Task, 0, 32)
	if d.linkedCloud.SupportedSubscriptionsEvents.NeedPullDevices() && d.linkedCloud.SupportedSubscriptionsEvents.NeedPullDevice() && d.linkedCloud.SupportedSubscriptionsEvents.NeedPullResources() {
		return out
	}
	for _, linkedAccount := range d.DumpLinkedAccounts() {
		v := linkedAccount.DumpTasks(d.linkedCloud)
		if len(v) > 0 {
			out = append(out, v...)
		}
	}
	return out
}

type Cache struct {
	clouds            *kitSync.Map
	subscriptionsByID *kitSync.Map
}

func NewCache() *Cache {
	return &Cache{
		clouds:            kitSync.NewMap(),
		subscriptionsByID: kitSync.NewMap(),
	}
}

func (s *Cache) loadResource(cloudID, linkedAccountID, deviceID, href string) (*CloudData, *LinkedAccountData, *DeviceData, *ResourceData, error) {
	cloud, linkedAccount, device, err := s.loadDevice(cloudID, linkedAccountID, deviceID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	resourceI, ok := device.resources.Load(href)
	if !ok {
		return nil, nil, nil, nil, fmt.Errorf("deviceID %v not found", deviceID)
	}
	resource := resourceI.(*ResourceData)
	return cloud, linkedAccount, device, resource, nil
}

func (s *Cache) loadDevice(cloudID, linkedAccountID, deviceID string) (*CloudData, *LinkedAccountData, *DeviceData, error) {
	cloud, linkedAccount, err := s.loadLinkedAccount(cloudID, linkedAccountID)
	if err != nil {
		return nil, nil, nil, err
	}
	deviceI, ok := linkedAccount.devices.Load(deviceID)
	if !ok {
		return nil, nil, nil, fmt.Errorf("deviceID %v not found", deviceID)
	}
	device := deviceI.(*DeviceData)
	return cloud, linkedAccount, device, nil
}

func (s *Cache) loadLinkedAccount(linkedCloudID, linkedAccountID string) (*CloudData, *LinkedAccountData, error) {
	cloudI, ok := s.clouds.Load(linkedCloudID)
	if !ok {
		return nil, nil, fmt.Errorf("cloudID %v not found", linkedCloudID)
	}
	cloud := cloudI.(*CloudData)
	linkedAccountI, ok := cloud.linkedAccounts.Load(linkedAccountID)
	if !ok {
		return nil, nil, fmt.Errorf("linkedAccountID %v not found", linkedAccountID)
	}
	return cloud, linkedAccountI.(*LinkedAccountData), nil
}

func (s *Cache) LoadSubscription(ID string) (subscriptionData, bool) {
	subI, ok := s.subscriptionsByID.Load(ID)
	if !ok {
		return subscriptionData{}, ok
	}
	sub := subI.(Subscription)
	cloud, linkedAccount, err := s.loadLinkedAccount(sub.LinkedCloudID, sub.LinkedAccountID)
	if err != nil {
		s.subscriptionsByID.Delete(sub.ID)
		return subscriptionData{}, ok
	}
	return subscriptionData{
		linkedAccount: linkedAccount.LinkedAccount(),
		linkedCloud:   cloud.linkedCloud,
		subscription:  sub,
	}, true
}

func (s *Cache) LoadDevicesSubscription(cloudID, linkedAccountID string) (subscriptionData, bool) {
	cloud, linkedAccount, err := s.loadLinkedAccount(cloudID, linkedAccountID)
	if err != nil {
		return subscriptionData{}, false
	}
	sub, ok := linkedAccount.Subscription()
	return subscriptionData{
		linkedAccount: linkedAccount.LinkedAccount(),
		linkedCloud:   cloud.linkedCloud,
		subscription:  sub,
	}, ok
}

func (s *Cache) LoadDeviceSubscription(cloudID, linkedAccountID, deviceID string) (subscriptionData, bool) {
	cloud, linkedAccount, device, err := s.loadDevice(cloudID, linkedAccountID, deviceID)
	if err != nil {
		return subscriptionData{}, false
	}
	sub, ok := device.Subscription()
	return subscriptionData{
		linkedAccount: linkedAccount.LinkedAccount(),
		linkedCloud:   cloud.linkedCloud,
		subscription:  sub,
	}, ok
}

func (s *Cache) LoadResourceSubscription(cloudID, linkedAccountID, deviceID, href string) (subscriptionData, bool) {
	cloud, linkedAccount, _, resource, err := s.loadResource(cloudID, linkedAccountID, deviceID, href)
	if err != nil {
		return subscriptionData{}, false
	}
	sub, ok := resource.Subscription()
	return subscriptionData{
		linkedAccount: linkedAccount.LinkedAccount(),
		linkedCloud:   cloud.linkedCloud,
		subscription:  sub,
	}, ok
}

func (s *Cache) LoadOrCreateCloud(cloud store.LinkedCloud) (store.LinkedCloud, bool) {
	cloudI, ok := s.clouds.LoadOrStoreWithFunc(cloud.ID, nil, func() interface{} {
		return NewCloudData(cloud)
	})
	return cloudI.(*CloudData).linkedCloud, ok
}

func (s *Cache) LoadCloud(cloudID string) (store.LinkedCloud, bool) {
	cloudI, ok := s.clouds.Load(cloudID)
	return cloudI.(*CloudData).linkedCloud, ok
}

func (s *Cache) LoadOrCreateLinkedAccount(linkedAccount store.LinkedAccount) (store.LinkedAccount, bool, error) {
	cloudI, ok := s.clouds.Load(linkedAccount.LinkedCloudID)
	if !ok {
		return store.LinkedAccount{}, false, fmt.Errorf("cloudID %v not found", linkedAccount.LinkedCloudID)
	}
	cloud := cloudI.(*CloudData)
	linkedAccountI, ok := cloud.linkedAccounts.LoadOrStoreWithFunc(linkedAccount.ID, nil, func() interface{} {
		return NewLinkedAccountData(linkedAccount)
	})
	return linkedAccountI.(*LinkedAccountData).linkedAccount, ok, nil
}

func (s *Cache) UpdateLinkedAccount(l store.LinkedAccount) error {
	_, linkedAccount, err := s.loadLinkedAccount(l.LinkedCloudID, l.ID)
	if err != nil {
		return err
	}
	linkedAccount.mutex.Lock()
	defer linkedAccount.mutex.Unlock()
	linkedAccount.linkedAccount = l
	return nil
}

func (s *Cache) LoadOrCreateSubscription(sub Subscription) (subscriptionData, bool, error) {
	subData, ok := s.LoadSubscription(sub.ID)
	if ok {
		return subData, ok, nil
	}
	cloud, linkedAccount, err := s.loadLinkedAccount(sub.LinkedCloudID, sub.LinkedAccountID)
	if err != nil {
		return subscriptionData{}, false, err
	}
	sub, loaded := linkedAccount.LoadOrCreate(sub)
	if !loaded {
		s.subscriptionsByID.Replace(sub.ID, sub)
	}
	return subscriptionData{
		linkedAccount: linkedAccount.LinkedAccount(),
		linkedCloud:   cloud.linkedCloud,
		subscription:  sub,
	}, loaded, nil
}

func (s *Cache) PullOutSubscription(subscripionID string) (subscriptionData, bool) {
	subI, ok := s.subscriptionsByID.PullOut(subscripionID)
	if !ok {
		return subscriptionData{}, ok
	}
	sub := subI.(Subscription)
	cloud, linkedAccount, err := s.loadLinkedAccount(sub.LinkedCloudID, sub.LinkedAccountID)
	if err != nil {
		return subscriptionData{}, false
	}
	sub, ok = linkedAccount.PullOut(sub)
	if !ok {
		return subscriptionData{}, ok
	}
	return subscriptionData{
		linkedAccount: linkedAccount.LinkedAccount(),
		linkedCloud:   cloud.linkedCloud,
		subscription:  sub,
	}, true
}

func cleanUpResourceSubcription(s *Cache, resource *ResourceData) {
	if resource.isSubscribed {
		s.subscriptionsByID.Delete(resource.subscription.ID)
	}
}

func cleanUpLinkedAccountsSubcriptions(s *Cache, linkedAccount *LinkedAccountData) {
	if linkedAccount.isSubscribed {
		s.subscriptionsByID.Delete(linkedAccount.subscription.ID)
	}
	for _, device := range linkedAccount.DumpDevices() {
		cleanUpDeviceSubcriptions(s, device)
	}
}

func cleanUpDeviceSubcriptions(s *Cache, device *DeviceData) {
	if device.isSubscribed {
		s.subscriptionsByID.Delete(device.subscription.ID)
	}
	for _, resource := range device.DumpResources() {
		cleanUpResourceSubcription(s, resource)
	}
}

func (s *Cache) PullOutCloud(cloudID string) (*CloudData, bool) {
	cloudI, ok := s.clouds.PullOut(cloudID)
	if !ok {
		return nil, ok
	}
	cloud := cloudI.(*CloudData)
	for _, linkedAccount := range cloud.DumpLinkedAccounts() {
		cleanUpLinkedAccountsSubcriptions(s, linkedAccount)
	}
	return cloud, true
}

func (s *Cache) PullOutLinkedAccount(cloudID, linkedAccountID string) (*LinkedAccountData, bool) {
	cloudI, ok := s.clouds.Load(cloudID)
	if !ok {
		return nil, ok
	}
	cloud := cloudI.(*CloudData)
	linkedAccountI, ok := cloud.linkedAccounts.PullOut(linkedAccountID)
	if !ok {
		return nil, ok
	}
	linkedAccount := linkedAccountI.(*LinkedAccountData)
	cleanUpLinkedAccountsSubcriptions(s, linkedAccount)
	return linkedAccount, true
}

func (s *Cache) PullOutDevice(cloudID, linkedAccountID, deviceID string) (*DeviceData, bool) {
	_, linkedAccount, err := s.loadLinkedAccount(cloudID, linkedAccountID)
	if err != nil {
		return nil, false
	}
	deviceI, ok := linkedAccount.devices.PullOut(deviceID)
	if !ok {
		return nil, ok
	}
	device := deviceI.(*DeviceData)
	cleanUpDeviceSubcriptions(s, device)
	return device, true
}

func (s *Cache) PullOutResource(cloudID, linkedAccountID, deviceID, href string) (*ResourceData, bool) {
	_, _, device, err := s.loadDevice(cloudID, linkedAccountID, deviceID)
	if err != nil {
		return nil, false
	}
	resourceI, ok := device.resources.PullOut(href)
	if !ok {
		return nil, ok
	}
	resource := resourceI.(*ResourceData)
	cleanUpResourceSubcription(s, resource)
	return resource, true
}

func (s *Cache) DumpClouds() map[string]*CloudData {
	out := make(map[string]*CloudData)
	s.clouds.Range(func(key, cloudI interface{}) bool {
		out[key.(string)] = cloudI.(*CloudData)
		return true
	})
	return out
}

func (s *Cache) Dump() interface{} {
	out := make(map[interface{}]interface{})
	for key, cloud := range s.DumpClouds() {
		out[key] = cloud.Dump()
	}
	return out
}

func (s *Cache) DumpLinkedAccounts() []provisionCacheData {
	out := make([]provisionCacheData, 0, 32)
	for _, cloud := range s.DumpClouds() {
		for _, linkedAccount := range cloud.DumpLinkedAccounts() {
			out = append(out, provisionCacheData{
				linkedCloud:   cloud.linkedCloud,
				linkedAccount: linkedAccount.LinkedAccount(),
			})
		}
	}
	return out
}

func (s *Cache) DumpDevices() []subscriptionData {
	out := make([]subscriptionData, 0, 32)
	for _, cloud := range s.DumpClouds() {
		for _, linkedAccount := range cloud.DumpLinkedAccounts() {
			for _, device := range linkedAccount.DumpDevices() {
				sub, ok := device.Subscription()
				if !ok {
					continue
				}
				out = append(out, subscriptionData{
					linkedCloud:   cloud.linkedCloud,
					linkedAccount: linkedAccount.LinkedAccount(),
					subscription:  sub,
				})
			}
		}
	}
	return out
}

func (s *Cache) DumpTasks() []Task {
	out := make([]Task, 0, 32)
	for _, cloud := range s.DumpClouds() {
		v := cloud.DumpTasks()
		if len(v) > 0 {
			out = append(out, v...)
		}
	}
	return out
}

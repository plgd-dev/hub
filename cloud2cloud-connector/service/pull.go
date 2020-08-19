package service

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/plgd-dev/kit/log"

	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/cloud2cloud-connector/store"
	pbCQRS "github.com/plgd-dev/cloud/resource-aggregate/pb"
	pbRA "github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/codec/json"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/plgd-dev/sdk/schema"
)

type Device struct {
	Device schema.Device `json:"device"`
	Status string        `json:"status"`
}

type RetrieveDeviceWithLinksResponse struct {
	Device
	Links []schema.ResourceLink `json:"links"`
}

type pullDevicesHandler struct {
	s                   *Store
	asClient            pbAS.AuthorizationServiceClient
	raClient            pbRA.ResourceAggregateClient
	devicesSubscription *DevicesSubscription
	linkedClouds        map[string]store.LinkedCloud
	subscriptionManager *SubscriptionManager
	oauthCallback       string
	triggerTask         func(Task)
}

func getUsersDevices(ctx context.Context, asClient pbAS.AuthorizationServiceClient) (map[string]bool, error) {
	getUserDevicesClient, err := asClient.GetUserDevices(ctx, &pbAS.GetUserDevicesRequest{})
	if err != nil {
		return nil, fmt.Errorf("cannot get users devices: %w", err)
	}
	defer getUserDevicesClient.CloseSend()
	userDevices := make(map[string]bool)
	for {
		userDevice, err := getUserDevicesClient.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("cannot get users devices: %w", err)
		}
		if userDevice == nil {
			continue
		}

		userDevices[userDevice.DeviceId] = true
	}
	return userDevices, nil
}

func Get(ctx context.Context, url string, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, v interface{}) error {
	client := linkedCloud.GetHTTPClient()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+string(linkedAccount.TargetCloud.AccessToken))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Connection", "close")
	req.Close = true

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = json.Decode(buf, v)
	if err != nil {
		return fmt.Errorf("cannot decode body(%v): %w", string(buf), err)
	}
	return nil
}

func publishDeviceResources(ctx context.Context, raClient pbRA.ResourceAggregateClient, deviceID string, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, subscriptionManager *SubscriptionManager, dev RetrieveDeviceWithLinksResponse, triggerTask func(Task)) error {
	var errors []error
	userID := linkedAccount.UserID
	for _, link := range dev.Links {
		link.DeviceID = deviceID
		href := removeDeviceIDFromHref(link.Href)
		link.Href = href
		err := publishResource(ctx, raClient, userID, link, pbCQRS.CommandMetadata{
			ConnectionId: linkedAccount.ID,
			Sequence:     uint64(time.Now().UnixNano()),
		})
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot publish respource %+v: %w", link, err))
			continue
		}
		if linkedCloud.SupportedSubscriptionsEvents.NeedPullResources() {
			continue
		}
		triggerTask(Task{
			taskType:      TaskType_SubscribeToResource,
			linkedAccount: linkedAccount,
			linkedCloud:   linkedCloud,
			deviceID:      deviceID,
			href:          href,
		})
	}
	if len(errors) > 0 {
		return fmt.Errorf("%+v", errors)
	}
	return nil
}

func (p *pullDevicesHandler) getDevicesWithResourceLinks(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	var errors []error
	userID := linkedAccount.UserID

	var devices []RetrieveDeviceWithLinksResponse
	err := Get(ctx, linkedCloud.Endpoint.URL+"/devices", linkedAccount, linkedCloud, &devices)
	if err != nil {
		return err
	}

	ctx = kitNetGrpc.CtxWithUserID(ctx, userID)
	if linkedCloud.SupportedSubscriptionsEvents.NeedPullDevices() {
		registeredDevices, err := getUsersDevices(ctx, p.asClient)
		if err != nil {
			return err
		}

		for _, dev := range devices {
			deviceID := dev.Device.Device.ID
			ok := registeredDevices[deviceID]
			err := p.devicesSubscription.Add(deviceID, linkedAccount, linkedCloud)
			if err != nil {
				errors = append(errors, fmt.Errorf("cannot add device %v to devicesSubscription: %w", deviceID, err))
			}

			if !ok {
				_, err := p.asClient.AddDevice(ctx, &pbAS.AddDeviceRequest{
					DeviceId: deviceID,
					UserId:   userID,
				})
				if err != nil {
					errors = append(errors, fmt.Errorf("cannot addDevice %v: %w", deviceID, err))
					continue
				}

				err = publishCloudDeviceStatus(ctx, p.raClient, userID, deviceID, pbCQRS.CommandMetadata{
					ConnectionId: linkedAccount.ID,
					Sequence:     uint64(time.Now().UnixNano()),
				})
				if err != nil {
					errors = append(errors, fmt.Errorf("cannot publish cloud status: %v: %w", deviceID, err))
					continue
				}

			}
			delete(registeredDevices, deviceID)
			var online bool
			if strings.ToLower(dev.Status) == "online" {
				online = true
			}

			err = updateCloudStatus(ctx, p.raClient, userID, deviceID, online, pbCQRS.CommandMetadata{
				ConnectionId: linkedAccount.ID,
				Sequence:     uint64(time.Now().UnixNano()),
			})
			if err != nil {
				errors = append(errors, fmt.Errorf("cannot update cloud status: %v: %w", deviceID, err))
			}
		}
		for deviceID := range registeredDevices {
			err := p.devicesSubscription.Delete(userID, deviceID)
			if err != nil {
				errors = append(errors, fmt.Errorf("cannot delete device %v from devicesSubscription: %w", deviceID, err))
			}
			_, err = p.asClient.RemoveDevice(ctx, &pbAS.RemoveDeviceRequest{
				DeviceId: deviceID,
			})
			if err != nil {
				errors = append(errors, fmt.Errorf("cannot removeDevice %v: %w", deviceID, err))
			}
		}
	}

	for _, dev := range devices {
		deviceID := dev.Device.Device.ID
		if linkedCloud.SupportedSubscriptionsEvents.StaticDeviceEvents {
			p.triggerTask(Task{
				taskType:      TaskType_PullDevice,
				linkedAccount: linkedAccount,
				linkedCloud:   linkedCloud,
				deviceID:      deviceID,
			})
		} else if linkedCloud.SupportedSubscriptionsEvents.NeedPullDevice() {
			err := publishDeviceResources(ctx, p.raClient, deviceID, linkedAccount, linkedCloud, p.subscriptionManager, dev, p.triggerTask)
			if err != nil {
				errors = append(errors, err)
				continue
			}
		} else {
			if _, ok := p.s.LoadDeviceSubscription(linkedCloud.ID, linkedAccount.ID, deviceID); ok {
				continue
			}
			p.triggerTask(Task{
				taskType:      TaskType_SubscribeToDevice,
				linkedAccount: linkedAccount,
				linkedCloud:   linkedCloud,
				deviceID:      deviceID,
			})
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%+v", errors)
	}
	return nil
}

type Representation struct {
	Href           string      `json:"href"`
	Representation interface{} `json:"rep"`
}

type RetrieveDeviceContentAllResponse struct {
	Device
	Links []Representation `json:"links"`
}

func removeDeviceIDFromHref(href string) string {
	hrefsp := strings.Split(href, "/")
	href = "/" + strings.Join(hrefsp[2:], "/")
	return href
}

func (p *pullDevicesHandler) getDevicesWithResourceValues(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	var errors []error
	userID := linkedAccount.UserID

	var devices []RetrieveDeviceContentAllResponse
	err := Get(ctx, linkedCloud.Endpoint.URL+"/devices?content=all", linkedAccount, linkedCloud, &devices)
	if err != nil {
		return err
	}

	ctx = kitNetGrpc.CtxWithUserID(ctx, userID)
	for _, dev := range devices {
		deviceID := dev.Device.Device.ID
		for _, link := range dev.Links {
			link.Href = removeDeviceIDFromHref(link.Href)
			body, err := json.Encode(link.Representation)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			err = notifyResourceChanged(
				ctx,
				p.raClient,
				deviceID,
				link.Href,
				userID,
				"application/json",
				body,
				pbCQRS.CommandMetadata{
					ConnectionId: linkedAccount.ID,
					Sequence:     uint64(time.Now().UnixNano()),
				},
			)
			log.Debugf("notifyResourceChanged %v%v: %v", deviceID, link.Href, string(body))
			if err != nil {
				errors = append(errors, fmt.Errorf("cannot notifyResourceChanged %+v: %w", link, err))
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%+v", errors)
	}

	return nil
}

func RefreshToken(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, oauthCallback string, s *Store) (store.LinkedAccount, error) {
	ctx = linkedCloud.CtxWithHTTPClient(ctx)
	oauthCfg := linkedCloud.OAuth
	if oauthCfg.RedirectURL == "" {
		oauthCfg.RedirectURL = oauthCallback
	}
	token, refreshed, err := linkedAccount.TargetCloud.Refresh(ctx, linkedCloud.OAuth.ToOAuth2())
	if err != nil {
		return store.LinkedAccount{}, err
	}
	linkedAccount.TargetCloud = token
	if refreshed {
		err = s.UpdateLinkedAccount(ctx, linkedAccount)
		if err != nil {
			return store.LinkedAccount{}, fmt.Errorf("cannot store updated linked linkedAccount: %w", err)
		}
	}
	return linkedAccount, nil
}

func (p *pullDevicesHandler) pullDevicesFromAccount(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	linkedAccount, err := RefreshToken(ctx, linkedAccount, linkedCloud, p.oauthCallback, p.s)
	if err != nil {
		return err
	}
	var errors []error
	if linkedCloud.SupportedSubscriptionsEvents.NeedPullDevices() || linkedCloud.SupportedSubscriptionsEvents.NeedPullDevice() {
		err = p.getDevicesWithResourceLinks(ctx, linkedAccount, linkedCloud)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if linkedCloud.SupportedSubscriptionsEvents.NeedPullResources() {
		err = p.getDevicesWithResourceValues(ctx, linkedAccount, linkedCloud)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%+v", errors)
	}
	return nil
}

func pullDevices(ctx context.Context, s *Store,
	asClient pbAS.AuthorizationServiceClient,
	raClient pbRA.ResourceAggregateClient,
	devicesSubscription *DevicesSubscription,
	subscriptionManager *SubscriptionManager,
	oauthCallback string,
	triggerTask func(Task)) error {

	data := s.DumpLinkedAccounts()
	var wg sync.WaitGroup
	for _, d := range data {
		log.Debugf("pulling devices for %v", d.linkedAccount)
		p := pullDevicesHandler{
			s:                   s,
			asClient:            asClient,
			raClient:            raClient,
			devicesSubscription: devicesSubscription,
			subscriptionManager: subscriptionManager,
			oauthCallback:       oauthCallback,
			triggerTask:         triggerTask,
		}
		wg.Add(1)
		go func(linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) {
			defer wg.Done()
			err := p.pullDevicesFromAccount(ctx, linkedAccount, linkedCloud)
			if err != nil {
				log.Errorf("cannot pull devices for linked linkedAccount(%v): %v", linkedAccount, err)
			}
		}(d.linkedAccount, d.linkedCloud)
	}

	wg.Wait()
	return nil
}

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

	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/cloud2cloud-connector/store"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	raService "github.com/plgd-dev/cloud/resource-aggregate/service"
	"github.com/plgd-dev/kit/codec/json"
	"github.com/plgd-dev/kit/log"
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
	raClient            raService.ResourceAggregateClient
	devicesSubscription *DevicesSubscription
	subscriptionManager *SubscriptionManager
	oauthCallback       string
	triggerTask         OnTaskTrigger
}

func getUsersDevices(ctx context.Context, asClient pbAS.AuthorizationServiceClient) (map[string]bool, error) {
	getUserDevicesClient, err := asClient.GetUserDevices(ctx, &pbAS.GetUserDevicesRequest{})
	if err != nil {
		return nil, fmt.Errorf("cannot get users devices: %w", err)
	}
	defer func() {
		if err := getUserDevicesClient.CloseSend(); err != nil {
			log.Errorf("failed to close user devices client: %v", err)
		}
	}()
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
	defer client.CloseIdleConnections()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+string(linkedAccount.Data.TargetCloud.AccessToken))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Connection", "close")
	req.Close = true

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Errorf("failed to close response body: %v", err)
		}
	}()
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

func publishDeviceResources(ctx context.Context, raClient raService.ResourceAggregateClient, deviceID string, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, subscriptionManager *SubscriptionManager, dev RetrieveDeviceWithLinksResponse, triggerTask OnTaskTrigger) error {
	var errors []error
	userID := linkedAccount.UserID
	ctx = kitNetGrpc.CtxWithToken(ctx, linkedAccount.Data.OriginCloud.AccessToken.String())
	for _, link := range dev.Links {
		link.DeviceID = deviceID
		href := removeDeviceIDFromHref(link.Href)
		link.Href = href
		err := publishResource(ctx, raClient, userID, link, &commands.CommandMetadata{
			ConnectionId: linkedAccount.ID,
			Sequence:     uint64(time.Now().UnixNano()),
		})
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot publish resource %+v: %w", link, err))
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
	var devices []RetrieveDeviceWithLinksResponse
	err := Get(ctx, linkedCloud.Endpoint.URL+"/devices", linkedAccount, linkedCloud, &devices)
	if err != nil {
		return err
	}

	var errors []error
	userID := linkedAccount.UserID
	ctx = kitNetGrpc.CtxWithToken(ctx, linkedAccount.Data.OriginCloud.AccessToken.String())
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
			}
			delete(registeredDevices, deviceID)
			if strings.ToLower(dev.Status) == "online" {
				_, err = p.raClient.UpdateDeviceMetadata(ctx, &commands.UpdateDeviceMetadataRequest{
					DeviceId: deviceID,
					Update: &commands.UpdateDeviceMetadataRequest_Status{
						Status: &commands.ConnectionStatus{
							Value: commands.ConnectionStatus_ONLINE,
						},
					},
					CommandMetadata: &commands.CommandMetadata{
						ConnectionId: linkedAccount.ID,
						Sequence:     uint64(time.Now().UnixNano()),
					},
				})
			} else {
				_, err = p.raClient.UpdateDeviceMetadata(ctx, &commands.UpdateDeviceMetadataRequest{
					DeviceId: deviceID,
					Update: &commands.UpdateDeviceMetadataRequest_Status{
						Status: &commands.ConnectionStatus{
							Value: commands.ConnectionStatus_OFFLINE,
						},
					},
					CommandMetadata: &commands.CommandMetadata{
						ConnectionId: linkedAccount.ID,
						Sequence:     uint64(time.Now().UnixNano()),
					},
				})
			}
			if err != nil {
				errors = append(errors, fmt.Errorf("cannot update cloud status: %v: %w", deviceID, err))
			}
		}
		for deviceID := range registeredDevices {
			err := p.devicesSubscription.Delete(userID, deviceID)
			if err != nil {
				errors = append(errors, fmt.Errorf("cannot delete device %v from devicesSubscription: %w", deviceID, err))
			}
			resp, err := p.asClient.DeleteDevices(ctx, &pbAS.DeleteDevicesRequest{
				DeviceIds: []string{deviceID},
			})
			if err != nil {
				errors = append(errors, fmt.Errorf("cannot delete device %v: %w", deviceID, err))
			}
			if err == nil && len(resp.DeviceIds) != 1 {
				errors = append(errors, fmt.Errorf("cannot remove device %v", deviceID))
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
	var devices []RetrieveDeviceContentAllResponse
	err := Get(ctx, linkedCloud.Endpoint.URL+"/devices?content=all", linkedAccount, linkedCloud, &devices)
	if err != nil {
		return err
	}

	var errors []error
	userID := linkedAccount.UserID
	ctx = kitNetGrpc.CtxWithToken(ctx, linkedAccount.Data.OriginCloud.AccessToken.String())
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
				&commands.CommandMetadata{
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
	// TODO: refresh both origin and target
	ctx = linkedCloud.CtxWithHTTPClient(ctx)
	oauthCfg := linkedCloud.OAuth
	if oauthCfg.RedirectURL == "" {
		oauthCfg.RedirectURL = oauthCallback
	}
	token, refreshed, err := linkedAccount.Data.TargetCloud.Refresh(ctx, linkedCloud.OAuth.ToOAuth2("", ""))
	if err != nil {
		return store.LinkedAccount{}, err
	}
	linkedAccount.Data.TargetCloud = token
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
	raClient raService.ResourceAggregateClient,
	devicesSubscription *DevicesSubscription,
	subscriptionManager *SubscriptionManager,
	oauthCallback string,
	triggerTask OnTaskTrigger) error {

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
				log.Errorf("cannot pull devices for linked linkedAccount(%v): %w", linkedAccount, err)
			}
		}(d.linkedAccount, d.linkedCloud)
	}

	wg.Wait()
	return nil
}

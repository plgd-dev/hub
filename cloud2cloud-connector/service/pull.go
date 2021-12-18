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

	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/cloud2cloud-connector/store"
	pbIS "github.com/plgd-dev/hub/identity-store/pb"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	kitHttp "github.com/plgd-dev/hub/pkg/net/http"
	"github.com/plgd-dev/hub/pkg/security/oauth2"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	raService "github.com/plgd-dev/hub/resource-aggregate/service"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/plgd-dev/kit/v2/log"
)

type Device struct {
	Device device.Device `json:"device"`
	Status string        `json:"status"`
}

type RetrieveDeviceWithLinksResponse struct {
	Device
	Links []schema.ResourceLink `json:"links"`
}

type pullDevicesHandler struct {
	s                   *Store
	isClient            pbIS.IdentityStoreClient
	raClient            raService.ResourceAggregateClient
	devicesSubscription *DevicesSubscription
	subscriptionManager *SubscriptionManager
	provider            *oauth2.PlgdProvider
	triggerTask         OnTaskTrigger
}

func getOwnerDevices(ctx context.Context, isClient pbIS.IdentityStoreClient) (map[string]bool, error) {
	getDevicesClient, err := isClient.GetDevices(ctx, &pbIS.GetDevicesRequest{})
	if err != nil {
		return nil, fmt.Errorf("cannot get owned devices: %w", err)
	}
	defer func() {
		if err := getDevicesClient.CloseSend(); err != nil {
			log.Errorf("failed to close user devices client: %v", err)
		}
	}()
	ownerDevices := make(map[string]bool)
	for {
		device, err := getDevicesClient.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("cannot get owned devices: %w", err)
		}
		if device == nil {
			continue
		}

		ownerDevices[device.DeviceId] = true
	}
	return ownerDevices, nil
}

func Get(ctx context.Context, url string, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, v interface{}) error {
	client := linkedCloud.GetHTTPClient()
	defer client.CloseIdleConnections()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+string(linkedAccount.Data.Target().AccessToken))
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

func publishDeviceResources(ctx context.Context, raClient raService.ResourceAggregateClient, deviceID string, linkedAccount store.LinkedAccount,
	linkedCloud store.LinkedCloud, dev RetrieveDeviceWithLinksResponse, triggerTask OnTaskTrigger) error {
	var errors []error
	ctx = kitNetGrpc.CtxWithToken(ctx, linkedAccount.Data.Origin().AccessToken.String())
	for _, link := range dev.Links {
		link.DeviceID = deviceID
		href := removeDeviceIDFromHref(link.Href)
		link.Href = href
		err := publishResource(ctx, raClient, link, &commands.CommandMetadata{
			ConnectionId: linkedAccount.ID,
			Sequence:     uint64(time.Now().UnixNano()),
		})
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot publish resource %+v: %w", link, err))
			continue
		}
		if linkedCloud.SupportedSubscriptionEvents.NeedPullResources() {
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

func toConnectionStatus(status string) commands.ConnectionStatus_Status {
	if strings.ToLower(status) == "online" {
		return commands.ConnectionStatus_ONLINE
	}
	return commands.ConnectionStatus_OFFLINE
}

func (p *pullDevicesHandler) triggerTaskForDevice(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, dev RetrieveDeviceWithLinksResponse) error {
	deviceID := dev.Device.Device.ID
	if linkedCloud.SupportedSubscriptionEvents.StaticDeviceEvents {
		p.triggerTask(Task{
			taskType:      TaskType_PullDevice,
			linkedAccount: linkedAccount,
			linkedCloud:   linkedCloud,
			deviceID:      deviceID,
		})
		return nil
	}

	if linkedCloud.SupportedSubscriptionEvents.NeedPullDevice() {
		return publishDeviceResources(ctx, p.raClient, deviceID, linkedAccount, linkedCloud, dev, p.triggerTask)
	}

	if _, ok := p.s.LoadDeviceSubscription(linkedCloud.ID, linkedAccount.ID, deviceID); ok {
		return nil
	}
	p.triggerTask(Task{
		taskType:      TaskType_SubscribeToDevice,
		linkedAccount: linkedAccount,
		linkedCloud:   linkedCloud,
		deviceID:      deviceID,
	})
	return nil
}

func (p *pullDevicesHandler) deleteDevice(ctx context.Context, userID, deviceID string) []error {
	var errors []error
	err := p.devicesSubscription.Delete(userID, deviceID)
	if err != nil {
		errors = append(errors, fmt.Errorf("cannot delete device %v from devicesSubscription: %w", deviceID, err))
	}
	resp, err := p.isClient.DeleteDevices(ctx, &pbIS.DeleteDevicesRequest{
		DeviceIds: []string{deviceID},
	})
	if err != nil {
		errors = append(errors, fmt.Errorf("cannot delete device %v: %w", deviceID, err))
	}
	if err == nil && len(resp.DeviceIds) != 1 {
		errors = append(errors, fmt.Errorf("cannot remove device %v", deviceID))
	}
	return errors
}

func (p *pullDevicesHandler) pullDevices(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, devices []RetrieveDeviceWithLinksResponse) (bool, []error) {
	var errors []error
	registeredDevices, err := getOwnerDevices(ctx, p.isClient)
	if err != nil {
		return false, []error{err}
	}

	for _, dev := range devices {
		deviceID := dev.Device.Device.ID
		err := p.devicesSubscription.Add(deviceID, linkedAccount, linkedCloud)
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot add device %v to devicesSubscription: %w", deviceID, err))
		}

		ok := registeredDevices[deviceID]
		if !ok {
			_, err := p.isClient.AddDevice(ctx, &pbIS.AddDeviceRequest{
				DeviceId: deviceID,
			})
			if err != nil {
				errors = append(errors, fmt.Errorf("cannot addDevice %v: %w", deviceID, err))
				continue
			}
		}
		delete(registeredDevices, deviceID)
		_, err = p.raClient.UpdateDeviceMetadata(ctx, &commands.UpdateDeviceMetadataRequest{
			DeviceId: deviceID,
			Update: &commands.UpdateDeviceMetadataRequest_Status{
				Status: &commands.ConnectionStatus{
					Value: toConnectionStatus(dev.Status),
				},
			},
			CommandMetadata: &commands.CommandMetadata{
				ConnectionId: linkedAccount.ID,
				Sequence:     uint64(time.Now().UnixNano()),
			},
		})
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot update cloud status: %v: %w", deviceID, err))
		}
	}

	userID := linkedAccount.UserID
	for deviceID := range registeredDevices {
		if err := p.deleteDevice(ctx, deviceID, userID); err != nil {
			errors = append(errors, err...)
		}
	}
	return true, errors
}

func (p *pullDevicesHandler) getDevicesWithResourceLinks(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	var devices []RetrieveDeviceWithLinksResponse
	err := Get(ctx, linkedCloud.Endpoint.URL+"/devices", linkedAccount, linkedCloud, &devices)
	if err != nil {
		return err
	}

	var errors []error
	ctx = kitNetGrpc.CtxWithToken(ctx, linkedAccount.Data.Origin().AccessToken.String())
	if linkedCloud.SupportedSubscriptionEvents.NeedPullDevices() {
		ok, pullErrors := p.pullDevices(ctx, linkedAccount, linkedCloud, devices)
		if !ok {
			return fmt.Errorf("%+v", errors)
		}
		errors = append(errors, pullErrors...)
	}

	for _, dev := range devices {
		if err := p.triggerTaskForDevice(ctx, linkedAccount, linkedCloud, dev); err != nil {
			errors = append(errors, err)
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
func (p *pullDevicesHandler) notifyResourceChanged(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, deviceID string, link Representation) error {
	link.Href = removeDeviceIDFromHref(link.Href)
	body, err := json.Encode(link.Representation)
	if err != nil {
		return err
	}

	_, err = p.raClient.NotifyResourceChanged(ctx, &commands.NotifyResourceChangedRequest{
		ResourceId: commands.NewResourceID(deviceID, kitHttp.CanonicalHref(link.Href)),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: linkedAccount.ID,
			Sequence:     uint64(time.Now().UnixNano()),
		},
		Content: &commands.Content{
			Data:              body,
			ContentType:       message.AppJSON.String(),
			CoapContentFormat: int32(message.AppJSON),
		},
	})
	log.Debugf("notifyResourceChanged %v%v: %v", deviceID, link.Href, string(body))
	if err != nil {
		return fmt.Errorf("cannot notifyResourceChanged %+v: %w", link, err)
	}
	return nil
}

func (p *pullDevicesHandler) getDevicesWithResourceValues(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	var devices []RetrieveDeviceContentAllResponse
	err := Get(ctx, linkedCloud.Endpoint.URL+"/devices?content=all", linkedAccount, linkedCloud, &devices)
	if err != nil {
		return err
	}

	var errors []error
	ctx = kitNetGrpc.CtxWithToken(ctx, linkedAccount.Data.Origin().AccessToken.String())
	for _, dev := range devices {
		deviceID := dev.Device.Device.ID
		for _, link := range dev.Links {
			if err := p.notifyResourceChanged(ctx, linkedAccount, linkedCloud, deviceID, link); err != nil {
				errors = append(errors, err)
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%+v", errors)
	}

	return nil
}

func refreshTokens(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, provider *oauth2.PlgdProvider, s *Store) (store.LinkedAccount, error) {
	if !linkedAccount.Data.Origin().IsValidAccessToken() {
		originToken, err := provider.Refresh(ctx, linkedAccount.Data.Origin().RefreshToken)
		if err != nil {
			return store.LinkedAccount{}, fmt.Errorf("cannot refresh origin cloud token: %w", err)
		}
		linkedAccount.Data = linkedAccount.Data.SetOrigin(*originToken)
	}

	ctx = linkedCloud.CtxWithHTTPClient(ctx)
	oauthCfg := linkedCloud.OAuth
	if oauthCfg.RedirectURL == "" {
		oauthCfg.RedirectURL = provider.Config.RedirectURL
	}
	targetToken, targetRefreshed, err := linkedAccount.Data.Target().Refresh(ctx, linkedCloud.OAuth.ToDefaultOAuth2())
	if err != nil {
		return store.LinkedAccount{}, fmt.Errorf("cannot refresh target cloud token: %w", err)
	}

	linkedAccount.Data = linkedAccount.Data.SetTarget(targetToken)
	if targetRefreshed {
		if err = s.UpdateLinkedAccount(ctx, linkedAccount); err != nil {
			return store.LinkedAccount{}, fmt.Errorf("cannot store updated linked linkedAccount: %w", err)
		}
	}
	return linkedAccount, nil
}

func (p *pullDevicesHandler) pullDevicesFromAccount(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	linkedAccount, err := refreshTokens(ctx, linkedAccount, linkedCloud, p.provider, p.s)
	if err != nil {
		return err
	}
	var errors []error
	if linkedCloud.SupportedSubscriptionEvents.NeedPullDevices() || linkedCloud.SupportedSubscriptionEvents.NeedPullDevice() {
		err = p.getDevicesWithResourceLinks(ctx, linkedAccount, linkedCloud)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if linkedCloud.SupportedSubscriptionEvents.NeedPullResources() {
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

func (p *pullDevicesHandler) pullLinkedAccountsDevices(ctx context.Context) error {
	data := p.s.DumpLinkedAccounts()
	var wg sync.WaitGroup
	for _, d := range data {
		log.Debugf("pulling devices for %v", d.linkedAccount)
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

func (p *pullDevicesHandler) runDevicePulling(ctx context.Context, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	err := p.pullLinkedAccountsDevices(ctx)
	if err != nil {
		log.Errorf("cannot pull devices: %v", err)
	}
	<-ctx.Done()
	return ctx.Err() != context.Canceled
}

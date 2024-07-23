package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/store"
	pbIS "github.com/plgd-dev/hub/v2/identity-store/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttpUri "github.com/plgd-dev/hub/v2/pkg/net/http/uri"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	raService "github.com/plgd-dev/hub/v2/resource-aggregate/service"
	"github.com/plgd-dev/kit/v2/codec/json"
	"go.opentelemetry.io/otel/trace"
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
	tracerProvider      trace.TracerProvider
}

func getOwnerDevices(ctx context.Context, isClient pbIS.IdentityStoreClient) (map[string]bool, error) {
	getDevicesClient, err := isClient.GetDevices(ctx, &pbIS.GetDevicesRequest{})
	if err != nil {
		return nil, fmt.Errorf("cannot get owned devices: %w", err)
	}
	defer func() {
		if err := getDevicesClient.CloseSend(); err != nil {
			log.Errorf("failed to close user devices client: %w", err)
		}
	}()
	ownerDevices := make(map[string]bool)
	for {
		device, err := getDevicesClient.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("cannot get owned devices: %w", err)
		}
		if device == nil {
			continue
		}

		ownerDevices[device.GetDeviceId()] = true
	}
	return ownerDevices, nil
}

func Get(ctx context.Context, tracerProvider trace.TracerProvider, url string, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, v interface{}) error {
	client := linkedCloud.GetHTTPClient(tracerProvider)
	defer client.CloseIdleConnections()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set(AuthorizationHeader, AuthorizationBearerPrefix+string(linkedAccount.Data.Target().AccessToken))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Connection", "close")
	req.Close = true

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if errC := resp.Body.Close(); errC != nil {
			log.Errorf("failed to close response body: %w", errC)
		}
	}()
	buf, err := io.ReadAll(resp.Body)
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
	linkedCloud store.LinkedCloud, dev RetrieveDeviceWithLinksResponse, triggerTask OnTaskTrigger,
) error {
	var errors *multierror.Error
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
			errors = multierror.Append(errors, fmt.Errorf("cannot publish resource %+v: %w", link, err))
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
	return errors.ErrorOrNil()
}

func toConnectionStatus(status string) commands.Connection_Status {
	if strings.ToLower(status) == "online" {
		return commands.Connection_ONLINE
	}
	return commands.Connection_OFFLINE
}

func toConnectionProtocol(status string) commands.Connection_Protocol {
	if strings.ToLower(status) == "online" {
		return commands.Connection_C2C
	}
	return commands.Connection_UNKNOWN
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

func (p *pullDevicesHandler) deleteDevice(ctx context.Context, userID, deviceID string) error {
	var errors *multierror.Error
	err := p.devicesSubscription.Delete(userID, deviceID)
	if err != nil {
		errors = multierror.Append(errors, fmt.Errorf("cannot delete device %v from devicesSubscription: %w", deviceID, err))
	}
	resp, err := p.isClient.DeleteDevices(ctx, &pbIS.DeleteDevicesRequest{
		DeviceIds: []string{deviceID},
	})
	if err != nil {
		errors = multierror.Append(errors, fmt.Errorf("cannot delete device %v: %w", deviceID, err))
	}
	if err == nil && len(resp.GetDeviceIds()) != 1 {
		errors = multierror.Append(errors, fmt.Errorf("cannot remove device %v", deviceID))
	}
	return errors.ErrorOrNil()
}

func (p *pullDevicesHandler) pullDevices(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, devices []RetrieveDeviceWithLinksResponse) (bool, *multierror.Error) {
	var errors *multierror.Error
	registeredDevices, err := getOwnerDevices(ctx, p.isClient)
	if err != nil {
		return false, multierror.Append(err)
	}

	for _, dev := range devices {
		deviceID := dev.Device.Device.ID
		err := p.devicesSubscription.Add(ctx, deviceID, linkedAccount, linkedCloud)
		if err != nil {
			errors = multierror.Append(errors, fmt.Errorf("cannot add device %v to devicesSubscription: %w", deviceID, err))
		}

		ok := registeredDevices[deviceID]
		if !ok {
			_, err = p.isClient.AddDevice(ctx, &pbIS.AddDeviceRequest{
				DeviceId: deviceID,
			})
			if err != nil {
				errors = multierror.Append(errors, fmt.Errorf("cannot addDevice %v: %w", deviceID, err))
				continue
			}
		}
		delete(registeredDevices, deviceID)
		_, err = p.raClient.UpdateDeviceMetadata(ctx, &commands.UpdateDeviceMetadataRequest{
			DeviceId: deviceID,
			Update: &commands.UpdateDeviceMetadataRequest_Connection{
				Connection: &commands.Connection{
					Status:   toConnectionStatus(dev.Status),
					Protocol: toConnectionProtocol(dev.Status),
				},
			},
			CommandMetadata: &commands.CommandMetadata{
				ConnectionId: linkedAccount.ID,
				Sequence:     uint64(time.Now().UnixNano()),
			},
		})
		if err != nil {
			errors = multierror.Append(errors, fmt.Errorf("cannot update cloud status: %v: %w", deviceID, err))
		}
	}

	userID := linkedAccount.UserID
	for deviceID := range registeredDevices {
		if err := p.deleteDevice(ctx, deviceID, userID); err != nil {
			errors = multierror.Append(errors, err)
		}
	}
	return true, errors
}

func (p *pullDevicesHandler) getDevicesWithResourceLinks(ctx context.Context, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	var devices []RetrieveDeviceWithLinksResponse
	err := Get(ctx, p.tracerProvider, linkedCloud.Endpoint.URL+"/devices", linkedAccount, linkedCloud, &devices)
	if err != nil {
		return err
	}

	var errors *multierror.Error
	ctx = kitNetGrpc.CtxWithToken(ctx, linkedAccount.Data.Origin().AccessToken.String())
	if linkedCloud.SupportedSubscriptionEvents.NeedPullDevices() {
		ok, pullErrors := p.pullDevices(ctx, linkedAccount, linkedCloud, devices)
		if !ok {
			return errors.ErrorOrNil()
		}
		errors = multierror.Append(errors, pullErrors)
	}

	for _, dev := range devices {
		if err := p.triggerTaskForDevice(ctx, linkedAccount, linkedCloud, dev); err != nil {
			errors = multierror.Append(errors, err)
		}
	}
	return errors.ErrorOrNil()
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

func (p *pullDevicesHandler) notifyResourceChanged(ctx context.Context, linkedAccount store.LinkedAccount, deviceID string, link Representation) error {
	link.Href = removeDeviceIDFromHref(link.Href)
	body, err := json.Encode(link.Representation)
	if err != nil {
		return err
	}

	_, err = p.raClient.NotifyResourceChanged(ctx, &commands.NotifyResourceChangedRequest{
		ResourceId: commands.NewResourceID(deviceID, pkgHttpUri.CanonicalHref(link.Href)),
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
	err := Get(ctx, p.tracerProvider, linkedCloud.Endpoint.URL+"/devices?content=all", linkedAccount, linkedCloud, &devices)
	if err != nil {
		return err
	}

	var errors *multierror.Error
	ctx = kitNetGrpc.CtxWithToken(ctx, linkedAccount.Data.Origin().AccessToken.String())
	for _, dev := range devices {
		deviceID := dev.Device.Device.ID
		for _, link := range dev.Links {
			if err := p.notifyResourceChanged(ctx, linkedAccount, deviceID, link); err != nil {
				errors = multierror.Append(errors, err)
			}
		}
	}
	return errors.ErrorOrNil()
}

func refreshTokens(ctx context.Context, traceProvider trace.TracerProvider, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud, provider *oauth2.PlgdProvider, s *Store) (store.LinkedAccount, error) {
	if !linkedAccount.Data.Origin().IsValidAccessToken() {
		originToken, err := provider.Refresh(ctx, linkedAccount.Data.Origin().RefreshToken)
		if err != nil {
			return store.LinkedAccount{}, fmt.Errorf("cannot refresh origin cloud token: %w", err)
		}
		linkedAccount.Data = linkedAccount.Data.SetOrigin(*originToken)
	}

	ctx = linkedCloud.CtxWithHTTPClient(ctx, traceProvider)
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
	linkedAccount, err := refreshTokens(ctx, p.tracerProvider, linkedAccount, linkedCloud, p.provider, p.s)
	if err != nil {
		return err
	}
	var errors *multierror.Error
	if linkedCloud.SupportedSubscriptionEvents.NeedPullDevices() || linkedCloud.SupportedSubscriptionEvents.NeedPullDevice() {
		err = p.getDevicesWithResourceLinks(ctx, linkedAccount, linkedCloud)
		if err != nil {
			errors = multierror.Append(errors, err)
		}
	}
	if linkedCloud.SupportedSubscriptionEvents.NeedPullResources() {
		err = p.getDevicesWithResourceValues(ctx, linkedAccount, linkedCloud)
		if err != nil {
			errors = multierror.Append(errors, err)
		}
	}
	return errors.ErrorOrNil()
}

func (p *pullDevicesHandler) pullLinkedAccountsDevices(ctx context.Context) {
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
}

func (p *pullDevicesHandler) runDevicePulling(ctx context.Context, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	p.pullLinkedAccountsDevices(ctx)
	<-ctx.Done()
	return !errors.Is(ctx.Err(), context.Canceled)
}

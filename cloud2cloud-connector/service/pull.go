package service

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/go-ocf/kit/log"

	"github.com/gofrs/uuid"

	pbAS "github.com/go-ocf/cloud/authorization/pb"
	"github.com/go-ocf/cloud/cloud2cloud-connector/store"
	pbCQRS "github.com/go-ocf/cloud/resource-aggregate/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	"github.com/go-ocf/kit/codec/json"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/sdk/schema"
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
	s                   store.Store
	asClient            pbAS.AuthorizationServiceClient
	raClient            pbRA.ResourceAggregateClient
	devicesSubscription *DevicesSubscription
	linkedClouds        map[string]store.LinkedCloud
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

func (p *pullDevicesHandler) getDevicesWithResourceLinks(ctx context.Context, account store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	var errors []error
	connectionIDRand, err := uuid.NewV4()
	if err != nil {
		return err
	}
	userID := account.UserID
	connectionID := "c2c-connector-pull:" + userID + "/devices:" + connectionIDRand.String()
	client := linkedCloud.GetHTTPClient()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, account.TargetURL+"/devices", nil)
	req.Header.Set("Authorization", "Bearer "+string(account.TargetCloud.AccessToken))
	req.Header.Set("Accept", "application/json")
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var devices []RetrieveDeviceWithLinksResponse
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = json.Decode(buf, &devices)
	if err != nil {
		return fmt.Errorf("cannot decode body(%v): %w", string(buf), err)
	}

	ctx = kitNetGrpc.CtxWithToken(ctx, userID)
	registeredDevices, err := getUsersDevices(ctx, p.asClient)
	if err != nil {
		return err
	}

	for _, dev := range devices {
		deviceID := dev.Device.Device.ID
		ok := registeredDevices[deviceID]
		err := p.devicesSubscription.Add(ctx, deviceID, account, linkedCloud)
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
				ConnectionId: connectionID,
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
			ConnectionId: connectionID,
		})
		if err != nil {
			errors = append(errors, fmt.Errorf("cannot update cloud status: %v: %w", deviceID, err))
			continue
		}
		for _, link := range dev.Links {
			link.DeviceID = deviceID
			link.Href = removeDeviceIDFromHref(link.Href)
			err := publishResource(ctx, p.raClient, userID, link, pbCQRS.CommandMetadata{
				ConnectionId: connectionID,
			})
			if err != nil {
				errors = append(errors, fmt.Errorf("cannot update cloud status: %+v: %w", link, err))
				continue
			}
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
			continue
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

func (p *pullDevicesHandler) getDevicesWithResourceValues(ctx context.Context, account store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	var errors []error
	connectionIDRand, err := uuid.NewV4()
	if err != nil {
		return err
	}
	userID := account.UserID
	connectionID := "c2c-connector-pull:" + userID + "/devices?content=all:" + connectionIDRand.String()
	client := linkedCloud.GetHTTPClient()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, account.TargetURL+"/devices?content=all", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+string(account.TargetCloud.AccessToken))
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var devices []RetrieveDeviceContentAllResponse
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = json.Decode(buf, &devices)
	if err != nil {
		return fmt.Errorf("cannot decode body(%v): %w", string(buf), err)
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
					ConnectionId: connectionID,
				},
			)
			log.Debugf("notifyResourceChanged %v%v: %v", deviceID, link.Href, string(body))
			if err != nil {
				errors = append(errors, fmt.Errorf("cannot notifyResourceChanged %+v: %w", link, err))
				continue
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%+v", errors)
	}

	return nil
}

func RefreshToken(ctx context.Context, account store.LinkedAccount, linkedCloud store.LinkedCloud, s store.Store) (store.LinkedAccount, error) {
	ctx = linkedCloud.CtxWithHTTPClient(ctx)
	token, refreshed, err := account.TargetCloud.Refresh(ctx, linkedCloud.OAuth.ToOAuth2())
	if err != nil {
		return store.LinkedAccount{}, err
	}
	account.TargetCloud = token
	if refreshed {
		err = s.UpdateLinkedAccount(ctx, account)
		if err != nil {
			return store.LinkedAccount{}, fmt.Errorf("cannot store updated linked account: %v", err)
		}
	}
	return account, nil
}

func (p *pullDevicesHandler) pullDevicesFromAccount(ctx context.Context, account store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	account, err := RefreshToken(ctx, account, linkedCloud, p.s)
	if err != nil {
		return err
	}
	if linkedCloud.SupportedSubscriptionsEvents.NeedPullDevices() {
		err = p.getDevicesWithResourceLinks(ctx, account, linkedCloud)
		if err != nil {
			return err
		}
	}
	if linkedCloud.SupportedSubscriptionsEvents.NeedPullDevicesWithContent() {
		err = p.getDevicesWithResourceValues(ctx, account, linkedCloud)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *pullDevicesHandler) Handle(ctx context.Context, iter store.LinkedAccountIter) error {
	var wg sync.WaitGroup
	for {
		var account store.LinkedAccount
		if !iter.Next(ctx, &account) {
			break
		}
		log.Debugf("pulling devices for %v", account)
		wg.Add(1)
		go func() {
			defer wg.Done()
			linkedCloud, ok := p.linkedClouds[account.LinkedCloudID]
			if !ok {
				log.Errorf("cannot find linked cloud %v for linked account %v", account.LinkedCloudID, account)
			}
			err := p.pullDevicesFromAccount(ctx, account, linkedCloud)
			if err != nil {
				log.Errorf("cannot pull devices for linked account(%v): %v", account, err)
			}
		}()
	}
	wg.Wait()
	return iter.Err()
}

func pullDevices(ctx context.Context, s store.Store,
	asClient pbAS.AuthorizationServiceClient,
	raClient pbRA.ResourceAggregateClient,
	devicesSubscription *DevicesSubscription) error {

	var lh LinkedCloudsHandler
	err := s.LoadLinkedClouds(ctx, store.Query{}, &lh)
	if err != nil {
		return fmt.Errorf("cannot load linked clouds: %v", err)
	}

	h := pullDevicesHandler{
		s:                   s,
		asClient:            asClient,
		raClient:            raClient,
		devicesSubscription: devicesSubscription,
		linkedClouds:        lh.linkedClouds,
	}
	return s.LoadLinkedAccounts(ctx, store.Query{}, &h)
}

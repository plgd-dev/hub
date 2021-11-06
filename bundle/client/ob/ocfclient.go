package main

import (
	"context"
	"fmt"
	"time"

	"github.com/plgd-dev/device/app"
	"github.com/plgd-dev/device/client"
	"github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/device/schema/acl"
	capb "github.com/plgd-dev/hub/certificate-authority/pb"
	"github.com/plgd-dev/hub/certificate-authority/signer"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	"github.com/plgd-dev/hub/pkg/log"
	"github.com/plgd-dev/kit/v2/security"
)

type OcfClient struct {
	localClient      *client.Client
	hubConfiguration *pb.HubConfigurationResponse
}

// Initialize creates and initializes new local client
func (c *OcfClient) Initialize(ctx context.Context, hubConfiguration *pb.HubConfigurationResponse, caClient capb.CertificateAuthorityClient) error {
	appCallback, err := app.NewApp(&app.AppConfig{
		RootCA: hubConfiguration.GetCertificateAuthorities(),
	})
	if err != nil {
		return fmt.Errorf("cannot create app callback: %w", err)
	}

	signer := signer.NewIdentityCertificateSigner(caClient)

	localClient, err := client.NewClientFromConfig(&client.Config{
		KeepAliveConnectionTimeoutSeconds: 30,
		ObserverPollingIntervalSeconds:    15,
		DeviceCacheExpirationSeconds:      3600,
		MaxMessageSize:                    512 * 1024,
		DeviceOwnershipBackend: &client.DeviceOwnershipBackendConfig{
			JWTClaimOwnerID: hubConfiguration.GetJwtOwnerClaim(),
			Sign:            signer.Sign,
		},
	}, appCallback, nil, func(err error) {
		log.Error(err)
	})

	if err != nil {
		return fmt.Errorf("cannot create client: %w", err)
	}

	err = localClient.Initialization(ctx)
	if err != nil {
		return fmt.Errorf("cannot initialize client: %w", err)
	}

	c.localClient = localClient
	c.hubConfiguration = hubConfiguration
	return nil
}

func (c *OcfClient) GetOwnerID() (string, error) {
	return c.localClient.CoreClient().GetSdkOwnerID()
}

// Discover devices in the local area
func (c *OcfClient) Discover(ctx context.Context, timeout time.Duration) (map[string]client.DeviceDetails, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return c.localClient.GetDevices(ctx)
}

// OwnDevice transfers the ownersip of the device to user represented by the token
func (c *OcfClient) OwnDevice(ctx context.Context, deviceID string) (string, error) {
	return c.localClient.OwnDevice(ctx, deviceID, client.WithOTM(client.OTMType_JustWorks))
}

// SetAccessForCloud sets required ACL for the Cloud
func (c *OcfClient) SetAccessForCloud(ctx context.Context, deviceID string) error {
	d, links, err := c.localClient.GetRefDevice(ctx, deviceID)
	if err != nil {
		return err
	}

	defer func() {
		_ = d.Release(ctx)
	}()
	p, err := d.Provision(ctx, links)
	if err != nil {
		return err
	}
	defer func() {
		_ = p.Close(ctx)
	}()

	link, err := core.GetResourceLink(links, acl.ResourceURI)
	if err != nil {
		return err
	}

	setACL := acl.UpdateRequest{
		AccessControlList: []acl.AccessControl{
			{
				Permission: acl.AllPermissions,
				Subject: acl.Subject{
					Subject_Device: &acl.Subject_Device{
						DeviceID: c.hubConfiguration.GetId(),
					},
				},
				Resources: acl.AllResources,
			},
		},
	}

	err = p.UpdateResource(ctx, link, setACL, nil)
	if err != nil {
		return err
	}
	caCert := []byte(c.hubConfiguration.GetCertificateAuthorities())
	certs, err := security.ParseX509FromPEM(caCert)
	if err != nil {
		return err
	}
	return p.AddCertificateAuthority(ctx, c.hubConfiguration.GetId(), certs[0])
}

// OnboardDevice registers the device to the plgd cloud
func (c *OcfClient) OnboardDevice(ctx context.Context, deviceID, apn, authCode string) error {
	cloudURL := c.hubConfiguration.GetCoapGateway()
	cloudID := c.hubConfiguration.GetId()
	return c.localClient.OnboardDevice(ctx, deviceID, apn, cloudURL, authCode, cloudID)
}

// DisownDevice removes the current ownership
func (c *OcfClient) DisownDevice(ctx context.Context, deviceID string) error {
	return c.localClient.DisownDevice(ctx, deviceID)
}

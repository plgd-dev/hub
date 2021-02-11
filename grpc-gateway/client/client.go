package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/plgd-dev/kit/strings"
)

type ApplicationCallback interface {
	GetRootCertificateAuthorities() ([]*x509.Certificate, error)
}

type OnboardOption = func() interface{}
type subscription = interface {
	Cancel() (wait func(), err error)
}

type Config struct {
	GatewayAddress string
	AccessTokenURL string
}

func validateURL(URL string) error {
	if URL == "" {
		return fmt.Errorf("empty url")
	}
	_, err := url.Parse(URL)
	if err != nil {
		return err
	}
	return nil
}

// NewClient constructs a new client client. For every call there is expected jwt token for grpc stored in context.
func NewClient(accessTokenURL string, client pb.GrpcGatewayClient) (*Client, error) {
	err := validateURL(accessTokenURL)
	if err != nil {
		return nil, fmt.Errorf("invalid AccessTokenURL: %w", err)
	}

	cl := Client{
		gateway:        client,
		subscriptions:  make(map[string]subscription),
		accessTokenURL: accessTokenURL,
	}
	return &cl, nil
}

// NewClientFromConfig constructs a new client client. For every call there is expected jwt token for grpc stored in context.
func NewClientFromConfig(cfg *Config, tlsCfg *tls.Config) (*Client, error) {
	if cfg == nil || cfg.GatewayAddress == "" {
		return nil, fmt.Errorf("missing client client config")
	}

	keepAlive := keepalive.ClientParameters{
		Time:                10 * time.Second,
		PermitWithoutStream: true,
	}

	conn, err := grpc.Dial(cfg.GatewayAddress, grpc.WithKeepaliveParams(keepAlive), grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
	if err != nil {
		return nil, fmt.Errorf("cannot create certificate authority client: %w", err)
	}

	ocfGW := pb.NewGrpcGatewayClient(conn)

	client, err := NewClient(cfg.AccessTokenURL, ocfGW)
	if err != nil {
		conn.Close()
		return nil, err
	}
	client.conn = conn

	return client, nil
}

// Client for interacting with the client.
type Client struct {
	gateway pb.GrpcGatewayClient
	conn    *grpc.ClientConn

	accessTokenURL string

	subscriptionsLock sync.Mutex
	subscriptions     map[string]subscription
}

func (c *Client) GrpcGatewayClient() pb.GrpcGatewayClient {
	return c.gateway
}

func (c *Client) popSubscriptions() map[string]subscription {
	c.subscriptionsLock.Lock()
	defer c.subscriptionsLock.Unlock()
	s := c.subscriptions
	c.subscriptions = make(map[string]subscription)
	return s
}

func (c *Client) popSubscription(ID string) (subscription, error) {
	c.subscriptionsLock.Lock()
	defer c.subscriptionsLock.Unlock()
	v, ok := c.subscriptions[ID]
	if !ok {
		return nil, fmt.Errorf("cannot find observation %v", ID)
	}
	delete(c.subscriptions, ID)
	return v, nil
}

func (c *Client) insertSubscription(ID string, s subscription) {
	c.subscriptionsLock.Lock()
	defer c.subscriptionsLock.Unlock()
	c.subscriptions[ID] = s
}

func (c *Client) Close(ctx context.Context) error {
	var errors []error
	for _, s := range c.popSubscriptions() {
		wait, err := s.Cancel()
		if err != nil {
			errors = append(errors, err)
			continue
		}
		wait()
	}
	if c.conn != nil {
		err := c.conn.Close()
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) != 0 {
		return fmt.Errorf("%v", errors)
	}

	return nil
}

func (c *Client) GetAccessTokenURL(ctx context.Context) (string, error) {
	return c.accessTokenURL, nil
}

// GetDevicesViaCallback returns devices. JWT token must be stored in context for grpc call.
func (c *Client) GetDevicesViaCallback(ctx context.Context, deviceIDs, resourceTypes []string, callback func(pb.Device)) error {
	it := c.GetDevicesIterator(ctx, deviceIDs, resourceTypes...)
	defer it.Close()

	for {
		var v pb.Device
		if !it.Next(&v) {
			break
		}
		callback(v)
	}
	return it.Err
}

// GetResourceLinksViaCallback returns resource links of devices. JWT token must be stored in context for grpc call.
func (c *Client) GetResourceLinksViaCallback(ctx context.Context, deviceIDs, resourceTypes []string, callback func(*pb.ResourceLink)) error {
	it := c.GetResourceLinksIterator(ctx, deviceIDs, resourceTypes...)
	defer it.Close()
	for {
		var v pb.ResourceLink
		if !it.Next(&v) {
			break
		}
		callback(&v)
	}
	return it.Err
}

type TypeCallback struct {
	Type     string
	Callback func(pb.ResourceValue)
}

func MakeTypeCallback(resourceType string, callback func(pb.ResourceValue)) TypeCallback {
	return TypeCallback{Type: resourceType, Callback: callback}
}

// RetrieveResourcesByType gets contents of resources by resource types. JWT token must be stored in context for grpc call.
func (c *Client) RetrieveResourcesByType(
	ctx context.Context,
	deviceIDs []string,
	typeCallbacks ...TypeCallback,
) error {
	tc := make(map[string]func(pb.ResourceValue), len(typeCallbacks))
	resourceTypes := make([]string, 0, len(typeCallbacks))
	for _, c := range typeCallbacks {
		tc[c.Type] = c.Callback
		resourceTypes = append(resourceTypes, c.Type)
	}

	it := c.RetrieveResourcesIterator(ctx, nil, deviceIDs, resourceTypes...)
	defer it.Close()

	for {
		var v pb.ResourceValue
		if !it.Next(&v) {
			break
		}
		for _, rt := range resourceTypes {
			if strings.SliceContains(v.Types, rt) {
				tc[rt](v)
				break
			}
		}
	}
	return it.Err
}

// GetDevicesIterator gets devices. JWT token must be stored in context for grpc call.
// Next queries the next resource value.
// Returns false when failed or having no more items.
// Check it.Err for errors.
// Usage:
//	for {
//		var v MyStruct
//		if !it.Next(ctx, &v) {
//			break
//		}
//	}
//	if it.Err != nil {
//	}
func (c *Client) GetDevicesIterator(ctx context.Context, deviceIDs []string, resourceTypes ...string) *kitNetGrpc.Iterator {
	r := pb.GetDevicesRequest{DeviceIdsFilter: deviceIDs, TypeFilter: resourceTypes}
	return kitNetGrpc.NewIterator(c.gateway.GetDevices(ctx, &r))
}

// GetResourceLinksIterator gets devices. JWT token must be stored in context for grpc call.
// Next queries the next resource value.
// Returns false when failed or having no more items.
// Check it.Err for errors.
// Usage:
//	for {
//		var v MyStruct
//		if !it.Next(ctx, &v) {
//			break
//		}
//	}
//	if it.Err != nil {
//	}
func (c *Client) GetResourceLinksIterator(ctx context.Context, deviceIDs []string, resourceTypes ...string) *kitNetGrpc.Iterator {
	r := pb.GetResourceLinksRequest{DeviceIdsFilter: deviceIDs, TypeFilter: resourceTypes}
	return kitNetGrpc.NewIterator(c.gateway.GetResourceLinks(ctx, &r))
}

// RetrieveResourcesIterator gets resources contents. JWT token must be stored in context for grpc call.
// Next queries the next resource value.
// Returns false when failed or having no more items.
// Check it.Err for errors.
// Usage:
//	for {
//		var v MyStruct
//		if !it.Next(ctx, &v) {
//			break
//		}
//	}
//	if it.Err != nil {
//	}
func (c *Client) RetrieveResourcesIterator(ctx context.Context, resourceIDs []*pb.ResourceId, deviceIDs []string, resourceTypes ...string) *kitNetGrpc.Iterator {
	r := pb.RetrieveResourcesValuesRequest{ResourceIdsFilter: resourceIDs, DeviceIdsFilter: deviceIDs, TypeFilter: resourceTypes}
	return kitNetGrpc.NewIterator(c.gateway.RetrieveResourcesValues(ctx, &r))
}

type ResourceIDCallback struct {
	ResourceID *pb.ResourceId
	Callback   func(pb.ResourceValue)
}

func MakeResourceIDCallback(deviceID, href string, callback func(pb.ResourceValue)) ResourceIDCallback {
	return ResourceIDCallback{ResourceID: &pb.ResourceId{
		DeviceId: deviceID,
		Href:     href,
	}, Callback: callback}
}

// RetrieveResourcesByResourceIDs gets resources contents by resourceIDs. JWT token must be stored in context for grpc call.
func (c *Client) RetrieveResourcesByResourceIDs(
	ctx context.Context,
	resourceIDsCallbacks ...ResourceIDCallback,
) error {
	tc := make(map[string]func(pb.ResourceValue), len(resourceIDsCallbacks))
	resourceIDs := make([]*pb.ResourceId, 0, len(resourceIDsCallbacks))
	for _, c := range resourceIDsCallbacks {
		tc[utils.MakeResourceId(c.ResourceID.DeviceId, c.ResourceID.Href)] = c.Callback
		resourceIDs = append(resourceIDs, c.ResourceID)
	}

	it := c.RetrieveResourcesIterator(ctx, resourceIDs, nil)
	defer it.Close()
	var v pb.ResourceValue
	for {
		if !it.Next(&v) {
			break
		}
		c, ok := tc[utils.MakeResourceId(v.GetResourceId().GetDeviceId(), v.GetResourceId().GetHref())]
		if ok {
			c(v)
		}
	}
	return it.Err
}

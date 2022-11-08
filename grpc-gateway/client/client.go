package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

type ApplicationCallback interface {
	GetRootCertificateAuthorities() ([]*x509.Certificate, error)
}

type (
	OnboardOption = func() interface{}
	subscription  = interface {
		Cancel() (wait func(), err error)
	}
)

type Config struct {
	GatewayAddress string
}

// NewClient constructs a new client client. For every call there is expected jwt token for grpc stored in context.
func New(client pb.GrpcGatewayClient) *Client {
	return &Client{
		gateway:       client,
		subscriptions: make(map[string]subscription),
	}
}

// NewFromConfig constructs a new client client. For every call there is expected jwt token for grpc stored in context.
func NewFromConfig(cfg *Config, tlsCfg *tls.Config) (*Client, error) {
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

	client := New(ocfGW)

	client.conn = conn

	return client, nil
}

// Client for interacting with the client.
type Client struct {
	gateway pb.GrpcGatewayClient
	conn    *grpc.ClientConn

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

func (c *Client) popSubscription(id string) (subscription, error) {
	c.subscriptionsLock.Lock()
	defer c.subscriptionsLock.Unlock()
	v, ok := c.subscriptions[id]
	if !ok {
		return nil, fmt.Errorf("cannot find observation %v", id)
	}
	delete(c.subscriptions, id)
	return v, nil
}

func (c *Client) insertSubscription(id string, s subscription) {
	c.subscriptionsLock.Lock()
	defer c.subscriptionsLock.Unlock()
	c.subscriptions[id] = s
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

// GetDevicesViaCallback returns devices. JWT token must be stored in context for grpc call.
func (c *Client) GetDevicesViaCallback(ctx context.Context, deviceIDs, resourceTypes []string, callback func(*pb.Device)) error {
	it := c.GetDevicesIterator(ctx, deviceIDs, resourceTypes...)
	defer it.Close()

	for {
		var v pb.Device
		if !it.Next(&v) {
			break
		}
		callback(&v)
	}
	return it.Err
}

// GetResourceLinksViaCallback returns resource links of devices. JWT token must be stored in context for grpc call.
func (c *Client) GetResourceLinksViaCallback(ctx context.Context, deviceIDs, resourceTypes []string, callback func(*events.ResourceLinksPublished)) error {
	it := c.GetResourceLinksIterator(ctx, deviceIDs, resourceTypes...)
	defer it.Close()
	for {
		var v events.ResourceLinksPublished
		if !it.Next(&v) {
			break
		}
		callback(&v)
	}
	return it.Err
}

// GetDevicesIterator gets devices. JWT token must be stored in context for grpc call.
// Next queries the next resource value.
// Returns false when failed or having no more items.
// Check it.Err for errors.
// Usage:
//
//	for {
//		var v MyStruct
//		if !it.Next(ctx, &v) {
//			break
//		}
//	}
//	if it.Err != nil {
//	}
func (c *Client) GetDevicesIterator(ctx context.Context, deviceIDs []string, resourceTypes ...string) *kitNetGrpc.Iterator {
	r := pb.GetDevicesRequest{DeviceIdFilter: deviceIDs, TypeFilter: resourceTypes}
	return kitNetGrpc.NewIterator(c.gateway.GetDevices(ctx, &r))
}

// GetResourceLinksIterator gets devices. JWT token must be stored in context for grpc call.
// Next queries the next resource value.
// Returns false when failed or having no more items.
// Check it.Err for errors.
// Usage:
//
//	for {
//		var v MyStruct
//		if !it.Next(ctx, &v) {
//			break
//		}
//	}
//	if it.Err != nil {
//	}
func (c *Client) GetResourceLinksIterator(ctx context.Context, deviceIDs []string, resourceTypes ...string) *kitNetGrpc.Iterator {
	r := pb.GetResourceLinksRequest{DeviceIdFilter: deviceIDs, TypeFilter: resourceTypes}
	return kitNetGrpc.NewIterator(c.gateway.GetResourceLinks(ctx, &r))
}

// GetResourcesIterator gets resources contents from resource twin (cache of backend). JWT token must be stored in context for grpc call.
// By resourceIDs you can specify resources by deviceID and Href which will be retrieved from the backend, nil means all resources.
// Or by deviceIDs or resourceTypes you can filter output when you get all resources.
// Eg:
//
//	 get all resources
//		it := client.GetResourcesIterator(ctx, nil, nil)
//
//	 get all oic.wk.d resources
//	 iter := client.GetResourcesIterator(ctx, nil, nil, "oic.wk.d")
//
//	 get oic.wk.d resources of 2 devices
//	 iter := client.GetResourcesIterator(ctx, nil, string["60f6869d-343a-4989-7462-81ef215d31af", "07ef9eb6-1ce9-4ce4-73a6-9ee0a1d534d2"], "oic.wk.d")
//
//	 get a certain resource /oic/p of the device"60f6869d-343a-4989-7462-81ef215d31af"
//	 iter := client.GetResourcesIterator(ctx, commands.NewResourceID("60f6869d-343a-4989-7462-81ef215d31af", /oic/p), nil)
//
// Next queries the next resource value.
// Returns false when failed or having no more items.
// Check it.Err for errors.
// Usage:
//
//	for {
//		var v MyStruct
//		if !it.Next(ctx, &v) {
//			break
//		}
//	}
//	if it.Err != nil {
//	}
func (c *Client) GetResourcesIterator(ctx context.Context, resourceIDs []string, deviceIDs []string, resourceTypes ...string) *kitNetGrpc.Iterator {
	r := pb.GetResourcesRequest{ResourceIdFilter: resourceIDs, DeviceIdFilter: deviceIDs, TypeFilter: resourceTypes}
	return kitNetGrpc.NewIterator(c.gateway.GetResources(ctx, &r))
}

type ResourceIDCallback struct {
	ResourceID *commands.ResourceId
	Callback   func(*pb.Resource)
}

func MakeResourceIDCallback(deviceID, href string, callback func(*pb.Resource)) ResourceIDCallback {
	return ResourceIDCallback{ResourceID: &commands.ResourceId{
		DeviceId: deviceID,
		Href:     href,
	}, Callback: callback}
}

// GetResourcesByResourceIDs gets resources contents by resourceIDs. JWT token must be stored in context for grpc call.
func (c *Client) GetResourcesByResourceIDs(
	ctx context.Context,
	resourceIDsCallbacks ...ResourceIDCallback,
) error {
	tc := make(map[string]func(*pb.Resource), len(resourceIDsCallbacks))
	resourceIDs := make([]string, 0, len(resourceIDsCallbacks))
	for _, c := range resourceIDsCallbacks {
		tc[c.ResourceID.GetDeviceId()+c.ResourceID.GetHref()] = c.Callback
		resourceIDs = append(resourceIDs, c.ResourceID.ToString())
	}

	it := c.GetResourcesIterator(ctx, resourceIDs, nil)
	defer it.Close()

	for {
		var v pb.Resource
		if !it.Next(&v) {
			break
		}
		c, ok := tc[v.GetData().GetResourceId().GetDeviceId()+v.GetData().GetResourceId().GetHref()]
		if ok {
			c(&v)
		}
	}
	return it.Err
}

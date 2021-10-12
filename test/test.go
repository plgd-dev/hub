package test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	deviceClient "github.com/plgd-dev/device/client"
	"github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/acl"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/grpc-gateway/client"
	"github.com/plgd-dev/hub/grpc-gateway/pb"
	"github.com/plgd-dev/hub/pkg/ocf"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	"github.com/plgd-dev/hub/test/config"
	oauthTest "github.com/plgd-dev/hub/test/oauth-server/test"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

var (
	TestDeviceName string

	TestDevsimResources        []schema.ResourceLink
	TestDevsimBackendResources []schema.ResourceLink
)

const (
	OCFResourcePlatformHref      = "/oic/p"
	OCFResourceDeviceHref        = "/oic/d"
	OCFResourceConfigurationHref = "/oc/con"
	TestResourceLightHref        = "/light/1"
	TestResourceSwitchesHref     = "/switches"
)

func TestResourceSwitchesInstanceHref(id string) string {
	return TestResourceSwitchesHref + "/" + id
}

func init() {
	TestDeviceName = "devsim-" + MustGetHostname()
	TestDevsimResources = []schema.ResourceLink{
		{
			Href:          OCFResourcePlatformHref,
			ResourceTypes: []string{ocf.OC_RT_P},
			Interfaces:    []string{ocf.OC_IF_R, ocf.OC_IF_BASELINE},
			Policy: &schema.Policy{
				BitMask: 3,
			},
		},

		{
			Href:          OCFResourceDeviceHref,
			ResourceTypes: []string{ocf.OC_RT_DEVICE_CLOUD, ocf.OC_RT_D},
			Interfaces:    []string{ocf.OC_IF_R, ocf.OC_IF_BASELINE},
			Policy: &schema.Policy{
				BitMask: 3,
			},
		},

		{
			Href:          OCFResourceConfigurationHref,
			ResourceTypes: []string{ocf.OC_RT_CON},
			Interfaces:    []string{ocf.OC_IF_RW, ocf.OC_IF_BASELINE},
			Policy: &schema.Policy{
				BitMask: 3,
			},
		},

		{
			Href:          TestResourceLightHref,
			ResourceTypes: []string{"core.light"},
			Interfaces:    []string{ocf.OC_IF_RW, ocf.OC_IF_BASELINE},
			Policy: &schema.Policy{
				BitMask: 3,
			},
		},

		{
			Href:          TestResourceSwitchesHref,
			ResourceTypes: []string{ocf.OC_RT_COL},
			Interfaces:    []string{ocf.OC_IF_LL, ocf.OC_IF_CREATE, ocf.OC_IF_B, ocf.OC_IF_BASELINE},
			Policy: &schema.Policy{
				BitMask: 3,
			},
		},
	}
}

func DefaultSwitchResourceLink(id string) schema.ResourceLink {
	return schema.ResourceLink{
		Href:          TestResourceSwitchesInstanceHref(id),
		ResourceTypes: []string{ocf.OC_RT_RESOURCE_SWITCH},
		Interfaces:    []string{ocf.OC_IF_A, ocf.OC_IF_BASELINE},
		Policy: &schema.Policy{
			BitMask: schema.BitMask(schema.Discoverable | schema.Observable),
		},
	}
}

func MakeSwitchResourceData(overrides map[string]interface{}) map[string]interface{} {
	data := MakeSwitchResourceDefaultData()
	for k, v := range overrides {
		data[k] = v
	}
	return data
}

func MakeSwitchResourceDefaultData() map[string]interface{} {
	s := DefaultSwitchResourceLink("")
	return map[string]interface{}{
		"if": s.Interfaces,
		"rt": s.ResourceTypes,
		"rep": map[string]interface{}{
			"value": false,
		},
		"p": map[string]interface{}{
			"bm": uint64(s.Policy.BitMask),
		},
	}
}

func AddDeviceSwitchResources(ctx context.Context, t *testing.T, deviceID string, c pb.GrpcGatewayClient, resourceIDs ...string) []schema.ResourceLink {
	toStringArray := func(v interface{}) []string {
		var result []string
		arr, ok := v.([]interface{})
		require.True(t, ok)
		for _, val := range arr {
			str, ok := val.(string)
			require.True(t, ok)
			result = append(result, str)
		}
		return result
	}

	links := make([]schema.ResourceLink, 0, len(resourceIDs))
	for _, resourceID := range resourceIDs {
		req := &pb.CreateResourceRequest{
			ResourceId: commands.NewResourceID(deviceID, TestResourceSwitchesHref),
			Content: &pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data:        EncodeToCbor(t, MakeSwitchResourceDefaultData()),
			},
		}
		resp, err := c.CreateResource(ctx, req)
		require.NoError(t, err)

		respData, ok := DecodeCbor(t, resp.GetData().GetContent().GetData()).(map[interface{}]interface{})
		require.True(t, ok)

		href, ok := respData["href"].(string)
		require.True(t, ok)
		require.Equal(t, TestResourceSwitchesInstanceHref(resourceID), href)

		resourceTypes := toStringArray(respData["rt"])
		interfaces := toStringArray(respData["if"])

		policy, ok := respData["p"].(map[interface{}]interface{})
		require.True(t, ok)
		bitmask, ok := policy["bm"].(uint64)
		require.True(t, ok)

		links = append(links, schema.ResourceLink{
			Href:          href,
			ResourceTypes: resourceTypes,
			Interfaces:    interfaces,
			Policy: &schema.Policy{
				BitMask: schema.BitMask(bitmask),
			},
		})
	}
	return links
}

func setAccessForCloud(ctx context.Context, t *testing.T, c *deviceClient.Client, deviceID string) {
	cloudSID := config.HubID()
	require.NotEmpty(t, cloudSID)

	d, links, err := c.GetRefDevice(ctx, deviceID)
	require.NoError(t, err)

	defer func() {
		err := d.Release(ctx)
		require.NoError(t, err)
	}()
	p, err := d.Provision(ctx, links)
	require.NoError(t, err)
	defer func() {
		_ = p.Close(ctx)
	}()

	link, err := core.GetResourceLink(links, "/oic/sec/acl2")
	require.NoError(t, err)

	setAcl := acl.UpdateRequest{
		AccessControlList: []acl.AccessControl{
			{
				Permission: acl.AllPermissions,
				Subject: acl.Subject{
					Subject_Device: &acl.Subject_Device{
						DeviceID: cloudSID,
					},
				},
				Resources: acl.AllResources,
			},
		},
	}

	err = p.UpdateResource(ctx, link, setAcl, nil)
	require.NoError(t, err)
}

func OnboardDevSimForClient(ctx context.Context, t *testing.T, c pb.GrpcGatewayClient, clientId, deviceID, gwHost string, expectedResources []schema.ResourceLink) (string, func()) {
	cloudSID := config.HubID()
	require.NotEmpty(t, cloudSID)
	client, err := NewSDKClient()
	require.NoError(t, err)
	defer func() {
		_ = client.Close(ctx)
	}()
	deviceID, err = client.OwnDevice(ctx, deviceID, deviceClient.WithOTM(deviceClient.OTMType_JustWorks))
	require.NoError(t, err)

	setAccessForCloud(ctx, t, client, deviceID)

	code := oauthTest.GetDeviceAuthorizationCode(t, config.OAUTH_SERVER_HOST, clientId, deviceID)
	err = client.OnboardDevice(ctx, deviceID, config.DEVICE_PROVIDER, "coaps+tcp://"+gwHost, code, cloudSID)
	require.NoError(t, err)

	if len(expectedResources) > 0 {
		waitForDevice(ctx, t, c, deviceID, expectedResources)
	}

	return deviceID, func() {
		client, err := NewSDKClient()
		require.NoError(t, err)
		err = client.DisownDevice(ctx, deviceID)
		require.NoError(t, err)
		err = client.Close(ctx)
		require.NoError(t, err)
		time.Sleep(time.Second * 2)
	}
}

func OnboardDevSim(ctx context.Context, t *testing.T, c pb.GrpcGatewayClient, deviceID, gwHost string, expectedResources []schema.ResourceLink) (string, func()) {
	return OnboardDevSimForClient(ctx, t, c, config.OAUTH_MANAGER_CLIENT_ID, deviceID, gwHost, expectedResources)
}

func waitForDevice(ctx context.Context, t *testing.T, c pb.GrpcGatewayClient, deviceID string, expectedResources []schema.ResourceLink) {
	client, err := client.New(c).SubscribeToEventsWithCurrentState(ctx, time.Second)
	require.NoError(t, err)

	err = client.Send(&pb.SubscribeToEvents{
		CorrelationId: "testToken0",
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
					pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED,
				},
			},
		},
	})
	require.NoError(t, err)
	ev, err := client.Recv()
	require.NoError(t, err)
	expectedEvent := &pb.Event{
		SubscriptionId: ev.SubscriptionId,
		CorrelationId:  "testToken0",
		Type: &pb.Event_OperationProcessed_{
			OperationProcessed: &pb.Event_OperationProcessed{
				ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
					Code: pb.Event_OperationProcessed_ErrorStatus_OK,
				},
			},
		},
	}
	CheckProtobufs(t, expectedEvent, ev, RequireToCheckFunc(require.Equal))

	for {
		ev, err = client.Recv()
		require.NoError(t, err)
		var endLoop bool

		if ev.GetDeviceMetadataUpdated().GetDeviceId() == deviceID && ev.GetDeviceMetadataUpdated().GetStatus().IsOnline() {
			endLoop = true
		}
		if endLoop {
			break
		}
	}

	err = client.Send(&pb.SubscribeToEvents{
		CorrelationId: "testToken1",
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				DeviceIdFilter: []string{deviceID},
				EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
					pb.SubscribeToEvents_CreateSubscription_RESOURCE_PUBLISHED,
				},
			},
		},
	})
	require.NoError(t, err)
	ev, err = client.Recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		CorrelationId: "testToken1",
		Type: &pb.Event_OperationProcessed_{
			OperationProcessed: &pb.Event_OperationProcessed{
				ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
					Code: pb.Event_OperationProcessed_ErrorStatus_OK,
				},
			},
		},
	}
	expectedEvent.SubscriptionId = ev.SubscriptionId
	CheckProtobufs(t, expectedEvent, ev, RequireToCheckFunc(require.Equal))
	subOnPublishedID := ev.SubscriptionId

	expectedLinks := make(map[string]*commands.Resource)
	for _, link := range ResourceLinksToResources(deviceID, expectedResources) {
		expectedLinks[link.GetHref()] = link
	}

	for {
		ev, err = client.Recv()
		require.NoError(t, err)
		ev.SubscriptionId = ""

		for _, l := range ev.GetResourcePublished().GetResources() {
			expLink := expectedLinks[l.GetHref()]
			l.ValidUntil = 0
			CheckProtobufs(t, expLink, l, RequireToCheckFunc(require.Equal))
			delete(expectedLinks, l.GetHref())
		}
		if len(expectedLinks) == 0 {
			break
		}
	}

	err = client.Send(&pb.SubscribeToEvents{
		CorrelationId: "testToken3",
		Action: &pb.SubscribeToEvents_CancelSubscription_{
			CancelSubscription: &pb.SubscribeToEvents_CancelSubscription{
				SubscriptionId: subOnPublishedID,
			},
		},
	})
	require.NoError(t, err)

	ev, err = client.Recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: ev.SubscriptionId,
		CorrelationId:  "testToken3",
		Type: &pb.Event_SubscriptionCanceled_{
			SubscriptionCanceled: &pb.Event_SubscriptionCanceled{},
		},
	}
	CheckProtobufs(t, expectedEvent, ev, RequireToCheckFunc(require.Equal))

	ev, err = client.Recv()
	require.NoError(t, err)
	expectedEvent = &pb.Event{
		SubscriptionId: ev.SubscriptionId,
		CorrelationId:  "testToken3",
		Type: &pb.Event_OperationProcessed_{
			OperationProcessed: &pb.Event_OperationProcessed{
				ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
					Code: pb.Event_OperationProcessed_ErrorStatus_OK,
				},
			},
		},
	}
	CheckProtobufs(t, expectedEvent, ev, RequireToCheckFunc(require.Equal))

	expectedEvents := ResourceLinksToExpectedResourceChangedEvents(deviceID, expectedResources)
	for _, e := range expectedEvents {
		err = client.Send(&pb.SubscribeToEvents{
			CorrelationId: "testToken3",
			Action: &pb.SubscribeToEvents_CreateSubscription_{
				CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
					ResourceIdFilter: []string{
						e.GetResourceChanged().GetResourceId().ToString(),
					},
					EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
						pb.SubscribeToEvents_CreateSubscription_RESOURCE_CHANGED,
					},
				},
			},
		})
		require.NoError(t, err)
		ev, err := client.Recv()
		require.NoError(t, err)
		expectedEvent := &pb.Event{
			SubscriptionId: ev.SubscriptionId,
			CorrelationId:  "testToken3",
			Type: &pb.Event_OperationProcessed_{
				OperationProcessed: &pb.Event_OperationProcessed{
					ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
						Code: pb.Event_OperationProcessed_ErrorStatus_OK,
					},
				},
			},
		}
		CheckProtobufs(t, expectedEvent, ev, RequireToCheckFunc(require.Equal))

		ev, err = client.Recv()
		require.NoError(t, err)
		require.Equal(t, e.GetResourceChanged().GetResourceId(), ev.GetResourceChanged().GetResourceId())
		require.Equal(t, e.GetResourceChanged().GetStatus(), ev.GetResourceChanged().GetStatus())

		err = client.Send(&pb.SubscribeToEvents{
			CorrelationId: "testToken4",
			Action: &pb.SubscribeToEvents_CancelSubscription_{
				CancelSubscription: &pb.SubscribeToEvents_CancelSubscription{
					SubscriptionId: ev.GetSubscriptionId(),
				},
			},
		})
		require.NoError(t, err)

		ev, err = client.Recv()
		require.NoError(t, err)
		expectedEvent = &pb.Event{
			SubscriptionId: ev.SubscriptionId,
			CorrelationId:  "testToken4",
			Type: &pb.Event_SubscriptionCanceled_{
				SubscriptionCanceled: &pb.Event_SubscriptionCanceled{},
			},
		}
		CheckProtobufs(t, expectedEvent, ev, RequireToCheckFunc(require.Equal))

		ev, err = client.Recv()
		require.NoError(t, err)
		expectedEvent = &pb.Event{
			SubscriptionId: ev.SubscriptionId,
			CorrelationId:  "testToken4",
			Type: &pb.Event_OperationProcessed_{
				OperationProcessed: &pb.Event_OperationProcessed{
					ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
						Code: pb.Event_OperationProcessed_ErrorStatus_OK,
					},
				},
			},
		}
		CheckProtobufs(t, expectedEvent, ev, RequireToCheckFunc(require.Equal))
	}

	err = client.CloseSend()
	require.NoError(t, err)
}

func MustGetHostname() string {
	n, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	return n
}

func MustFindDeviceByName(name string) (deviceID string) {
	var err error
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		deviceID, err = FindDeviceByName(ctx, name)
		if err == nil {
			return deviceID
		}
	}
	panic(err)
}

type findDeviceIDByNameHandler struct {
	id     atomic.Value
	name   string
	cancel context.CancelFunc
}

func (h *findDeviceIDByNameHandler) Handle(ctx context.Context, dev *core.Device, deviceLinks schema.ResourceLinks) {
	defer func() {
		err := dev.Close(ctx)
		h.Error(err)
	}()
	l, ok := deviceLinks.GetResourceLink("/oic/d")
	if !ok {
		return
	}
	var d device.Device
	err := dev.GetResource(ctx, l, &d)
	if err != nil {
		return
	}
	if d.Name == h.name {
		h.id.Store(d.ID)
		h.cancel()
	}
}

func (h *findDeviceIDByNameHandler) Error(err error) {}

func FindDeviceByName(ctx context.Context, name string) (deviceID string, _ error) {
	client := core.NewClient()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	h := findDeviceIDByNameHandler{
		name:   name,
		cancel: cancel,
	}

	err := client.GetDevices(ctx, &h)
	if err != nil {
		return "", fmt.Errorf("could not find the device named %s: %w", name, err)
	}
	id, ok := h.id.Load().(string)
	if !ok || id == "" {
		return "", fmt.Errorf("could not find the device named %s: not found", name)
	}
	return id, nil
}

func GetAllBackendResourceLinks() []schema.ResourceLink {
	return append(TestDevsimResources, TestDevsimBackendResources...)
}

func ProtobufToInterface(t *testing.T, val interface{}) interface{} {
	expJSON, err := json.Encode(val)
	require.NoError(t, err)
	var v interface{}
	err = json.Decode(expJSON, &v)
	require.NoError(t, err)
	return v
}

func RequireToCheckFunc(checFunc func(t require.TestingT, expected interface{}, actual interface{}, msgAndArgs ...interface{})) func(t *testing.T, expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
	return func(t *testing.T, expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
		checFunc(t, expected, actual, msgAndArgs)
	}
}

func AssertToCheckFunc(checFunc func(t assert.TestingT, expected interface{}, actual interface{}, msgAndArgs ...interface{}) bool) func(t *testing.T, expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
	return func(t *testing.T, expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
		checFunc(t, expected, actual, msgAndArgs)
	}
}

func CheckProtobufs(t *testing.T, expected interface{}, actual interface{}, checkFunc func(t *testing.T, expected interface{}, actual interface{}, msgAndArgs ...interface{})) {
	v1 := ProtobufToInterface(t, expected)
	v2 := ProtobufToInterface(t, actual)
	checkFunc(t, v1, v2)
}

func NATSSStart(ctx context.Context, t *testing.T) {
	err := exec.CommandContext(ctx, "docker", "start", "nats").Run()
	require.NoError(t, err)
}

func NATSSStop(ctx context.Context, t *testing.T) {
	err := exec.CommandContext(ctx, "docker", "stop", "nats").Run()
	require.NoError(t, err)
}

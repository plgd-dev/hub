package test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	deviceClient "github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/acl"
	"github.com/plgd-dev/device/v2/schema/collection"
	"github.com/plgd-dev/device/v2/schema/configuration"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/device/v2/test/resource/types"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/client"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	pkgStrings "github.com/plgd-dev/hub/v2/pkg/strings"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

var (
	TestDeviceName                     string
	TestDeviceNameWithOicResObservable string

	TestDevsimResources        []schema.ResourceLink
	TestDevsimBackendResources []schema.ResourceLink
)

const (
	TestResourceSwitchesHref = "/switches"
)

func TestResourceLightInstanceHref(id string) string {
	return "/light/" + id
}

func TestResourceSwitchesInstanceHref(id string) string {
	return TestResourceSwitchesHref + "/" + id
}

func init() {
	TestDeviceName = "devsim-" + MustGetHostname()
	TestDeviceNameWithOicResObservable = "devsim-resobs-" + MustGetHostname()
	TestDevsimResources = []schema.ResourceLink{
		{
			Href:          platform.ResourceURI,
			ResourceTypes: []string{platform.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
			Policy: &schema.Policy{
				BitMask: 3,
			},
		},

		{
			Href:          device.ResourceURI,
			ResourceTypes: []string{types.DEVICE_CLOUD, device.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
			Policy: &schema.Policy{
				BitMask: 3,
			},
		},

		{
			Href:          configuration.ResourceURI,
			ResourceTypes: []string{configuration.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
			Policy: &schema.Policy{
				BitMask: 3,
			},
		},

		{
			Href:          TestResourceLightInstanceHref("1"),
			ResourceTypes: []string{types.CORE_LIGHT},
			Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
			Policy: &schema.Policy{
				BitMask: 3,
			},
		},

		{
			Href:          TestResourceSwitchesHref,
			ResourceTypes: []string{collection.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_LL, interfaces.OC_IF_CREATE, interfaces.OC_IF_B, interfaces.OC_IF_BASELINE},
			Policy: &schema.Policy{
				BitMask: 3,
			},
		},
	}
}

func FilterResourceLink(filter func(schema.ResourceLink) bool, links []schema.ResourceLink) []schema.ResourceLink {
	var l []schema.ResourceLink
	for _, link := range links {
		if filter(link) {
			l = append(l, link)
		}
	}
	return l
}

func DefaultSwitchResourceLink(deviceID, id string) schema.ResourceLink {
	return schema.ResourceLink{
		DeviceID:      deviceID,
		Href:          TestResourceSwitchesInstanceHref(id),
		ResourceTypes: []string{types.BINARY_SWITCH},
		Interfaces:    []string{interfaces.OC_IF_A, interfaces.OC_IF_BASELINE},
		Policy: &schema.Policy{
			BitMask: schema.Discoverable | schema.Observable,
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
	s := DefaultSwitchResourceLink("", "")
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

	d, links, err := c.GetDevice(ctx, deviceID)
	require.NoError(t, err)

	defer func() {
		errC := d.Close(ctx)
		require.NoError(t, errC)
	}()
	p, err := d.Provision(ctx, links)
	require.NoError(t, err)
	defer func() {
		_ = p.Close(ctx)
	}()

	link, err := core.GetResourceLink(links, acl.ResourceURI)
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

func OnboardDevSimForClient(ctx context.Context, t *testing.T, c pb.GrpcGatewayClient, clientID, deviceID, hubEndpoint string, expectedResources []schema.ResourceLink) (string, func()) {
	cloudSID := config.HubID()
	require.NotEmpty(t, cloudSID)
	devClient, err := NewSDKClient()
	require.NoError(t, err)
	defer func() {
		_ = devClient.Close(ctx)
	}()
	deviceID, err = devClient.OwnDevice(ctx, deviceID, deviceClient.WithOTM(deviceClient.OTMType_JustWorks))
	require.NoError(t, err)

	setAccessForCloud(ctx, t, devClient, deviceID)

	code := oauthTest.GetAuthorizationCode(t, config.OAUTH_SERVER_HOST, clientID, deviceID, "")

	onboard := func() {
		err = devClient.OnboardDevice(ctx, deviceID, config.DEVICE_PROVIDER, hubEndpoint, code, cloudSID)
		require.NoError(t, err)
	}
	if len(expectedResources) > 0 {
		subClient, err := client.New(c).GrpcGatewayClient().SubscribeToEvents(ctx)
		require.NoError(t, err)
		err = subClient.Send(&pb.SubscribeToEvents{
			CorrelationId: "allEvents",
			Action: &pb.SubscribeToEvents_CreateSubscription_{
				CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{},
			},
		})
		require.NoError(t, err)
		ev, err := subClient.Recv()
		require.NoError(t, err)
		expectedEvent := &pb.Event{
			SubscriptionId: ev.SubscriptionId,
			CorrelationId:  "allEvents",
			Type: &pb.Event_OperationProcessed_{
				OperationProcessed: &pb.Event_OperationProcessed{
					ErrorStatus: &pb.Event_OperationProcessed_ErrorStatus{
						Code: pb.Event_OperationProcessed_ErrorStatus_OK,
					},
				},
			},
		}
		CheckProtobufs(t, expectedEvent, ev, RequireToCheckFunc(require.Equal))
		onboard()
		WaitForDevice(ctx, t, subClient, deviceID, ev.GetSubscriptionId(), ev.GetCorrelationId(), expectedResources)
		err = subClient.CloseSend()
		require.NoError(t, err)
	} else {
		onboard()
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

func OnboardDevSim(ctx context.Context, t *testing.T, c pb.GrpcGatewayClient, deviceID, hubEndpoint string, expectedResources []schema.ResourceLink) (string, func()) {
	return OnboardDevSimForClient(ctx, t, c, config.OAUTH_MANAGER_CLIENT_ID, deviceID, hubEndpoint, expectedResources)
}

func WaitForDevice(ctx context.Context, t *testing.T, client pb.GrpcGateway_SubscribeToEventsClient, deviceID, subID, correlationID string, expectedResources []schema.ResourceLink) {
	getID := func(ev *pb.Event) string {
		switch v := ev.GetType().(type) {
		case *pb.Event_DeviceRegistered_:
			return fmt.Sprintf("%T", ev.GetType())
		case *pb.Event_DeviceMetadataUpdated:
			return fmt.Sprintf("%T:%v:%v", ev.GetType(), v.DeviceMetadataUpdated.GetConnection().GetStatus(), v.DeviceMetadataUpdated.GetTwinSynchronization().GetState())
		case *pb.Event_ResourcePublished:
			return fmt.Sprintf("%T", ev.GetType())
		case *pb.Event_ResourceChanged:
			return fmt.Sprintf("%T:%v", ev.GetType(), v.ResourceChanged.GetResourceId().ToString())
		}
		return ""
	}

	cleanUpEvent := func(ev *pb.Event) {
		switch val := ev.GetType().(type) {
		case *pb.Event_DeviceRegistered_:
			val.DeviceRegistered.OpenTelemetryCarrier = nil
		case *pb.Event_DeviceUnregistered_:
			val.DeviceUnregistered.OpenTelemetryCarrier = nil
		case *pb.Event_DeviceMetadataUpdated:
			require.NotEmpty(t, val.DeviceMetadataUpdated.GetAuditContext().GetUserId())
			val.DeviceMetadataUpdated.AuditContext = nil
			require.NotZero(t, val.DeviceMetadataUpdated.GetEventMetadata().GetTimestamp())
			val.DeviceMetadataUpdated.EventMetadata = nil
			if val.DeviceMetadataUpdated.GetConnection() != nil {
				val.DeviceMetadataUpdated.GetConnection().Id = ""
				require.NotZero(t, val.DeviceMetadataUpdated.GetConnection().GetConnectedAt())
				val.DeviceMetadataUpdated.GetConnection().ConnectedAt = 0
			}
			if val.DeviceMetadataUpdated.GetTwinSynchronization() != nil {
				val.DeviceMetadataUpdated.GetTwinSynchronization().CommandMetadata = nil
				val.DeviceMetadataUpdated.GetTwinSynchronization().StartedAt = 0
				val.DeviceMetadataUpdated.GetTwinSynchronization().FinishedAt = 0
			}
			val.DeviceMetadataUpdated.OpenTelemetryCarrier = nil
		case *pb.Event_ResourcePublished:
			require.NotEmpty(t, val.ResourcePublished.GetAuditContext().GetUserId())
			val.ResourcePublished.AuditContext = nil
			require.NotZero(t, val.ResourcePublished.GetEventMetadata().GetTimestamp())
			val.ResourcePublished.EventMetadata = nil
			val.ResourcePublished.Resources = CleanUpResourcesArray(val.ResourcePublished.GetResources())
			val.ResourcePublished.OpenTelemetryCarrier = nil
		case *pb.Event_ResourceChanged:
			require.NotEmpty(t, val.ResourceChanged.GetAuditContext().GetUserId())
			val.ResourceChanged.AuditContext = nil
			require.NotZero(t, val.ResourceChanged.GetEventMetadata().GetTimestamp())
			val.ResourceChanged.EventMetadata = nil
			require.NotEmpty(t, val.ResourceChanged.GetContent().GetData())
			val.ResourceChanged.Content = nil
			val.ResourceChanged.OpenTelemetryCarrier = nil
		}
	}

	expectedEvents := map[string]*pb.Event{
		getID(&pb.Event{Type: &pb.Event_DeviceRegistered_{}}): {
			SubscriptionId: subID,
			CorrelationId:  correlationID,
			Type: &pb.Event_DeviceRegistered_{
				DeviceRegistered: &pb.Event_DeviceRegistered{
					DeviceIds: []string{deviceID},
				},
			},
		},
		getID(&pb.Event{Type: &pb.Event_DeviceMetadataUpdated{
			DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
				Connection: &commands.Connection{
					Status: commands.Connection_ONLINE,
				},
			},
		}}): {
			SubscriptionId: subID,
			CorrelationId:  correlationID,
			Type: &pb.Event_DeviceMetadataUpdated{
				DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
					DeviceId: deviceID,
					Connection: &commands.Connection{
						Status: commands.Connection_ONLINE,
					},
					TwinSynchronization: &commands.TwinSynchronization{
						State: commands.TwinSynchronization_NONE,
					},
				},
			},
		},
		getID(&pb.Event{Type: &pb.Event_DeviceMetadataUpdated{
			DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
				Connection: &commands.Connection{
					Status: commands.Connection_ONLINE,
				},
				TwinSynchronization: &commands.TwinSynchronization{
					State: commands.TwinSynchronization_STARTED,
				},
			},
		}}): {
			SubscriptionId: subID,
			CorrelationId:  correlationID,
			Type: &pb.Event_DeviceMetadataUpdated{
				DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
					DeviceId: deviceID,
					Connection: &commands.Connection{
						Status: commands.Connection_ONLINE,
					},
					TwinSynchronization: &commands.TwinSynchronization{
						State: commands.TwinSynchronization_STARTED,
					},
				},
			},
		},
		getID(&pb.Event{Type: &pb.Event_DeviceMetadataUpdated{
			DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
				Connection: &commands.Connection{
					Status: commands.Connection_ONLINE,
				},
				TwinSynchronization: &commands.TwinSynchronization{
					State: commands.TwinSynchronization_FINISHED,
				},
			},
		}}): {
			SubscriptionId: subID,
			CorrelationId:  correlationID,
			Type: &pb.Event_DeviceMetadataUpdated{
				DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
					DeviceId: deviceID,
					Connection: &commands.Connection{
						Status: commands.Connection_ONLINE,
					},
					TwinSynchronization: &commands.TwinSynchronization{
						State: commands.TwinSynchronization_FINISHED,
					},
				},
			},
		},
		getID(&pb.Event{Type: &pb.Event_ResourcePublished{}}): {
			SubscriptionId: subID,
			CorrelationId:  correlationID,
			Type: &pb.Event_ResourcePublished{
				ResourcePublished: &events.ResourceLinksPublished{
					DeviceId:  deviceID,
					Resources: ResourceLinksToResources(deviceID, expectedResources),
				},
			},
		},
	}
	for _, r := range expectedResources {
		expectedEvents[getID(&pb.Event{Type: &pb.Event_ResourceChanged{
			ResourceChanged: &events.ResourceChanged{
				ResourceId: commands.NewResourceID(deviceID, r.Href),
			},
		}})] = &pb.Event{
			SubscriptionId: subID,
			CorrelationId:  correlationID,
			Type: &pb.Event_ResourceChanged{
				ResourceChanged: &events.ResourceChanged{
					ResourceId: commands.NewResourceID(deviceID, r.Href),
					Status:     commands.Status_OK,
				},
			},
		}
	}

	for {
		ev, err := client.Recv()
		require.NoError(t, err)

		expectedEvent, ok := expectedEvents[getID(ev)]
		if !ok {
			t.Logf("unexpected event %+v", ev)
			continue
		}
		cleanUpEvent(ev)
		CheckProtobufs(t, expectedEvent, ev, RequireToCheckFunc(require.Equal))
		delete(expectedEvents, getID(ev))
		if len(expectedEvents) == 0 {
			return
		}
	}
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

func (h *findDeviceIDByNameHandler) Handle(ctx context.Context, dev *core.Device) {
	defer func() {
		err := dev.Close(ctx)
		h.Error(err)
	}()
	deviceLinks, err := dev.GetResourceLinks(ctx, dev.GetEndpoints())
	if err != nil {
		h.Error(err)
		return
	}
	l, ok := deviceLinks.GetResourceLink(device.ResourceURI)
	if !ok {
		return
	}
	var d device.Device
	err = dev.GetResource(ctx, l, &d)
	if err != nil {
		h.Error(err)
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

	err := client.GetDevicesByMulticast(ctx, core.DefaultDiscoveryConfiguration(), &h)
	if err != nil {
		return "", fmt.Errorf("could not find the device named %s: %w", name, err)
	}
	id, ok := h.id.Load().(string)
	if !ok || id == "" {
		return "", fmt.Errorf("could not find the device named %s: not found", name)
	}
	return id, nil
}

func IsDiscoveryResourceBatchObservable(ctx context.Context, t *testing.T, deviceID string) bool {
	devClient, err := NewSDKClient()
	require.NoError(t, err)
	defer func() {
		_ = devClient.Close(ctx)
	}()

	_, links, err := devClient.GetDevice(ctx, deviceID)
	require.NoError(t, err)
	links = links.GetResourceLinks(resources.ResourceType)
	if len(links) == 0 {
		return false
	}
	for _, iface := range links[0].Interfaces {
		if iface == interfaces.OC_IF_B && links[0].Policy.BitMask.Has(schema.Observable) {
			return true
		}
	}
	return false
}

func CheckResource(ctx context.Context, t *testing.T, deviceID, href, resourceType string, checkFn func(schema.ResourceLink) bool) bool {
	devClient, err := NewSDKClient()
	require.NoError(t, err)
	defer func() {
		_ = devClient.Close(ctx)
	}()

	var resp schema.ResourceLinks
	err = devClient.GetResource(ctx, deviceID, resources.ResourceURI, &resp, deviceClient.WithResourceTypes(resources.ResourceType))
	require.NoError(t, err)

	return len(resp) == 1 && checkFn(resp[0])
}

func ResourceIsBatchObservable(ctx context.Context, t *testing.T, deviceID, href, resourceType string) bool {
	return CheckResource(ctx, t, deviceID, href, resourceType, func(rl schema.ResourceLink) bool {
		return rl.Policy.BitMask.Has(schema.Observable) &&
			pkgStrings.Contains(rl.Interfaces, interfaces.OC_IF_B)
	})
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

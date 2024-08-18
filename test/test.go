package test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	deviceClient "github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/client/core"
	bridgeDevice "github.com/plgd-dev/device/v2/cmd/bridge-device/device"
	deviceCoap "github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/acl"
	"github.com/plgd-dev/device/v2/schema/cloud"
	"github.com/plgd-dev/device/v2/schema/collection"
	"github.com/plgd-dev/device/v2/schema/configuration"
	schemaDevice "github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/maintenance"
	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/plgd-dev/device/v2/schema/plgdtime"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/device/v2/schema/softwareupdate"
	"github.com/plgd-dev/device/v2/test/resource/types"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/pkg/sync"
	"github.com/plgd-dev/hub/v2/grpc-gateway/client"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	isEvents "github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/hub/v2/test/device"
	"github.com/plgd-dev/hub/v2/test/device/bridge"
	"github.com/plgd-dev/hub/v2/test/device/ocf"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/sdk"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ugorji/go/codec"
)

type ResourceLinkRepresentation struct {
	Href           string      /*`json:"href"`*/
	ResourceTypes  []string    /*`json:"rt"`*/
	Representation interface{} /*`json:"rep"`*/
}

func (d *ResourceLinkRepresentation) MarshalJSON() ([]byte, error) {
	v, err := json.Encode(d.Representation)
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf(`{"href":"%v","rep":%s}`, d.Href, v)), nil
}

func (d *ResourceLinkRepresentation) UnmarshalJSON(data []byte) error {
	type representation struct {
		decode        func(data []byte) (interface{}, error)
		resourceTypes []string
	}
	reps := map[string]representation{
		configuration.ResourceURI: {
			decode: func(data []byte) (interface{}, error) {
				var r configuration.Configuration
				err := json.Decode(data, &r)
				return r, err
			},
			resourceTypes: []string{configuration.ResourceType},
		},
		schemaDevice.ResourceURI: {
			decode: func(data []byte) (interface{}, error) {
				var r schemaDevice.Device
				err := json.Decode(data, &r)
				r.ProtocolIndependentID = ""
				return r, err
			},
			resourceTypes: TestResourceDeviceResourceTypes,
		},
		platform.ResourceURI: {
			decode: func(data []byte) (interface{}, error) {
				var r platform.Platform
				err := json.Decode(data, &r)
				r.PlatformIdentifier = ""
				return r, err
			},
			resourceTypes: []string{platform.ResourceType},
		},
		maintenance.ResourceURI: {
			decode: func(data []byte) (interface{}, error) {
				var r MaintenanceResourceRepresentation
				err := json.Decode(data, &r)
				return r, err
			},
			resourceTypes: []string{maintenance.ResourceType},
		},
		plgdtime.ResourceURI: {
			decode: func(data []byte) (interface{}, error) {
				var r PlgdTimeResourceRepresentation
				err := json.Decode(data, &r)
				return r, err
			},
			resourceTypes: []string{plgdtime.ResourceType},
		},
		TestResourceLightInstanceHref("1"): {
			decode: func(data []byte) (interface{}, error) {
				var r LightResourceRepresentation
				err := json.Decode(data, &r)
				return r, err
			},
			resourceTypes: TestResourceLightInstanceResourceTypes,
		},
		TestResourceSwitchesHref: {
			decode: func(data []byte) (interface{}, error) {
				var r schema.ResourceLinks
				err := json.Decode(data, &r)
				if err != nil {
					return nil, err
				}
				r.Sort()
				for i := range r {
					r[i].Endpoints = nil
					r[i].InstanceID = 0
				}

				return r, err
			},
			resourceTypes: TestResourceSwitchesResourceTypes,
		},
		TestResourceSwitchesInstanceHref("1"): {
			decode: func(data []byte) (interface{}, error) {
				var r SwitchResourceRepresentation
				err := json.Decode(data, &r)
				return r, err
			},
			resourceTypes: TestResourceLightInstanceResourceTypes,
		},
	}
	var rep struct {
		Href string    `json:"href"`
		Rep  codec.Raw `json:"rep"`
	}
	err := json.Decode(data, &rep)
	if err != nil {
		return err
	}
	dec := representation{
		decode: func(data []byte) (interface{}, error) {
			var r interface{}
			errD := json.Decode(data, &r)
			return r, errD
		},
	}
	for k, v := range reps {
		if strings.HasSuffix(rep.Href, k) {
			dec = v
			break
		}
	}
	d.Href = rep.Href
	d.ResourceTypes = dec.resourceTypes
	d.Representation, err = dec.decode(rep.Rep)
	if err != nil {
		return err
	}
	return err
}

var (
	TestDeviceName                     string
	TestDeviceNameWithOicResObservable string
	TestDeviceModelNumber              = "CS-0"
	TestDeviceSoftwareVersion          = "1.0.1-rc1"
	TestDeviceType                     device.Type

	TestResourceSwitchesInstanceResourceTypes = []string{types.BINARY_SWITCH}
	TestResourceSwitchesResourceTypes         = []string{collection.ResourceType}
	TestResourceLightInstanceResourceTypes    = []string{types.CORE_LIGHT}
	TestResourceDeviceResourceTypes           = []string{types.DEVICE_CLOUD, schemaDevice.ResourceType}

	testIovityLiteVersion *sync.Map[string, uint32]
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

func TestBridgeDeviceInstanceName(id string) string {
	return "bridged-device-" + id
}

type LightResourceRepresentation struct {
	Name  string `json:"name"`
	Power uint64 `json:"power"`
	State bool   `json:"state"`
}

type SwitchResourceRepresentation struct {
	Value bool `json:"value"`
}

type MaintenanceResourceRepresentation struct {
	FactoryReset bool `json:"fr"`
}

type PlgdTimeResourceRepresentation struct {
	Time           string `json:"time"`
	LastSyncedTime string `json:"lastSyncedTime"`
}

type CollectionLinkRepresentation struct {
	Href           string      `json:"href"`
	Representation interface{} `json:"rep"`
}

type CollectionLinkRepresentations []CollectionLinkRepresentation

func init() {
	if name := os.Getenv("TEST_DEVICE_NAME"); name != "" {
		TestDeviceName = name
	} else {
		TestDeviceName = "devsim-" + MustGetHostname()
	}
	TestDeviceNameWithOicResObservable = "devsim-resobs-" + MustGetHostname()
	if dtype := os.Getenv("TEST_DEVICE_TYPE"); dtype == "bridged" {
		TestDeviceType = device.Bridged
	} else {
		TestDeviceType = device.OCF
	}

	testIovityLiteVersion = sync.NewMap[string, uint32]()
}

func GetDeviceResourceRepresentation(deviceID, deviceName string) schemaDevice.Device {
	return schemaDevice.Device{
		ID:                   deviceID,
		Interfaces:           []string{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
		Name:                 deviceName,
		ResourceTypes:        TestResourceDeviceResourceTypes,
		DataModelVersion:     "ocf.res.1.3.0",
		SpecificationVersion: "ocf.2.0.5",
		ModelNumber:          TestDeviceModelNumber,
		SoftwareVersion:      TestDeviceSoftwareVersion,
	}
}

type ResourceLinkRepresentations []ResourceLinkRepresentation

func (r ResourceLinkRepresentations) Sort() ResourceLinkRepresentations {
	sort.Slice(r, func(i, j int) bool {
		return r[i].Href < r[j].Href
	})
	return r
}

func GetAllBackendResourceRepresentations(t *testing.T, deviceID, deviceName string) ResourceLinkRepresentations {
	dev := GetDeviceResourceRepresentation(deviceID, deviceName)
	dev.Interfaces = nil
	dev.ResourceTypes = nil
	iotVersion := GetIotivityLiteVersion(t, deviceID)
	return ResourceLinkRepresentations{
		{
			Href: "/" + commands.NewResourceID(deviceID, TestResourceLightInstanceHref("1")).ToString(),
			Representation: LightResourceRepresentation{
				Name:  "Light",
				Power: 0,
				State: false,
			},
			ResourceTypes: TestResourceLightInstanceResourceTypes,
		},
		{
			Href: "/" + commands.NewResourceID(deviceID, configuration.ResourceURI).ToString(),
			Representation: configuration.Configuration{
				Name: deviceName,
			},
			ResourceTypes: []string{configuration.ResourceType},
		},
		{
			Href:           "/" + commands.NewResourceID(deviceID, schemaDevice.ResourceURI).ToString(),
			Representation: dev,
			ResourceTypes:  TestResourceDeviceResourceTypes,
		},
		{
			Href:           "/" + commands.NewResourceID(deviceID, maintenance.ResourceURI).ToString(),
			Representation: MaintenanceResourceRepresentation{},
			ResourceTypes:  []string{maintenance.ResourceType},
		},
		{
			Href: "/" + commands.NewResourceID(deviceID, platform.ResourceURI).ToString(),
			Representation: platform.Platform{
				ManufacturerName: "ocfcloud.com",
				Version:          iotVersion,
			},
			ResourceTypes: []string{platform.ResourceType},
		},
		{
			Href:           "/" + commands.NewResourceID(deviceID, TestResourceSwitchesHref).ToString(),
			Representation: schema.ResourceLinks{},
			ResourceTypes:  []string{collection.ResourceType},
		},
		{
			Href:           "/" + commands.NewResourceID(deviceID, plgdtime.ResourceURI).ToString(),
			Representation: PlgdTimeResourceRepresentation{},
			ResourceTypes:  []string{plgdtime.ResourceType},
		},
		{
			Href: "/" + commands.NewResourceID(deviceID, softwareupdate.ResourceURI).ToString(),
			Representation: map[interface{}]interface{}{
				"purl":           "",
				"nv":             "",
				"signed":         "vendor",
				"swupdateaction": "idle",
				"swupdatestate":  "idle",
				"swupdateresult": uint64(0),
				"updatetime":     "1970-01-01T00:00:00Z",
			},
			ResourceTypes: []string{softwareupdate.ResourceType},
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
		ResourceTypes: TestResourceSwitchesInstanceResourceTypes,
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

func StringToApplicationProtocol(p string) commands.Connection_Protocol {
	switch p {
	case string(schema.TCPSecureScheme):
		return commands.Connection_COAPS_TCP
	case string(schema.UDPSecureScheme):
		return commands.Connection_COAPS
	case string(schema.TCPScheme):
		return commands.Connection_COAP_TCP
	case string(schema.UDPScheme):
		return commands.Connection_COAP
	}
	return commands.Connection_UNKNOWN
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

func setAccessForCloud(ctx context.Context, c *deviceClient.Client, deviceID, cloudSID string) error {
	d, links, err := c.GetDevice(ctx, deviceID)
	if err != nil {
		return err
	}
	defer func() {
		_ = d.Close(ctx)
	}()

	// skip setting of ACLs for insecure devices
	if !d.IsSecured() {
		return nil
	}

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
	confResources := acl.AllResources
	for _, href := range links.GetResourceHrefs(maintenance.ResourceType) {
		confResources = append(confResources, acl.Resource{
			Href:       href,
			Interfaces: []string{"*"},
		})
	}
	for _, href := range links.GetResourceHrefs(plgdtime.ResourceType) {
		confResources = append(confResources, acl.Resource{
			Href:       href,
			Interfaces: []string{"*"},
		})
	}
	setAcl := acl.UpdateRequest{
		AccessControlList: []acl.AccessControl{
			{
				Permission: acl.AllPermissions,
				Subject: acl.Subject{
					Subject_Device: &acl.Subject_Device{
						DeviceID: cloudSID,
					},
				},
				Resources: confResources,
			},
		},
	}

	return p.UpdateResource(ctx, link, setAcl, nil)
}

func disownDevice(t *testing.T, d device.Device) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()
	client, err := sdk.NewClient(d.GetSDKClientOptions()...)
	require.NoError(t, err)
	defer func() {
		_ = client.Close(ctx)
	}()
	err = client.DisownDevice(ctx, d.GetID())
	require.NoError(t, err)
	time.Sleep(time.Second * 2)
}

func OnboardDevice(ctx context.Context, t *testing.T, c pb.GrpcGatewayClient, d device.Device, hubEndpoint string, expectedResources []schema.ResourceLink) func() {
	return OnboardDeviceForClient(ctx, t, c, d, config.OAUTH_MANAGER_CLIENT_ID, hubEndpoint, expectedResources)
}

func OnboardDeviceForClient(ctx context.Context, t *testing.T, c pb.GrpcGatewayClient, d device.Device, clientID, hubEndpoint string, expectedResources []schema.ResourceLink) func() {
	cloudSID := config.HubID()
	require.NotEmpty(t, cloudSID)
	devClient, err := sdk.NewClient(d.GetSDKClientOptions()...)
	require.NoError(t, err)
	defer func() {
		_ = devClient.Close(ctx)
	}()

	deviceID, err := devClient.OwnDevice(ctx, d.GetID(), deviceClient.WithOTM(deviceClient.OTMType_JustWorks))
	require.NoError(t, err)
	d.SetID(deviceID)

	err = setAccessForCloud(ctx, devClient, d.GetID(), cloudSID)
	require.NoError(t, err)

	code := oauthTest.GetAuthorizationCode(t, config.OAUTH_SERVER_HOST, clientID, d.GetID(), "")

	onboard := func() {
		var cloudRes cloud.Configuration
		err = devClient.GetResource(ctx, d.GetID(), cloud.ResourceURI, &cloudRes)
		require.NoError(t, err)

		if cloudRes.ProvisioningStatus != cloud.ProvisioningStatus_UNINITIALIZED {
			// device cloud is configured so we need to remove it first
			err = devClient.OffboardDevice(ctx, d.GetID())
			require.NoError(t, err)
		}

		err = devClient.OnboardDevice(ctx, d.GetID(), config.DEVICE_PROVIDER, hubEndpoint, code, cloudSID)
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
			SubscriptionId: ev.GetSubscriptionId(),
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
		WaitForDevice(t, subClient, d, ev.GetSubscriptionId(), ev.GetCorrelationId(), expectedResources)
		err = subClient.CloseSend()
		require.NoError(t, err)
	} else {
		onboard()
	}

	return func() {
		disownDevice(t, d)
	}
}

func OnboardDevSimForClient(ctx context.Context, t *testing.T, c pb.GrpcGatewayClient, clientID, deviceID, hubEndpoint string, expectedResources []schema.ResourceLink) (string, func()) {
	d := ocf.NewDevice(deviceID, TestDeviceName)
	cleanup := OnboardDeviceForClient(ctx, t, c, d, clientID, hubEndpoint, expectedResources)
	return d.GetID(), cleanup
}

func OnboardDevSim(ctx context.Context, t *testing.T, c pb.GrpcGatewayClient, deviceID, hubEndpoint string, expectedResources []schema.ResourceLink) (string, func()) {
	return OnboardDevSimForClient(ctx, t, c, config.OAUTH_MANAGER_CLIENT_ID, deviceID, hubEndpoint, expectedResources)
}

func OffboardDevice(ctx context.Context, t *testing.T, d device.Device) {
	devClient, err := sdk.NewClient(d.GetSDKClientOptions()...)
	require.NoError(t, err)
	defer func() {
		_ = devClient.Close(ctx)
	}()
	err = devClient.OffboardDevice(ctx, d.GetID())
	require.NoError(t, err)
}

func OffBoardDevSim(ctx context.Context, t *testing.T, deviceID string) {
	OffboardDevice(ctx, t, ocf.NewDevice(deviceID, TestDeviceName))
}

func WaitForDevice(t *testing.T, client pb.GrpcGateway_SubscribeToEventsClient, dev device.Device, subID, correlationID string, expectedResources []schema.ResourceLink) {
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
				require.NotEmpty(t, val.DeviceMetadataUpdated.GetConnection().GetServiceId())
				val.DeviceMetadataUpdated.GetConnection().ServiceId = ""
				if dev.GetType() != device.Bridged && !config.COAP_GATEWAY_UDP_ENABLED {
					// TODO: fix bug in iotivity-lite, for DTLS it is not fill endpoints
					require.NotEmpty(t, val.DeviceMetadataUpdated.GetConnection().GetLocalEndpoints())
				}
				val.DeviceMetadataUpdated.GetConnection().LocalEndpoints = nil
			}
			if val.DeviceMetadataUpdated.GetTwinSynchronization() != nil {
				val.DeviceMetadataUpdated.GetTwinSynchronization().CommandMetadata = nil
				val.DeviceMetadataUpdated.GetTwinSynchronization().SyncingAt = 0
				val.DeviceMetadataUpdated.GetTwinSynchronization().InSyncAt = 0
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
			val.ResourceChanged.Etag = nil
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
					DeviceIds: []string{dev.GetID()},
					EventMetadata: &isEvents.EventMetadata{
						HubId: config.HubID(),
					},
				},
			},
		},
		getID(&pb.Event{Type: &pb.Event_DeviceMetadataUpdated{
			DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
				Connection: &commands.Connection{
					Status:   commands.Connection_ONLINE,
					Protocol: StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME),
				},
				TwinEnabled: true,
			},
		}}): {
			SubscriptionId: subID,
			CorrelationId:  correlationID,
			Type: &pb.Event_DeviceMetadataUpdated{
				DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
					DeviceId: dev.GetID(),
					Connection: &commands.Connection{
						Status:   commands.Connection_ONLINE,
						Protocol: StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME),
					},
					TwinSynchronization: &commands.TwinSynchronization{
						State: commands.TwinSynchronization_OUT_OF_SYNC,
					},
					TwinEnabled: true,
				},
			},
		},
		getID(&pb.Event{Type: &pb.Event_DeviceMetadataUpdated{
			DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
				Connection: &commands.Connection{
					Status:   commands.Connection_ONLINE,
					Protocol: StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME),
				},
				TwinSynchronization: &commands.TwinSynchronization{
					State: commands.TwinSynchronization_SYNCING,
				},
				TwinEnabled: true,
			},
		}}): {
			SubscriptionId: subID,
			CorrelationId:  correlationID,
			Type: &pb.Event_DeviceMetadataUpdated{
				DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
					DeviceId: dev.GetID(),
					Connection: &commands.Connection{
						Status:   commands.Connection_ONLINE,
						Protocol: StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME),
					},
					TwinSynchronization: &commands.TwinSynchronization{
						State: commands.TwinSynchronization_SYNCING,
					},
					TwinEnabled: true,
				},
			},
		},
		getID(&pb.Event{Type: &pb.Event_DeviceMetadataUpdated{
			DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
				Connection: &commands.Connection{
					Status:   commands.Connection_ONLINE,
					Protocol: StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME),
				},
				TwinSynchronization: &commands.TwinSynchronization{
					State: commands.TwinSynchronization_IN_SYNC,
				},
				TwinEnabled: true,
			},
		}}): {
			SubscriptionId: subID,
			CorrelationId:  correlationID,
			Type: &pb.Event_DeviceMetadataUpdated{
				DeviceMetadataUpdated: &events.DeviceMetadataUpdated{
					DeviceId: dev.GetID(),
					Connection: &commands.Connection{
						Status:   commands.Connection_ONLINE,
						Protocol: StringToApplicationProtocol(config.ACTIVE_COAP_SCHEME),
					},
					TwinSynchronization: &commands.TwinSynchronization{
						State: commands.TwinSynchronization_IN_SYNC,
					},
					TwinEnabled: true,
				},
			},
		},
		getID(&pb.Event{Type: &pb.Event_ResourcePublished{}}): {
			SubscriptionId: subID,
			CorrelationId:  correlationID,
			Type: &pb.Event_ResourcePublished{
				ResourcePublished: &events.ResourceLinksPublished{
					DeviceId:  dev.GetID(),
					Resources: ResourceLinksToResources(dev.GetID(), expectedResources),
				},
			},
		},
	}
	for _, r := range expectedResources {
		expectedEvents[getID(&pb.Event{Type: &pb.Event_ResourceChanged{
			ResourceChanged: &events.ResourceChanged{
				ResourceId: commands.NewResourceID(dev.GetID(), r.Href),
			},
		}})] = &pb.Event{
			SubscriptionId: subID,
			CorrelationId:  correlationID,
			Type: &pb.Event_ResourceChanged{
				ResourceChanged: &events.ResourceChanged{
					ResourceId:    commands.NewResourceID(dev.GetID(), r.Href),
					Status:        commands.Status_OK,
					ResourceTypes: r.ResourceTypes,
				},
			},
		}
	}

	for {
		ev, err := client.Recv()
		if err != nil {
			fmt.Printf("expectedEvents: %+v", expectedEvents)
		}
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
			break
		}
	}

	// wait for sync RD
	time.Sleep(500 * time.Millisecond)
}

func MustGetHostname() string {
	n, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	return n
}

func MustFindDeviceByName(name string, getResourceOpts ...device.GetResourceOpts) (deviceID string) {
	var err error
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		deviceID, err = device.FindDeviceByName(ctx, name, getResourceOpts...)
		if err == nil {
			return deviceID
		}
	}
	panic(err)
}

func GetBridgeDeviceConfig() (bridgeDevice.Config, error) {
	cfgFile := os.Getenv("TEST_BRIDGE_DEVICE_CONFIG")
	if cfgFile == "" {
		return bridgeDevice.Config{}, errors.New("TEST_BRIDGE_DEVICE_CONFIG not set")
	}
	return bridgeDevice.LoadConfig(cfgFile)
}

func MustFindTestDevice() device.Device {
	var getResourceOpts []device.GetResourceOpts
	if TestDeviceType == device.Bridged {
		getResourceOpts = append(getResourceOpts, func(d *core.Device) deviceCoap.OptionFunc {
			return deviceCoap.WithQuery("di=" + d.DeviceID())
		})
	}

	var deviceID string
	var err error
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		deviceID, err = device.FindDeviceByName(ctx, TestDeviceName, getResourceOpts...)
		if err == nil {
			break
		}
	}

	if err != nil {
		panic(err)
	}

	if TestDeviceType == device.Bridged {
		bridgeDeviceCfg, err := GetBridgeDeviceConfig()
		if err != nil {
			panic(err)
		}
		return bridge.NewDevice(deviceID, TestDeviceName, bridgeDeviceCfg.NumResourcesPerDevice, true)
	}
	return ocf.NewDevice(deviceID, TestDeviceName)
}

func IsDiscoveryResourceBatchObservable(ctx context.Context, t *testing.T, deviceID string) bool {
	devClient, err := sdk.NewClient()
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

func GetResource(ctx context.Context, deviceID, resourceURI, resourceType string, resp interface{}, opts ...deviceClient.GetOption) error {
	devClient, err := sdk.NewClient()
	defer func() {
		_ = devClient.Close(ctx)
	}()
	if err != nil {
		return err
	}
	opts = append(opts, deviceClient.WithResourceTypes(resourceType))
	err = devClient.GetResource(ctx, deviceID, resourceURI, resp, opts...)
	if err != nil {
		return err
	}
	return nil
}

func GetIotivityLiteVersion(t *testing.T, deviceID string) uint32 {
	version, ok := testIovityLiteVersion.Load(deviceID)
	if ok {
		return version
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	plt := platform.Platform{}
	err := GetResource(ctx, deviceID, platform.ResourceURI, platform.ResourceType, &plt)
	require.NoError(t, err)
	testIovityLiteVersion.Store(deviceID, plt.Version)
	return plt.Version
}

func DeviceIsBatchObservable(ctx context.Context, t *testing.T, deviceID string) bool {
	var links schema.ResourceLinks
	err := GetResource(ctx, deviceID, resources.ResourceURI, resources.ResourceType, &links, deviceClient.WithQuery("di="+deviceID))
	require.NoError(t, err)
	require.Len(t, links, 1)
	return links[0].Policy.BitMask.Has(schema.Observable) &&
		slices.Contains(links[0].Interfaces, interfaces.OC_IF_B)
}

func GetAllBackendResourceLinks() schema.ResourceLinks {
	return ocf.TestResources
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
		checFunc(t, expected, actual, msgAndArgs...)
	}
}

func AssertToCheckFunc(checFunc func(t assert.TestingT, expected interface{}, actual interface{}, msgAndArgs ...interface{}) bool) func(t *testing.T, expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
	return func(t *testing.T, expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
		checFunc(t, expected, actual, msgAndArgs...)
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

func GenerateIDbyIdx(prefix string, deviceIndex int) string {
	if len(prefix) == 0 {
		prefix = "0"
	}
	if len(prefix) > 1 {
		prefix = prefix[:1]
	}
	return fmt.Sprintf("%s0000000-0000-0000-0000-%012v", prefix, deviceIndex)
}

func GenerateDeviceIDbyIdx(deviceIndex int) string {
	return GenerateIDbyIdx("d", deviceIndex)
}

type ListenSocket struct {
	Network string
	Address string
}

func (ls *ListenSocket) IsClosed(t require.TestingT) bool {
	if strings.Contains(ls.Network, "udp") {
		addr, err := net.ResolveUDPAddr(ls.Network, ls.Address)
		require.NoError(t, err)
		c, err := net.ListenUDP(ls.Network, addr)
		if err != nil {
			return false
		}
		err = c.Close()
		require.NoError(t, err)
		return true
	}

	addr, err := net.ResolveTCPAddr(ls.Network, ls.Address)
	require.NoError(t, err)
	c, err := net.ListenTCP(ls.Network, addr)
	if err != nil {
		return false
	}
	err = c.Close()
	require.NoError(t, err)
	return true
}

type ListenSockets []ListenSocket

func (ls ListenSockets) CheckForClosedSockets(t require.TestingT) {
	// wait for all sockets to be closed - max 3 minutes = 900*200
	socketClosed := make([]bool, len(ls))
	for j := 0; j < 900; j++ {
		allClosed := true
		for i, socket := range ls {
			if socketClosed[i] {
				continue
			}
			closed := socket.IsClosed(t)
			socketClosed[i] = closed
			if socketClosed[i] {
				continue
			}
			allClosed = false
		}
		if allClosed {
			return
		}
		time.Sleep(time.Millisecond * 200)
	}
	require.FailNow(t, "ports not closed")
}

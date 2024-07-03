// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.1
// 	protoc        v5.26.1
// source: grpc-gateway/pb/hubConfiguration.proto

package pb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type HubConfigurationRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *HubConfigurationRequest) Reset() {
	*x = HubConfigurationRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_grpc_gateway_pb_hubConfiguration_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HubConfigurationRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HubConfigurationRequest) ProtoMessage() {}

func (x *HubConfigurationRequest) ProtoReflect() protoreflect.Message {
	mi := &file_grpc_gateway_pb_hubConfiguration_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HubConfigurationRequest.ProtoReflect.Descriptor instead.
func (*HubConfigurationRequest) Descriptor() ([]byte, []int) {
	return file_grpc_gateway_pb_hubConfiguration_proto_rawDescGZIP(), []int{0}
}

type OAuthClient struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ClientId            string   `protobuf:"bytes,1,opt,name=client_id,json=clientId,proto3" json:"client_id,omitempty" yaml:"clientID"` // @gotags: yaml:"clientID"
	Audience            string   `protobuf:"bytes,2,opt,name=audience,proto3" json:"audience,omitempty"`
	Scopes              []string `protobuf:"bytes,3,rep,name=scopes,proto3" json:"scopes,omitempty"`
	ProviderName        string   `protobuf:"bytes,4,opt,name=provider_name,json=providerName,proto3" json:"provider_name,omitempty" yaml:"providerName"`                        // @gotags: yaml:"providerName"
	ClientAssertionType string   `protobuf:"bytes,5,opt,name=client_assertion_type,json=clientAssertionType,proto3" json:"client_assertion_type,omitempty" yaml:"clientAssertionType"` // @gotags: yaml:"clientAssertionType"
	Authority           string   `protobuf:"bytes,6,opt,name=authority,proto3" json:"authority,omitempty"`
	GrantType           string   `protobuf:"bytes,7,opt,name=grant_type,json=grantType,proto3" json:"grant_type,omitempty" yaml:"grantType"` // @gotags: yaml:"grantType"
}

func (x *OAuthClient) Reset() {
	*x = OAuthClient{}
	if protoimpl.UnsafeEnabled {
		mi := &file_grpc_gateway_pb_hubConfiguration_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *OAuthClient) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OAuthClient) ProtoMessage() {}

func (x *OAuthClient) ProtoReflect() protoreflect.Message {
	mi := &file_grpc_gateway_pb_hubConfiguration_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OAuthClient.ProtoReflect.Descriptor instead.
func (*OAuthClient) Descriptor() ([]byte, []int) {
	return file_grpc_gateway_pb_hubConfiguration_proto_rawDescGZIP(), []int{1}
}

func (x *OAuthClient) GetClientId() string {
	if x != nil {
		return x.ClientId
	}
	return ""
}

func (x *OAuthClient) GetAudience() string {
	if x != nil {
		return x.Audience
	}
	return ""
}

func (x *OAuthClient) GetScopes() []string {
	if x != nil {
		return x.Scopes
	}
	return nil
}

func (x *OAuthClient) GetProviderName() string {
	if x != nil {
		return x.ProviderName
	}
	return ""
}

func (x *OAuthClient) GetClientAssertionType() string {
	if x != nil {
		return x.ClientAssertionType
	}
	return ""
}

func (x *OAuthClient) GetAuthority() string {
	if x != nil {
		return x.Authority
	}
	return ""
}

func (x *OAuthClient) GetGrantType() string {
	if x != nil {
		return x.GrantType
	}
	return ""
}

type BuildInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// version of the service
	Version string `protobuf:"bytes,1,opt,name=version,proto3" json:"version,omitempty"`
	// build date of the service
	BuildDate string `protobuf:"bytes,2,opt,name=build_date,json=buildDate,proto3" json:"build_date,omitempty"`
	// commit hash of the service
	CommitHash string `protobuf:"bytes,3,opt,name=commit_hash,json=commitHash,proto3" json:"commit_hash,omitempty"`
	// commit date of the service
	CommitDate string `protobuf:"bytes,4,opt,name=commit_date,json=commitDate,proto3" json:"commit_date,omitempty"`
	// release url of the service
	ReleaseUrl string `protobuf:"bytes,5,opt,name=release_url,json=releaseUrl,proto3" json:"release_url,omitempty"`
}

func (x *BuildInfo) Reset() {
	*x = BuildInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_grpc_gateway_pb_hubConfiguration_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BuildInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BuildInfo) ProtoMessage() {}

func (x *BuildInfo) ProtoReflect() protoreflect.Message {
	mi := &file_grpc_gateway_pb_hubConfiguration_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BuildInfo.ProtoReflect.Descriptor instead.
func (*BuildInfo) Descriptor() ([]byte, []int) {
	return file_grpc_gateway_pb_hubConfiguration_proto_rawDescGZIP(), []int{2}
}

func (x *BuildInfo) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

func (x *BuildInfo) GetBuildDate() string {
	if x != nil {
		return x.BuildDate
	}
	return ""
}

func (x *BuildInfo) GetCommitHash() string {
	if x != nil {
		return x.CommitHash
	}
	return ""
}

func (x *BuildInfo) GetCommitDate() string {
	if x != nil {
		return x.CommitDate
	}
	return ""
}

func (x *BuildInfo) GetReleaseUrl() string {
	if x != nil {
		return x.ReleaseUrl
	}
	return ""
}

// UI visibility configuration
// If true - show UI element, if false - hide UI element
type UIVisibility struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Main sidebar visibility
	MainSidebar *UIVisibility_MainSidebar `protobuf:"bytes,1,opt,name=main_sidebar,json=mainSidebar,proto3" json:"main_sidebar,omitempty"`
}

func (x *UIVisibility) Reset() {
	*x = UIVisibility{}
	if protoimpl.UnsafeEnabled {
		mi := &file_grpc_gateway_pb_hubConfiguration_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UIVisibility) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UIVisibility) ProtoMessage() {}

func (x *UIVisibility) ProtoReflect() protoreflect.Message {
	mi := &file_grpc_gateway_pb_hubConfiguration_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UIVisibility.ProtoReflect.Descriptor instead.
func (*UIVisibility) Descriptor() ([]byte, []int) {
	return file_grpc_gateway_pb_hubConfiguration_proto_rawDescGZIP(), []int{3}
}

func (x *UIVisibility) GetMainSidebar() *UIVisibility_MainSidebar {
	if x != nil {
		return x.MainSidebar
	}
	return nil
}

// UI configuration
type UIConfiguration struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Visibility *UIVisibility `protobuf:"bytes,1,opt,name=visibility,proto3" json:"visibility,omitempty"`
	// Address to device provisioning service HTTP API in format https://host:port
	DeviceProvisioningService string `protobuf:"bytes,2,opt,name=device_provisioning_service,json=deviceProvisioningService,proto3" json:"device_provisioning_service,omitempty"`
	// Address to snippet service HTTP API in format https://host:port
	SnippetService string `protobuf:"bytes,3,opt,name=snippet_service,json=snippetService,proto3" json:"snippet_service,omitempty"`
}

func (x *UIConfiguration) Reset() {
	*x = UIConfiguration{}
	if protoimpl.UnsafeEnabled {
		mi := &file_grpc_gateway_pb_hubConfiguration_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UIConfiguration) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UIConfiguration) ProtoMessage() {}

func (x *UIConfiguration) ProtoReflect() protoreflect.Message {
	mi := &file_grpc_gateway_pb_hubConfiguration_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UIConfiguration.ProtoReflect.Descriptor instead.
func (*UIConfiguration) Descriptor() ([]byte, []int) {
	return file_grpc_gateway_pb_hubConfiguration_proto_rawDescGZIP(), []int{4}
}

func (x *UIConfiguration) GetVisibility() *UIVisibility {
	if x != nil {
		return x.Visibility
	}
	return nil
}

func (x *UIConfiguration) GetDeviceProvisioningService() string {
	if x != nil {
		return x.DeviceProvisioningService
	}
	return ""
}

func (x *UIConfiguration) GetSnippetService() string {
	if x != nil {
		return x.SnippetService
	}
	return ""
}

type HubConfigurationResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// provides a current time of server in nanoseconds.
	CurrentTime            int64  `protobuf:"varint,1,opt,name=current_time,json=currentTime,proto3" json:"current_time,omitempty"`
	JwtOwnerClaim          string `protobuf:"bytes,2,opt,name=jwt_owner_claim,json=jwtOwnerClaim,proto3" json:"jwt_owner_claim,omitempty"`
	JwtDeviceIdClaim       string `protobuf:"bytes,3,opt,name=jwt_device_id_claim,json=jwtDeviceIdClaim,proto3" json:"jwt_device_id_claim,omitempty"`
	Id                     string `protobuf:"bytes,4,opt,name=id,proto3" json:"id,omitempty"`
	CoapGateway            string `protobuf:"bytes,5,opt,name=coap_gateway,json=coapGateway,proto3" json:"coap_gateway,omitempty"`
	CertificateAuthorities string `protobuf:"bytes,6,opt,name=certificate_authorities,json=certificateAuthorities,proto3" json:"certificate_authorities,omitempty"`
	Authority              string `protobuf:"bytes,7,opt,name=authority,proto3" json:"authority,omitempty"`
	// exposes default command time to live in nanoseconds for CreateResource, RetrieveResource, UpdateResource, DeleteResource, and UpdateDeviceMetadata commands when it is not set in the request. 0 - means forever.
	DefaultCommandTimeToLive int64 `protobuf:"varint,8,opt,name=default_command_time_to_live,json=defaultCommandTimeToLive,proto3" json:"default_command_time_to_live,omitempty"`
	// certificate_authority in format https://host:port
	CertificateAuthority string `protobuf:"bytes,9,opt,name=certificate_authority,json=certificateAuthority,proto3" json:"certificate_authority,omitempty"`
	// cfg for UI http-gateway
	HttpGatewayAddress string           `protobuf:"bytes,10,opt,name=http_gateway_address,json=httpGatewayAddress,proto3" json:"http_gateway_address,omitempty"`
	WebOauthClient     *OAuthClient     `protobuf:"bytes,11,opt,name=web_oauth_client,json=webOauthClient,proto3" json:"web_oauth_client,omitempty"`
	DeviceOauthClient  *OAuthClient     `protobuf:"bytes,12,opt,name=device_oauth_client,json=deviceOauthClient,proto3" json:"device_oauth_client,omitempty"`
	M2MOauthClient     *OAuthClient     `protobuf:"bytes,15,opt,name=m2m_oauth_client,json=m2mOauthClient,proto3" json:"m2m_oauth_client,omitempty"`
	Ui                 *UIConfiguration `protobuf:"bytes,14,opt,name=ui,proto3" json:"ui,omitempty"`
	// build info
	BuildInfo *BuildInfo `protobuf:"bytes,13,opt,name=build_info,json=buildInfo,proto3" json:"build_info,omitempty"`
}

func (x *HubConfigurationResponse) Reset() {
	*x = HubConfigurationResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_grpc_gateway_pb_hubConfiguration_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HubConfigurationResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HubConfigurationResponse) ProtoMessage() {}

func (x *HubConfigurationResponse) ProtoReflect() protoreflect.Message {
	mi := &file_grpc_gateway_pb_hubConfiguration_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HubConfigurationResponse.ProtoReflect.Descriptor instead.
func (*HubConfigurationResponse) Descriptor() ([]byte, []int) {
	return file_grpc_gateway_pb_hubConfiguration_proto_rawDescGZIP(), []int{5}
}

func (x *HubConfigurationResponse) GetCurrentTime() int64 {
	if x != nil {
		return x.CurrentTime
	}
	return 0
}

func (x *HubConfigurationResponse) GetJwtOwnerClaim() string {
	if x != nil {
		return x.JwtOwnerClaim
	}
	return ""
}

func (x *HubConfigurationResponse) GetJwtDeviceIdClaim() string {
	if x != nil {
		return x.JwtDeviceIdClaim
	}
	return ""
}

func (x *HubConfigurationResponse) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *HubConfigurationResponse) GetCoapGateway() string {
	if x != nil {
		return x.CoapGateway
	}
	return ""
}

func (x *HubConfigurationResponse) GetCertificateAuthorities() string {
	if x != nil {
		return x.CertificateAuthorities
	}
	return ""
}

func (x *HubConfigurationResponse) GetAuthority() string {
	if x != nil {
		return x.Authority
	}
	return ""
}

func (x *HubConfigurationResponse) GetDefaultCommandTimeToLive() int64 {
	if x != nil {
		return x.DefaultCommandTimeToLive
	}
	return 0
}

func (x *HubConfigurationResponse) GetCertificateAuthority() string {
	if x != nil {
		return x.CertificateAuthority
	}
	return ""
}

func (x *HubConfigurationResponse) GetHttpGatewayAddress() string {
	if x != nil {
		return x.HttpGatewayAddress
	}
	return ""
}

func (x *HubConfigurationResponse) GetWebOauthClient() *OAuthClient {
	if x != nil {
		return x.WebOauthClient
	}
	return nil
}

func (x *HubConfigurationResponse) GetDeviceOauthClient() *OAuthClient {
	if x != nil {
		return x.DeviceOauthClient
	}
	return nil
}

func (x *HubConfigurationResponse) GetM2MOauthClient() *OAuthClient {
	if x != nil {
		return x.M2MOauthClient
	}
	return nil
}

func (x *HubConfigurationResponse) GetUi() *UIConfiguration {
	if x != nil {
		return x.Ui
	}
	return nil
}

func (x *HubConfigurationResponse) GetBuildInfo() *BuildInfo {
	if x != nil {
		return x.BuildInfo
	}
	return nil
}

type UIVisibility_MainSidebar struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Devices              bool `protobuf:"varint,1,opt,name=devices,proto3" json:"devices,omitempty"`
	Configuration        bool `protobuf:"varint,2,opt,name=configuration,proto3" json:"configuration,omitempty"`
	RemoteClients        bool `protobuf:"varint,3,opt,name=remote_clients,json=remoteClients,proto3" json:"remote_clients,omitempty"`
	PendingCommands      bool `protobuf:"varint,4,opt,name=pending_commands,json=pendingCommands,proto3" json:"pending_commands,omitempty"`
	Certificates         bool `protobuf:"varint,5,opt,name=certificates,proto3" json:"certificates,omitempty"`
	DeviceProvisioning   bool `protobuf:"varint,6,opt,name=device_provisioning,json=deviceProvisioning,proto3" json:"device_provisioning,omitempty"`
	Docs                 bool `protobuf:"varint,7,opt,name=docs,proto3" json:"docs,omitempty"`
	ChatRoom             bool `protobuf:"varint,8,opt,name=chat_room,json=chatRoom,proto3" json:"chat_room,omitempty"`
	Dashboard            bool `protobuf:"varint,9,opt,name=dashboard,proto3" json:"dashboard,omitempty"`
	Integrations         bool `protobuf:"varint,10,opt,name=integrations,proto3" json:"integrations,omitempty"`
	DeviceFirmwareUpdate bool `protobuf:"varint,11,opt,name=device_firmware_update,json=deviceFirmwareUpdate,proto3" json:"device_firmware_update,omitempty"`
	DeviceLogs           bool `protobuf:"varint,12,opt,name=device_logs,json=deviceLogs,proto3" json:"device_logs,omitempty"`
	ApiTokens            bool `protobuf:"varint,13,opt,name=api_tokens,json=apiTokens,proto3" json:"api_tokens,omitempty"`
	SchemaHub            bool `protobuf:"varint,14,opt,name=schema_hub,json=schemaHub,proto3" json:"schema_hub,omitempty"`
}

func (x *UIVisibility_MainSidebar) Reset() {
	*x = UIVisibility_MainSidebar{}
	if protoimpl.UnsafeEnabled {
		mi := &file_grpc_gateway_pb_hubConfiguration_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UIVisibility_MainSidebar) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UIVisibility_MainSidebar) ProtoMessage() {}

func (x *UIVisibility_MainSidebar) ProtoReflect() protoreflect.Message {
	mi := &file_grpc_gateway_pb_hubConfiguration_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UIVisibility_MainSidebar.ProtoReflect.Descriptor instead.
func (*UIVisibility_MainSidebar) Descriptor() ([]byte, []int) {
	return file_grpc_gateway_pb_hubConfiguration_proto_rawDescGZIP(), []int{3, 0}
}

func (x *UIVisibility_MainSidebar) GetDevices() bool {
	if x != nil {
		return x.Devices
	}
	return false
}

func (x *UIVisibility_MainSidebar) GetConfiguration() bool {
	if x != nil {
		return x.Configuration
	}
	return false
}

func (x *UIVisibility_MainSidebar) GetRemoteClients() bool {
	if x != nil {
		return x.RemoteClients
	}
	return false
}

func (x *UIVisibility_MainSidebar) GetPendingCommands() bool {
	if x != nil {
		return x.PendingCommands
	}
	return false
}

func (x *UIVisibility_MainSidebar) GetCertificates() bool {
	if x != nil {
		return x.Certificates
	}
	return false
}

func (x *UIVisibility_MainSidebar) GetDeviceProvisioning() bool {
	if x != nil {
		return x.DeviceProvisioning
	}
	return false
}

func (x *UIVisibility_MainSidebar) GetDocs() bool {
	if x != nil {
		return x.Docs
	}
	return false
}

func (x *UIVisibility_MainSidebar) GetChatRoom() bool {
	if x != nil {
		return x.ChatRoom
	}
	return false
}

func (x *UIVisibility_MainSidebar) GetDashboard() bool {
	if x != nil {
		return x.Dashboard
	}
	return false
}

func (x *UIVisibility_MainSidebar) GetIntegrations() bool {
	if x != nil {
		return x.Integrations
	}
	return false
}

func (x *UIVisibility_MainSidebar) GetDeviceFirmwareUpdate() bool {
	if x != nil {
		return x.DeviceFirmwareUpdate
	}
	return false
}

func (x *UIVisibility_MainSidebar) GetDeviceLogs() bool {
	if x != nil {
		return x.DeviceLogs
	}
	return false
}

func (x *UIVisibility_MainSidebar) GetApiTokens() bool {
	if x != nil {
		return x.ApiTokens
	}
	return false
}

func (x *UIVisibility_MainSidebar) GetSchemaHub() bool {
	if x != nil {
		return x.SchemaHub
	}
	return false
}

var File_grpc_gateway_pb_hubConfiguration_proto protoreflect.FileDescriptor

var file_grpc_gateway_pb_hubConfiguration_proto_rawDesc = []byte{
	0x0a, 0x26, 0x67, 0x72, 0x70, 0x63, 0x2d, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x2f, 0x70,
	0x62, 0x2f, 0x68, 0x75, 0x62, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0e, 0x67, 0x72, 0x70, 0x63, 0x67, 0x61,
	0x74, 0x65, 0x77, 0x61, 0x79, 0x2e, 0x70, 0x62, 0x22, 0x19, 0x0a, 0x17, 0x48, 0x75, 0x62, 0x43,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x22, 0xf4, 0x01, 0x0a, 0x0b, 0x4f, 0x41, 0x75, 0x74, 0x68, 0x43, 0x6c, 0x69,
	0x65, 0x6e, 0x74, 0x12, 0x1b, 0x0a, 0x09, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x5f, 0x69, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x49, 0x64,
	0x12, 0x1a, 0x0a, 0x08, 0x61, 0x75, 0x64, 0x69, 0x65, 0x6e, 0x63, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x08, 0x61, 0x75, 0x64, 0x69, 0x65, 0x6e, 0x63, 0x65, 0x12, 0x16, 0x0a, 0x06,
	0x73, 0x63, 0x6f, 0x70, 0x65, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x09, 0x52, 0x06, 0x73, 0x63,
	0x6f, 0x70, 0x65, 0x73, 0x12, 0x23, 0x0a, 0x0d, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72,
	0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x70, 0x72, 0x6f,
	0x76, 0x69, 0x64, 0x65, 0x72, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x32, 0x0a, 0x15, 0x63, 0x6c, 0x69,
	0x65, 0x6e, 0x74, 0x5f, 0x61, 0x73, 0x73, 0x65, 0x72, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x74, 0x79,
	0x70, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x13, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74,
	0x41, 0x73, 0x73, 0x65, 0x72, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x79, 0x70, 0x65, 0x12, 0x1c, 0x0a,
	0x09, 0x61, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x69, 0x74, 0x79, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x09, 0x61, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x69, 0x74, 0x79, 0x12, 0x1d, 0x0a, 0x0a, 0x67,
	0x72, 0x61, 0x6e, 0x74, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x07, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x09, 0x67, 0x72, 0x61, 0x6e, 0x74, 0x54, 0x79, 0x70, 0x65, 0x22, 0xa7, 0x01, 0x0a, 0x09, 0x42,
	0x75, 0x69, 0x6c, 0x64, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73,
	0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69,
	0x6f, 0x6e, 0x12, 0x1d, 0x0a, 0x0a, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x5f, 0x64, 0x61, 0x74, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x44, 0x61, 0x74,
	0x65, 0x12, 0x1f, 0x0a, 0x0b, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x5f, 0x68, 0x61, 0x73, 0x68,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x48, 0x61,
	0x73, 0x68, 0x12, 0x1f, 0x0a, 0x0b, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x5f, 0x64, 0x61, 0x74,
	0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x44,
	0x61, 0x74, 0x65, 0x12, 0x1f, 0x0a, 0x0b, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x75,
	0x72, 0x6c, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73,
	0x65, 0x55, 0x72, 0x6c, 0x22, 0xda, 0x04, 0x0a, 0x0c, 0x55, 0x49, 0x56, 0x69, 0x73, 0x69, 0x62,
	0x69, 0x6c, 0x69, 0x74, 0x79, 0x12, 0x4b, 0x0a, 0x0c, 0x6d, 0x61, 0x69, 0x6e, 0x5f, 0x73, 0x69,
	0x64, 0x65, 0x62, 0x61, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x28, 0x2e, 0x67, 0x72,
	0x70, 0x63, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x2e, 0x70, 0x62, 0x2e, 0x55, 0x49, 0x56,
	0x69, 0x73, 0x69, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x79, 0x2e, 0x4d, 0x61, 0x69, 0x6e, 0x53, 0x69,
	0x64, 0x65, 0x62, 0x61, 0x72, 0x52, 0x0b, 0x6d, 0x61, 0x69, 0x6e, 0x53, 0x69, 0x64, 0x65, 0x62,
	0x61, 0x72, 0x1a, 0xfc, 0x03, 0x0a, 0x0b, 0x4d, 0x61, 0x69, 0x6e, 0x53, 0x69, 0x64, 0x65, 0x62,
	0x61, 0x72, 0x12, 0x18, 0x0a, 0x07, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x73, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x08, 0x52, 0x07, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x73, 0x12, 0x24, 0x0a, 0x0d,
	0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x08, 0x52, 0x0d, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x12, 0x25, 0x0a, 0x0e, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x5f, 0x63, 0x6c, 0x69,
	0x65, 0x6e, 0x74, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0d, 0x72, 0x65, 0x6d, 0x6f,
	0x74, 0x65, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x73, 0x12, 0x29, 0x0a, 0x10, 0x70, 0x65, 0x6e,
	0x64, 0x69, 0x6e, 0x67, 0x5f, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x73, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x08, 0x52, 0x0f, 0x70, 0x65, 0x6e, 0x64, 0x69, 0x6e, 0x67, 0x43, 0x6f, 0x6d, 0x6d,
	0x61, 0x6e, 0x64, 0x73, 0x12, 0x22, 0x0a, 0x0c, 0x63, 0x65, 0x72, 0x74, 0x69, 0x66, 0x69, 0x63,
	0x61, 0x74, 0x65, 0x73, 0x18, 0x05, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0c, 0x63, 0x65, 0x72, 0x74,
	0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x65, 0x73, 0x12, 0x2f, 0x0a, 0x13, 0x64, 0x65, 0x76, 0x69,
	0x63, 0x65, 0x5f, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x69, 0x6e, 0x67, 0x18,
	0x06, 0x20, 0x01, 0x28, 0x08, 0x52, 0x12, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x50, 0x72, 0x6f,
	0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x69, 0x6e, 0x67, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x6f, 0x63,
	0x73, 0x18, 0x07, 0x20, 0x01, 0x28, 0x08, 0x52, 0x04, 0x64, 0x6f, 0x63, 0x73, 0x12, 0x1b, 0x0a,
	0x09, 0x63, 0x68, 0x61, 0x74, 0x5f, 0x72, 0x6f, 0x6f, 0x6d, 0x18, 0x08, 0x20, 0x01, 0x28, 0x08,
	0x52, 0x08, 0x63, 0x68, 0x61, 0x74, 0x52, 0x6f, 0x6f, 0x6d, 0x12, 0x1c, 0x0a, 0x09, 0x64, 0x61,
	0x73, 0x68, 0x62, 0x6f, 0x61, 0x72, 0x64, 0x18, 0x09, 0x20, 0x01, 0x28, 0x08, 0x52, 0x09, 0x64,
	0x61, 0x73, 0x68, 0x62, 0x6f, 0x61, 0x72, 0x64, 0x12, 0x22, 0x0a, 0x0c, 0x69, 0x6e, 0x74, 0x65,
	0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0c,
	0x69, 0x6e, 0x74, 0x65, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x34, 0x0a, 0x16,
	0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x5f, 0x66, 0x69, 0x72, 0x6d, 0x77, 0x61, 0x72, 0x65, 0x5f,
	0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x08, 0x52, 0x14, 0x64, 0x65,
	0x76, 0x69, 0x63, 0x65, 0x46, 0x69, 0x72, 0x6d, 0x77, 0x61, 0x72, 0x65, 0x55, 0x70, 0x64, 0x61,
	0x74, 0x65, 0x12, 0x1f, 0x0a, 0x0b, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x5f, 0x6c, 0x6f, 0x67,
	0x73, 0x18, 0x0c, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0a, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x4c,
	0x6f, 0x67, 0x73, 0x12, 0x1d, 0x0a, 0x0a, 0x61, 0x70, 0x69, 0x5f, 0x74, 0x6f, 0x6b, 0x65, 0x6e,
	0x73, 0x18, 0x0d, 0x20, 0x01, 0x28, 0x08, 0x52, 0x09, 0x61, 0x70, 0x69, 0x54, 0x6f, 0x6b, 0x65,
	0x6e, 0x73, 0x12, 0x1d, 0x0a, 0x0a, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x5f, 0x68, 0x75, 0x62,
	0x18, 0x0e, 0x20, 0x01, 0x28, 0x08, 0x52, 0x09, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x48, 0x75,
	0x62, 0x22, 0xb8, 0x01, 0x0a, 0x0f, 0x55, 0x49, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x3c, 0x0a, 0x0a, 0x76, 0x69, 0x73, 0x69, 0x62, 0x69, 0x6c,
	0x69, 0x74, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x72, 0x70, 0x63,
	0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x2e, 0x70, 0x62, 0x2e, 0x55, 0x49, 0x56, 0x69, 0x73,
	0x69, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x79, 0x52, 0x0a, 0x76, 0x69, 0x73, 0x69, 0x62, 0x69, 0x6c,
	0x69, 0x74, 0x79, 0x12, 0x3e, 0x0a, 0x1b, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x5f, 0x70, 0x72,
	0x6f, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x69, 0x6e, 0x67, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69,
	0x63, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x19, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65,
	0x50, 0x72, 0x6f, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x69, 0x6e, 0x67, 0x53, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x12, 0x27, 0x0a, 0x0f, 0x73, 0x6e, 0x69, 0x70, 0x70, 0x65, 0x74, 0x5f, 0x73,
	0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0e, 0x73, 0x6e,
	0x69, 0x70, 0x70, 0x65, 0x74, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x22, 0x8b, 0x06, 0x0a,
	0x18, 0x48, 0x75, 0x62, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x21, 0x0a, 0x0c, 0x63, 0x75, 0x72,
	0x72, 0x65, 0x6e, 0x74, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x0b, 0x63, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x54, 0x69, 0x6d, 0x65, 0x12, 0x26, 0x0a, 0x0f,
	0x6a, 0x77, 0x74, 0x5f, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x5f, 0x63, 0x6c, 0x61, 0x69, 0x6d, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x6a, 0x77, 0x74, 0x4f, 0x77, 0x6e, 0x65, 0x72, 0x43,
	0x6c, 0x61, 0x69, 0x6d, 0x12, 0x2d, 0x0a, 0x13, 0x6a, 0x77, 0x74, 0x5f, 0x64, 0x65, 0x76, 0x69,
	0x63, 0x65, 0x5f, 0x69, 0x64, 0x5f, 0x63, 0x6c, 0x61, 0x69, 0x6d, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x10, 0x6a, 0x77, 0x74, 0x44, 0x65, 0x76, 0x69, 0x63, 0x65, 0x49, 0x64, 0x43, 0x6c,
	0x61, 0x69, 0x6d, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x02, 0x69, 0x64, 0x12, 0x21, 0x0a, 0x0c, 0x63, 0x6f, 0x61, 0x70, 0x5f, 0x67, 0x61, 0x74, 0x65,
	0x77, 0x61, 0x79, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x63, 0x6f, 0x61, 0x70, 0x47,
	0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x12, 0x37, 0x0a, 0x17, 0x63, 0x65, 0x72, 0x74, 0x69, 0x66,
	0x69, 0x63, 0x61, 0x74, 0x65, 0x5f, 0x61, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x69, 0x74, 0x69, 0x65,
	0x73, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x16, 0x63, 0x65, 0x72, 0x74, 0x69, 0x66, 0x69,
	0x63, 0x61, 0x74, 0x65, 0x41, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x69, 0x74, 0x69, 0x65, 0x73, 0x12,
	0x1c, 0x0a, 0x09, 0x61, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x69, 0x74, 0x79, 0x18, 0x07, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x09, 0x61, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x69, 0x74, 0x79, 0x12, 0x3e, 0x0a,
	0x1c, 0x64, 0x65, 0x66, 0x61, 0x75, 0x6c, 0x74, 0x5f, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64,
	0x5f, 0x74, 0x69, 0x6d, 0x65, 0x5f, 0x74, 0x6f, 0x5f, 0x6c, 0x69, 0x76, 0x65, 0x18, 0x08, 0x20,
	0x01, 0x28, 0x03, 0x52, 0x18, 0x64, 0x65, 0x66, 0x61, 0x75, 0x6c, 0x74, 0x43, 0x6f, 0x6d, 0x6d,
	0x61, 0x6e, 0x64, 0x54, 0x69, 0x6d, 0x65, 0x54, 0x6f, 0x4c, 0x69, 0x76, 0x65, 0x12, 0x33, 0x0a,
	0x15, 0x63, 0x65, 0x72, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x65, 0x5f, 0x61, 0x75, 0x74,
	0x68, 0x6f, 0x72, 0x69, 0x74, 0x79, 0x18, 0x09, 0x20, 0x01, 0x28, 0x09, 0x52, 0x14, 0x63, 0x65,
	0x72, 0x74, 0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x65, 0x41, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x69,
	0x74, 0x79, 0x12, 0x30, 0x0a, 0x14, 0x68, 0x74, 0x74, 0x70, 0x5f, 0x67, 0x61, 0x74, 0x65, 0x77,
	0x61, 0x79, 0x5f, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x12, 0x68, 0x74, 0x74, 0x70, 0x47, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x41, 0x64, 0x64,
	0x72, 0x65, 0x73, 0x73, 0x12, 0x45, 0x0a, 0x10, 0x77, 0x65, 0x62, 0x5f, 0x6f, 0x61, 0x75, 0x74,
	0x68, 0x5f, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1b,
	0x2e, 0x67, 0x72, 0x70, 0x63, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x2e, 0x70, 0x62, 0x2e,
	0x4f, 0x41, 0x75, 0x74, 0x68, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x52, 0x0e, 0x77, 0x65, 0x62,
	0x4f, 0x61, 0x75, 0x74, 0x68, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x12, 0x4b, 0x0a, 0x13, 0x64,
	0x65, 0x76, 0x69, 0x63, 0x65, 0x5f, 0x6f, 0x61, 0x75, 0x74, 0x68, 0x5f, 0x63, 0x6c, 0x69, 0x65,
	0x6e, 0x74, 0x18, 0x0c, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1b, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x67,
	0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x2e, 0x70, 0x62, 0x2e, 0x4f, 0x41, 0x75, 0x74, 0x68, 0x43,
	0x6c, 0x69, 0x65, 0x6e, 0x74, 0x52, 0x11, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x4f, 0x61, 0x75,
	0x74, 0x68, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x12, 0x45, 0x0a, 0x10, 0x6d, 0x32, 0x6d, 0x5f,
	0x6f, 0x61, 0x75, 0x74, 0x68, 0x5f, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x18, 0x0f, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x1b, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79,
	0x2e, 0x70, 0x62, 0x2e, 0x4f, 0x41, 0x75, 0x74, 0x68, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x52,
	0x0e, 0x6d, 0x32, 0x6d, 0x4f, 0x61, 0x75, 0x74, 0x68, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x12,
	0x2f, 0x0a, 0x02, 0x75, 0x69, 0x18, 0x0e, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x67, 0x72,
	0x70, 0x63, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x2e, 0x70, 0x62, 0x2e, 0x55, 0x49, 0x43,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x02, 0x75, 0x69,
	0x12, 0x38, 0x0a, 0x0a, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x5f, 0x69, 0x6e, 0x66, 0x6f, 0x18, 0x0d,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x67, 0x61, 0x74, 0x65, 0x77,
	0x61, 0x79, 0x2e, 0x70, 0x62, 0x2e, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x49, 0x6e, 0x66, 0x6f, 0x52,
	0x09, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x49, 0x6e, 0x66, 0x6f, 0x42, 0x2f, 0x5a, 0x2d, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x70, 0x6c, 0x67, 0x64, 0x2d, 0x64, 0x65,
	0x76, 0x2f, 0x68, 0x75, 0x62, 0x2f, 0x76, 0x32, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x2d, 0x67, 0x61,
	0x74, 0x65, 0x77, 0x61, 0x79, 0x2f, 0x70, 0x62, 0x3b, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_grpc_gateway_pb_hubConfiguration_proto_rawDescOnce sync.Once
	file_grpc_gateway_pb_hubConfiguration_proto_rawDescData = file_grpc_gateway_pb_hubConfiguration_proto_rawDesc
)

func file_grpc_gateway_pb_hubConfiguration_proto_rawDescGZIP() []byte {
	file_grpc_gateway_pb_hubConfiguration_proto_rawDescOnce.Do(func() {
		file_grpc_gateway_pb_hubConfiguration_proto_rawDescData = protoimpl.X.CompressGZIP(file_grpc_gateway_pb_hubConfiguration_proto_rawDescData)
	})
	return file_grpc_gateway_pb_hubConfiguration_proto_rawDescData
}

var file_grpc_gateway_pb_hubConfiguration_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_grpc_gateway_pb_hubConfiguration_proto_goTypes = []interface{}{
	(*HubConfigurationRequest)(nil),  // 0: grpcgateway.pb.HubConfigurationRequest
	(*OAuthClient)(nil),              // 1: grpcgateway.pb.OAuthClient
	(*BuildInfo)(nil),                // 2: grpcgateway.pb.BuildInfo
	(*UIVisibility)(nil),             // 3: grpcgateway.pb.UIVisibility
	(*UIConfiguration)(nil),          // 4: grpcgateway.pb.UIConfiguration
	(*HubConfigurationResponse)(nil), // 5: grpcgateway.pb.HubConfigurationResponse
	(*UIVisibility_MainSidebar)(nil), // 6: grpcgateway.pb.UIVisibility.MainSidebar
}
var file_grpc_gateway_pb_hubConfiguration_proto_depIdxs = []int32{
	6, // 0: grpcgateway.pb.UIVisibility.main_sidebar:type_name -> grpcgateway.pb.UIVisibility.MainSidebar
	3, // 1: grpcgateway.pb.UIConfiguration.visibility:type_name -> grpcgateway.pb.UIVisibility
	1, // 2: grpcgateway.pb.HubConfigurationResponse.web_oauth_client:type_name -> grpcgateway.pb.OAuthClient
	1, // 3: grpcgateway.pb.HubConfigurationResponse.device_oauth_client:type_name -> grpcgateway.pb.OAuthClient
	1, // 4: grpcgateway.pb.HubConfigurationResponse.m2m_oauth_client:type_name -> grpcgateway.pb.OAuthClient
	4, // 5: grpcgateway.pb.HubConfigurationResponse.ui:type_name -> grpcgateway.pb.UIConfiguration
	2, // 6: grpcgateway.pb.HubConfigurationResponse.build_info:type_name -> grpcgateway.pb.BuildInfo
	7, // [7:7] is the sub-list for method output_type
	7, // [7:7] is the sub-list for method input_type
	7, // [7:7] is the sub-list for extension type_name
	7, // [7:7] is the sub-list for extension extendee
	0, // [0:7] is the sub-list for field type_name
}

func init() { file_grpc_gateway_pb_hubConfiguration_proto_init() }
func file_grpc_gateway_pb_hubConfiguration_proto_init() {
	if File_grpc_gateway_pb_hubConfiguration_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_grpc_gateway_pb_hubConfiguration_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HubConfigurationRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_grpc_gateway_pb_hubConfiguration_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*OAuthClient); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_grpc_gateway_pb_hubConfiguration_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BuildInfo); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_grpc_gateway_pb_hubConfiguration_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UIVisibility); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_grpc_gateway_pb_hubConfiguration_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UIConfiguration); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_grpc_gateway_pb_hubConfiguration_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HubConfigurationResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_grpc_gateway_pb_hubConfiguration_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UIVisibility_MainSidebar); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_grpc_gateway_pb_hubConfiguration_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_grpc_gateway_pb_hubConfiguration_proto_goTypes,
		DependencyIndexes: file_grpc_gateway_pb_hubConfiguration_proto_depIdxs,
		MessageInfos:      file_grpc_gateway_pb_hubConfiguration_proto_msgTypes,
	}.Build()
	File_grpc_gateway_pb_hubConfiguration_proto = out.File
	file_grpc_gateway_pb_hubConfiguration_proto_rawDesc = nil
	file_grpc_gateway_pb_hubConfiguration_proto_goTypes = nil
	file_grpc_gateway_pb_hubConfiguration_proto_depIdxs = nil
}

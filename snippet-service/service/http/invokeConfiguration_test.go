package http_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	snippetPb "github.com/plgd-dev/hub/v2/snippet-service/pb"
	snippetHttp "github.com/plgd-dev/hub/v2/snippet-service/service/http"
	snippetTest "github.com/plgd-dev/hub/v2/snippet-service/test"
	"github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	httpTest "github.com/plgd-dev/hub/v2/test/http"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestRequestHandlerInvokeConfiguration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT*100)
	defer cancel()

	shutDown := service.SetUpServices(context.Background(), t, service.SetUpServicesOAuth)
	defer shutDown()

	_, shutdownHttp := snippetTest.SetUp(t)
	defer shutdownHttp()

	conn, err := grpc.NewClient(config.SNIPPET_SERVICE_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()
	c := snippetPb.NewSnippetServiceClient(conn)

	token := oauthTest.GetDefaultAccessToken(t)
	conf, err := c.CreateConfiguration(pkgGrpc.CtxWithToken(ctx, token), &snippetPb.Configuration{
		Name:  "conf1",
		Owner: oauthService.DeviceUserID,
		Resources: []*snippetPb.Configuration_Resource{
			{
				Href: "/test/1",
				Content: &commands.Content{
					Data:              test.EncodeToCbor(t, map[string]interface{}{"value": 42}),
					ContentType:       message.AppOcfCbor.String(),
					CoapContentFormat: int32(message.AppOcfCbor),
				},
			},
			{
				Href: "/test/2",
				Content: &commands.Content{
					Data:              test.EncodeToCbor(t, map[string]interface{}{"value": 43}),
					ContentType:       message.AppOcfCbor.String(),
					CoapContentFormat: int32(message.AppOcfCbor),
				},
			},
		},
	})
	require.NoError(t, err)

	correlationID1 := uuid.NewString()
	type args struct {
		id     string
		invoke *snippetPb.InvokeConfigurationRequest
		token  string
	}
	tests := []struct {
		name         string
		args         args
		wantHTTPCode int
		wantErr      bool
		want         *snippetPb.AppliedDeviceConfiguration
	}{
		{
			name: "invoke conf1",
			args: args{
				id: conf.GetId(),
				invoke: &snippetPb.InvokeConfigurationRequest{
					ConfigurationId: conf.GetId(),
					DeviceId:        "dev1",
					CorrelationId:   correlationID1,
				},
				token: token,
			},
			wantHTTPCode: http.StatusOK,
			want: &snippetPb.AppliedDeviceConfiguration{
				DeviceId: "dev1",
				ConfigurationId: &snippetPb.AppliedDeviceConfiguration_RelationTo{
					Id:      conf.GetId(),
					Version: conf.GetVersion(),
				},
				ExecutedBy: snippetPb.MakeExecutedByOnDemand(),
				Resources: []*snippetPb.AppliedDeviceConfiguration_Resource{
					{
						Href:          "/test/1",
						Status:        snippetPb.AppliedDeviceConfiguration_Resource_PENDING,
						CorrelationId: correlationID1,
					},
					{
						Href:          "/test/2",
						Status:        snippetPb.AppliedDeviceConfiguration_Resource_PENDING,
						CorrelationId: correlationID1,
					},
				},
				Owner: oauthService.DeviceUserID,
			},
		},
		{
			name: "error - duplicit invocation of conf1",
			args: args{
				id: conf.GetId(),
				invoke: &snippetPb.InvokeConfigurationRequest{
					ConfigurationId: conf.GetId(),
					DeviceId:        "dev1",
					CorrelationId:   correlationID1,
				},
				token: token,
			},
			wantErr:      true,
			wantHTTPCode: http.StatusInternalServerError,
		},
		// TODO: force
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := httpTest.GetContentData(&pb.Content{
				ContentType: message.AppOcfCbor.String(),
				Data:        test.EncodeToCbor(t, tt.args.invoke),
			}, message.AppJSON.String())
			require.NoError(t, err)

			rb := httpTest.NewRequest(http.MethodPost, snippetTest.HTTPURI(snippetHttp.AliasConfigurations), bytes.NewReader(data)).AuthToken(tt.args.token)
			rb.Accept(pkgHttp.ApplicationProtoJsonContentType).ContentType(message.AppJSON.String()).ID(tt.args.id)
			resp := httpTest.Do(t, rb.Build(ctx, t))
			defer func() {
				_ = resp.Body.Close()
			}()
			require.Equal(t, tt.wantHTTPCode, resp.StatusCode)

			var got snippetPb.AppliedDeviceConfiguration
			err = httpTest.Unmarshal(resp.StatusCode, resp.Body, &got)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			tt.want.Id = got.GetId()
			wantCorrelationIDs := make(map[string]string)
			gotCorrelationIDs := make(map[string]string)
			for _, r := range tt.want.GetResources() {
				wantCorrelationIDs[r.GetHref()] = r.GetCorrelationId()
				r.CorrelationId = ""
			}
			for _, r := range got.GetResources() {
				gotCorrelationIDs[r.GetHref()] = r.GetCorrelationId()
				r.CorrelationId = ""
			}
			require.Len(t, wantCorrelationIDs, len(gotCorrelationIDs))
			for href, wantCorrelationID := range wantCorrelationIDs {
				gotCorrelationID, ok := gotCorrelationIDs[href]
				require.True(t, ok)
				require.True(t, strings.Contains(gotCorrelationID, wantCorrelationID))
			}
			snippetTest.CmpAppliedDeviceConfiguration(t, tt.want, &got, true)
		})
	}

	getClient, errG := c.GetAppliedConfigurations(pkgGrpc.CtxWithToken(ctx, token), &snippetPb.GetAppliedDeviceConfigurationsRequest{})
	require.NoError(t, errG)
	defer func() {
		_ = getClient.CloseSend()
	}()
	appliedConfs := make(map[string]*snippetPb.AppliedDeviceConfiguration)
	for {
		conf, errR := getClient.Recv()
		if errors.Is(errR, io.EOF) {
			break
		}
		require.NoError(t, errR)
		appliedConfs[conf.GetId()] = conf
	}
	require.Len(t, appliedConfs, 1)
}

package http_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	httpService "github.com/plgd-dev/hub/v2/device-provisioning-service/service/http"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/test"
	httpgwTest "github.com/plgd-dev/hub/v2/http-gateway/test"
	pkgHttpPb "github.com/plgd-dev/hub/v2/pkg/net/http/pb"
	hubTest "github.com/plgd-dev/hub/v2/test"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	hubTestService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceProvisionServiceServerGetProvisioningRecords(t *testing.T) {
	samples := pb.ProvisioningRecords{
		{
			Id:                "mfgID1",
			EnrollmentGroupId: "eg1",
			DeviceId:          "d1",
			Owner:             test.DPSOwner,
		}, {
			Id:                "mfgID2",
			EnrollmentGroupId: "eg2",
			DeviceId:          "d2",
			Owner:             test.DPSOwner,
		},
	}
	type args struct {
		accept                  string
		idFilter                []string
		enrollmentGroupIDFilter []string
		deviceIDFilter          []string
	}
	tests := []struct {
		name    string
		args    args
		want    pb.ProvisioningRecords
		wantErr bool
	}{
		{
			name: "invalidID",
			args: args{
				idFilter: []string{"invalidID"},
			},
			wantErr: true,
		},
		{
			name: "filter by id",
			args: args{
				idFilter: []string{samples[0].GetId()},
			},
			want: []*pb.ProvisioningRecord{samples[0]},
		},
		{
			name: "filter by enrollmentGroupId",
			args: args{
				enrollmentGroupIDFilter: []string{samples[1].GetEnrollmentGroupId()},
			},
			want: []*pb.ProvisioningRecord{samples[1]},
		},
		{
			name: "filter by deviceId",
			args: args{
				deviceIDFilter: []string{samples[1].GetDeviceId()},
			},
			want: []*pb.ProvisioningRecord{samples[1]},
		},
	}

	hubShutdown := hubTestService.SetUpServices(context.Background(), t, hubTestService.SetUpServicesMachine2MachineOAuth|hubTestService.SetUpServicesOAuth)
	defer hubShutdown()

	store, closeStore := test.NewMongoStore(t)
	defer closeStore()

	_, closeHTTP := test.NewHTTPService(context.Background(), t, store)
	defer closeHTTP()

	for _, sample := range samples {
		err := store.UpdateProvisioningRecord(context.Background(), test.DPSOwner, sample)
		require.NoError(t, err)
	}
	err := store.FlushBulkWriter()
	require.NoError(t, err)

	token := oauthTest.GetDefaultAccessToken(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httpgwTest.NewRequest(http.MethodGet, httpService.ProvisioningRecords, nil).
				Host(test.DPSHTTPHost).AuthToken(token).Accept(tt.args.accept).AddQuery(httpService.IDFilterQueryKey, tt.args.idFilter...).
				AddQuery(httpService.EnrollmentGroupIDFilterQueryKey, tt.args.enrollmentGroupIDFilter...).AddQuery(httpService.DeviceIDFilterQueryKey, tt.args.deviceIDFilter...).Build()
			resp := httpgwTest.HTTPDo(t, request)
			defer func() {
				_ = resp.Body.Close()
			}()

			var got pb.ProvisioningRecords
			for {
				var dev pb.ProvisioningRecord
				err := pkgHttpPb.Unmarshal(resp.StatusCode, resp.Body, &dev)
				if errors.Is(err, io.EOF) {
					break
				}
				require.NoError(t, err)
				assert.NotEmpty(t, dev.GetCreationDate())
				dev.CreationDate = 0
				got = append(got, &dev)
			}
			require.Len(t, got, len(tt.want))
			tt.want.Sort()
			got.Sort()
			for i := range got {
				hubTest.CheckProtobufs(t, tt.want[i], got[i], hubTest.RequireToCheckFunc(require.Equal))
			}
		})
	}
}

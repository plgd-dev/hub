package mongodb_test

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/store"
	"github.com/plgd-dev/hub/v2/snippet-service/test"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func makeLightResourceConfiguration(t *testing.T, id string, power int, ttl int64) *pb.Configuration_Resource {
	return &pb.Configuration_Resource{
		Href: hubTest.TestResourceLightInstanceHref(id),
		Content: &commands.Content{
			Data: hubTest.EncodeToCbor(t, map[string]interface{}{
				"power": power,
			}),
			ContentType:       message.AppOcfCbor.String(),
			CoapContentFormat: int32(message.AppOcfCbor),
		},
		TimeToLive: ttl,
	}
}

func TestStoreCreateConfiguration(t *testing.T) {
	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	confID := uuid.New().String()
	const owner = "owner1"
	resources := []*pb.Configuration_Resource{
		makeLightResourceConfiguration(t, "1", 1, 1337),
	}

	type args struct {
		create *pb.Configuration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				create: &pb.Configuration{
					Id:        confID,
					Name:      "valid",
					Owner:     owner,
					Version:   0,
					Resources: resources,
				},
			},
		},
		{
			name: "duplicit item (ID)",
			args: args{
				create: &pb.Configuration{
					Id:        confID,
					Name:      "duplicit ID",
					Owner:     owner,
					Resources: resources,
					Version:   42,
				},
			},
			wantErr: true,
		},
		{
			name: "missing ID",
			args: args{
				create: &pb.Configuration{
					Name:      "missing ID",
					Owner:     owner,
					Version:   0,
					Resources: resources,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid ID",
			args: args{
				create: &pb.Configuration{
					Id:        "invalid",
					Name:      "invalid ID",
					Owner:     owner,
					Version:   0,
					Resources: resources,
				},
			},
			wantErr: true,
		},
		{
			name: "missing owner",
			args: args{
				create: &pb.Configuration{
					Id:        confID,
					Name:      "missing owner",
					Version:   0,
					Resources: resources,
				},
			},
			wantErr: true,
		},
		{
			name: "missing resources",
			args: args{
				create: &pb.Configuration{
					Id:      confID,
					Name:    "missing resources",
					Owner:   owner,
					Version: 0,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.CreateConfiguration(ctx, tt.args.create)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUpdateConfiguration(t *testing.T) {
	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	confID := uuid.New().String()
	const owner = "owner1"
	resources := []*pb.Configuration_Resource{
		makeLightResourceConfiguration(t, "1", 1, 1337),
	}
	_, err := s.CreateConfiguration(ctx, &pb.Configuration{
		Id:        confID,
		Name:      "valid",
		Owner:     owner,
		Version:   0,
		Resources: resources,
	})
	require.NoError(t, err)

	type args struct {
		update *pb.Configuration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "non-matching owner",
			args: args{
				update: &pb.Configuration{
					Id:        confID,
					Owner:     "invalid",
					Version:   0,
					Resources: resources,
				},
			},
			wantErr: true,
		},
		{
			name: "duplicit version",
			args: args{
				update: &pb.Configuration{
					Id:        confID,
					Owner:     owner,
					Version:   0,
					Resources: resources,
				},
			},
			wantErr: true,
		},
		{
			name: "missing ID",
			args: args{
				update: &pb.Configuration{
					Owner:     owner,
					Version:   1,
					Resources: resources,
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				update: &pb.Configuration{
					Id:      confID,
					Owner:   owner,
					Version: 1,
					Resources: []*pb.Configuration_Resource{
						makeLightResourceConfiguration(t, "2", 2, 42),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.UpdateConfiguration(ctx, tt.args.update)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

var testConfigurationIDs = make(map[int]string)

func testConfigurationID(i int) string {
	if id, ok := testConfigurationIDs[i]; ok {
		return id
	}
	id := uuid.New().String()
	testConfigurationIDs[i] = id
	return id
}

func testConfigurationName(i int) string {
	return "cfg" + strconv.Itoa(i)
}

func testConfigurationOwner(i int) string {
	return "owner" + strconv.Itoa(i)
}

func testConfigurationResources(t *testing.T, start, n int) []*pb.Configuration_Resource {
	resources := make([]*pb.Configuration_Resource, 0, n)
	for i := start; i < start+n; i++ {
		resources = append(resources, &pb.Configuration_Resource{
			Href: hubTest.TestResourceLightInstanceHref(strconv.Itoa(i)),
			Content: &commands.Content{
				Data: hubTest.EncodeToCbor(t, map[string]interface{}{
					"power": i,
				}),
				ContentType:       message.AppOcfCbor.String(),
				CoapContentFormat: int32(message.AppOcfCbor),
			},
			TimeToLive: 1337,
		})
	}
	return resources
}

func addConfigurationsToStore(ctx context.Context, t *testing.T, s store.Store, n int) map[string]store.Configuration {
	const numConfigs = 10
	const numOwners = 3
	versions := make(map[int]uint64, numConfigs)
	owners := make(map[int]string, numConfigs)
	configurations := make(map[string]store.Configuration)
	for i := 0; i < n; i++ {
		version, ok := versions[i%numConfigs]
		if !ok {
			version = 0
			versions[i%numConfigs] = version
		}
		versions[i%numConfigs]++
		owner, ok := owners[i%numConfigs]
		if !ok {
			owner = testConfigurationOwner(i % numOwners)
			owners[i%numConfigs] = owner
		}
		confIn := &pb.Configuration{
			Id:        testConfigurationID(i % numConfigs),
			Version:   version,
			Resources: testConfigurationResources(t, i%16, (i%5)+1),
			Owner:     owner,
		}
		var conf *pb.Configuration
		var err error
		if !ok {
			confIn.Name = testConfigurationName(i % numConfigs)
			conf, err = s.CreateConfiguration(ctx, confIn)
			require.NoError(t, err)
		} else {
			conf, err = s.UpdateConfiguration(ctx, confIn)
			require.NoError(t, err)
		}

		configuration, ok := configurations[conf.GetId()]
		if !ok {
			configuration = store.Configuration{
				Id:    conf.GetId(),
				Owner: conf.GetOwner(),
				Name:  conf.GetName(),
			}
			configurations[conf.GetId()] = configuration
		}
		configuration.Versions = append(configuration.Versions, store.ConfigurationVersion{
			Version:   conf.GetVersion(),
			Resources: conf.GetResources(),
		})
		configurations[conf.GetId()] = configuration
	}
	return configurations
}

func TestStoreGetConfigurations(t *testing.T) {
	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()
	confs := addConfigurationsToStore(ctx, t, s, 500)

	type args struct {
		owner string
		query *pb.GetConfigurationsRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    func(t *testing.T, configurations []*store.Configuration)
	}{
		{
			name: "all",
			args: args{
				owner: "",
				query: nil,
			},
			want: func(t *testing.T, configurations []*store.Configuration) {
				require.Len(t, configurations, len(confs))
			},
		},
		{
			name: "owner0",
			args: args{
				owner: testConfigurationOwner(0),
				query: nil,
			},
			want: func(t *testing.T, configurations []*store.Configuration) {
				require.NotEmpty(t, configurations)
				for _, c := range configurations {
					require.Equal(t, testConfigurationOwner(0), c.Owner)
					conf, ok := confs[c.Id]
					require.True(t, ok)
					test.CmpJSON(t, &conf, c)
				}
			},
		},
		{
			name: "id1/all",
			args: args{
				owner: "",
				query: &pb.GetConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Id: testConfigurationID(1),
							Version: &pb.IDFilter_All{
								All: true,
							},
						},
					},
				},
			},
			want: func(t *testing.T, configurations []*store.Configuration) {
				require.Len(t, configurations, 1)
				c := configurations[0]
				conf, ok := confs[c.Id]
				require.True(t, ok)
				test.CmpJSON(t, &conf, c)
			},
		},
		{
			name: "owner2/id2/all",
			args: args{
				owner: testConfigurationOwner(2),
				query: &pb.GetConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Id: testConfigurationID(2),
							Version: &pb.IDFilter_All{
								All: true,
							},
						},
					},
				},
			},
			want: func(t *testing.T, configurations []*store.Configuration) {
				require.Len(t, configurations, 1)
				c := configurations[0]
				conf, ok := confs[c.Id]
				require.True(t, ok)
				require.Equal(t, testConfigurationID(2), conf.Id)
				require.Equal(t, testConfigurationOwner(2), conf.Owner)
				test.CmpJSON(t, &conf, c)
			},
		},
		{
			name: "latest",
			args: args{
				query: &pb.GetConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
					},
				},
			},
			want: func(t *testing.T, configurations []*store.Configuration) {
				require.Len(t, configurations, 10)
				for _, c := range configurations {
					_, ok := confs[c.Id]
					require.True(t, ok)
					require.Len(t, c.Versions, 1)
				}
			},
		},
		{
			name: "owner1/latest",
			args: args{
				owner: testConfigurationOwner(1),
				query: &pb.GetConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
					},
				},
			},
			want: func(t *testing.T, configurations []*store.Configuration) {
				require.Len(t, configurations, 3)
				for _, c := range configurations {
					conf, ok := confs[c.Id]
					require.True(t, ok)
					require.Equal(t, testConfigurationOwner(1), conf.Owner)
					require.Len(t, c.Versions, 1)
				}
			},
		},
		{
			name: "owner1/latest - non-matching owner",
			args: args{
				owner: testConfigurationOwner(2),
				query: &pb.GetConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Id: testConfigurationID(1),
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
					},
				},
			},
			want: func(t *testing.T, configurations []*store.Configuration) {
				require.Empty(t, configurations)
			},
		},
		{
			name: "owner2{latest, id2/latest, id5/latest} - non-matching owner", args: args{
				owner: testConfigurationOwner(2),
				query: &pb.GetConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
						{
							Id: testConfigurationID(2),
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
						{
							Id: testConfigurationID(5),
							Version: &pb.IDFilter_Latest{
								Latest: true,
							},
						},
					},
				},
			},
			want: func(t *testing.T, configurations []*store.Configuration) {
				require.Len(t, configurations, 3)
				for _, c := range configurations {
					conf, ok := confs[c.Id]
					require.True(t, ok)
					require.Equal(t, testConfigurationOwner(2), conf.Owner)
					require.Len(t, c.Versions, 1)
				}
			},
		},
		{
			name: "version/42", args: args{
				query: &pb.GetConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_Value{
								Value: 42,
							},
						},
					},
				},
			},
			want: func(t *testing.T, configurations []*store.Configuration) {
				require.Len(t, configurations, 10)
				for _, c := range configurations {
					_, ok := confs[c.Id]
					require.True(t, ok)
					require.Len(t, c.Versions, 1)
					require.Equal(t, uint64(42), c.Versions[0].Version)
				}
			},
		},
		{
			name: "owner3/version/{13, 37, 42}", args: args{
				owner: testConfigurationOwner(2),
				query: &pb.GetConfigurationsRequest{
					IdFilter: []*pb.IDFilter{
						{
							Version: &pb.IDFilter_Value{
								Value: 13,
							},
						},
						{
							Version: &pb.IDFilter_Value{
								Value: 37,
							},
						},
						{
							Version: &pb.IDFilter_Value{
								Value: 42,
							},
						},
						// duplicates should be ignored
						{
							Version: &pb.IDFilter_Value{
								Value: 37,
							},
						},
						// filter with Id should be ignored if there are filters without Id
						{
							Id: testConfigurationID(2),
							Version: &pb.IDFilter_Value{
								Value: 37,
							},
						},
					},
				},
			},
			want: func(t *testing.T, configurations []*store.Configuration) {
				require.Len(t, configurations, 3)
				for _, c := range configurations {
					_, ok := confs[c.Id]
					require.True(t, ok)
					require.Len(t, c.Versions, 3)
					slices.SortFunc(c.Versions, func(i, j store.ConfigurationVersion) int {
						return cmp.Compare(i.Version, j.Version)
					})
					require.Equal(t, uint64(13), c.Versions[0].Version)
					require.Equal(t, uint64(37), c.Versions[1].Version)
					require.Equal(t, uint64(42), c.Versions[2].Version)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configurations []*store.Configuration
			err := s.GetConfigurations(ctx, tt.args.owner, tt.args.query, func(iterCtx context.Context, iter store.Iterator[store.Configuration]) error {
				var conf store.Configuration
				for iter.Next(iterCtx, &conf) {
					configurations = append(configurations, conf.Clone())
				}
				return iter.Err()
			})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.want(t, configurations)
		})
	}
}

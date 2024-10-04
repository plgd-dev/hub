// ************************************************************************
// Copyright (C) 2022 plgd.dev, s.r.o.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ************************************************************************

package service_test

import (
	"testing"

	"github.com/plgd-dev/hub/v2/certificate-authority/service"
	"github.com/plgd-dev/hub/v2/certificate-authority/test"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	type args struct {
		cfg service.Config
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				cfg: func() service.Config {
					c := test.MakeConfig(t)
					c.Clients.Storage.ExtendCronParserBySeconds = true
					c.Clients.Storage.CleanUpRecords = "*/1 * * * * *"
					return c
				}(),
			},
		},
		{
			name: "invalid log config",
			args: args{
				cfg: func() service.Config {
					c := test.MakeConfig(t)
					c.Log.Level = 42
					return c
				}(),
			},
			wantErr: true,
		},
		{
			name: "invalid grpc api config",
			args: args{
				cfg: func() service.Config {
					c := test.MakeConfig(t)
					c.APIs.GRPC.Addr = "invalid"
					return c
				}(),
			},
			wantErr: true,
		},
		{
			name: "invalid http api config",
			args: args{
				cfg: func() service.Config {
					c := test.MakeConfig(t)
					c.APIs.HTTP.Addr = "invalid"
					return c
				}(),
			},
			wantErr: true,
		},
		{
			name: "invalid signer config",
			args: args{
				cfg: func() service.Config {
					c := test.MakeConfig(t)
					c.Signer.CAPool = 42
					return c
				}(),
			},
			wantErr: true,
		},
		{
			name: "invalid clients storage config",
			args: args{
				cfg: func() service.Config {
					c := test.MakeConfig(t)
					c.Clients.Storage.CleanUpRecords = "invalid"
					return c
				}(),
			},
			wantErr: true,
		},
		{
			name: "invalid clients telemetry config",
			args: args{
				cfg: func() service.Config {
					c := test.MakeConfig(t)
					c.Clients.OpenTelemetryCollector.GRPC.Enabled = true
					c.Clients.OpenTelemetryCollector.GRPC.Connection.Addr = ""
					return c
				}(),
			},
			wantErr: true,
		},
		{
			name: "invalid hubID",
			args: args{
				cfg: func() service.Config {
					c := test.MakeConfig(t)
					c.HubID = "invalid"
					return c
				}(),
			},
			wantErr: true,
		},
		{
			name: "invalid signer",
			args: args{
				cfg: func() service.Config {
					c := test.MakeConfig(t)
					c.Signer.CertFile = "invalid"
					return c
				}(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.args.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestHTTPConfigValidate(t *testing.T) {
	type args struct {
		cfg service.HTTPConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				cfg: test.MakeHTTPConfig(),
			},
		},
		{
			name: "invalid external address",
			args: args{
				cfg: func() service.HTTPConfig {
					cfg := test.MakeHTTPConfig()
					cfg.ExternalAddress = "invalid"
					return cfg
				}(),
			},
			wantErr: true,
		},
		{
			name: "invalid address",
			args: args{
				cfg: func() service.HTTPConfig {
					cfg := test.MakeHTTPConfig()
					cfg.Addr = "invalid"
					return cfg
				}(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.args.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestStorageConfigValidate(t *testing.T) {
	type args struct {
		cfg service.StorageConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid - disabled",
			args: args{
				cfg: func() service.StorageConfig {
					c := test.MakeStorageConfig()
					c.CleanUpRecords = ""
					return c
				}(),
			},
		},
		{
			name: "invalid",
			args: args{
				cfg: func() service.StorageConfig {
					c := test.MakeStorageConfig()
					c.CleanUpRecords = "invalid"
					return c
				}(),
			},
			wantErr: true,
		},
		{
			name: "invalid db",
			args: args{
				cfg: func() service.StorageConfig {
					c := test.MakeStorageConfig()
					c.Embedded.Use = "invalid"
					return c
				}(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.args.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

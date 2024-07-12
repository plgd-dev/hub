package config_test

import (
	"testing"

	"github.com/plgd-dev/hub/v2/snippet-service/store/config"
	"github.com/plgd-dev/hub/v2/snippet-service/test"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.Config
		wantErr bool
	}{
		{
			name: "valid",
			cfg:  test.MakeStoreConfig(),
		},
		{
			name: "valid - no cron",
			cfg: func() config.Config {
				cfg := test.MakeStoreConfig()
				cfg.CleanUpExpiredUpdates = ""
				return cfg
			}(),
		},
		{
			name: "invalid - bad cron expression",
			cfg: func() config.Config {
				cfg := test.MakeStoreConfig()
				cfg.CleanUpExpiredUpdates = "bad"
				return cfg
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

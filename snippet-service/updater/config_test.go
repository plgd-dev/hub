package updater_test

import (
	"testing"

	"github.com/plgd-dev/hub/v2/snippet-service/test"
	"github.com/plgd-dev/hub/v2/snippet-service/updater"
	"github.com/stretchr/testify/require"
)

func TestResourceAggregateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     updater.ResourceUpdaterConfig
		wantErr bool
	}{
		{
			name: "valid",
			cfg:  test.MakeResourceUpdaterConfig(),
		},
		{
			name: "valid - no cron",
			cfg: func() updater.ResourceUpdaterConfig {
				cfg := test.MakeResourceUpdaterConfig()
				cfg.CleanUpExpiredUpdates = ""
				return cfg
			}(),
		},
		{
			name: "invalid - no connection",
			cfg: func() updater.ResourceUpdaterConfig {
				cfg := updater.ResourceUpdaterConfig{}
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid - bad cron expression",
			cfg: func() updater.ResourceUpdaterConfig {
				cfg := test.MakeResourceUpdaterConfig()
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

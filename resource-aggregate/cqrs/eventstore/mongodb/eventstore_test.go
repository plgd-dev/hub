// Copyright (c) 2015 - The Event Horizon authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mongodb

import (
	"context"
	"testing"

	"github.com/plgd-dev/kit/security/certManager"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEventStore(t *testing.T) {
	var config certManager.Config
	err := envconfig.Process("DIAL", &config)
	assert.NoError(t, err)

	dialCertManager, err := certManager.NewCertManager(config)
	require.NoError(t, err)

	tlsConfig := dialCertManager.GetClientTLSConfig()

	bus, err := NewEventStore(Config{
		URI: "mongodb://localhost:27017",
	}, nil, WithTLS(tlsConfig))
	assert.NoError(t, err)
	assert.NotNil(t, bus)
}

func TestInstanceId(t *testing.T) {
	var config certManager.Config
	err := envconfig.Process("DIAL", &config)
	assert.NoError(t, err)

	dialCertManager, err := certManager.NewCertManager(config)
	require.NoError(t, err)

	tlsConfig := dialCertManager.GetClientTLSConfig()

	ctx := context.Background()
	store, err := NewEventStore(Config{
		URI:          "mongodb://localhost:27017",
		DatabaseName: "test",
	}, nil, WithTLS(tlsConfig))
	defer func() {
		store.Clear(ctx)
		store.Close(ctx)
	}()
	assert.NoError(t, err)

	for i := int64(1); i < 10; i++ {
		instanceId, err := store.GetInstanceId(ctx, "b")
		assert.NoError(t, err)
		err = store.RemoveInstanceId(ctx, instanceId)
		assert.NoError(t, err)
	}
}

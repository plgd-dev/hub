package cqldb_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gocql/gocql"
	"github.com/plgd-dev/hub/v2/pkg/cqldb"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestCqlDB(t *testing.T) {
	config := cqldb.Config{
		Hosts:          config.SCYLLA_HOSTS,
		TLS:            config.MakeHttpClientConfig().TLS,
		NumConns:       1,
		ConnectTimeout: time.Second * 5,
		Keyspace: cqldb.KeyspaceConfig{
			Name:   "example",
			Create: true,
			Replication: map[string]interface{}{
				"class":              "SimpleStrategy",
				"replication_factor": 1,
			},
		},
	}
	err := config.Validate()
	require.NoError(t, err)

	logger := log.NewLogger(log.MakeDefaultConfig())
	// cert manager
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		err = fileWatcher.Close()
		require.NoError(t, err)
	}()
	certManagerClient, err := client.New(config.TLS, fileWatcher, logger, nil)
	require.NoError(t, err)
	defer certManagerClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	client, err := cqldb.New(ctx, config, certManagerClient.GetTLSConfig(), logger, nil)
	require.NoError(t, err)
	defer func() {
		err1 := client.DropKeyspace(ctx)
		require.NoError(t, err1)
		client.Close()
	}()

	s := client.Session()

	err = s.Query("create table if not exists example.batches(pk int, ck int, description text, PRIMARY KEY(pk, ck));").Exec()
	require.NoError(t, err)

	defer func() {
		err1 := client.DropTable(ctx, "batches")
		require.NoError(t, err1)
	}()

	now := time.Now()
	b := s.NewBatch(gocql.UnloggedBatch).WithContext(ctx)
	b.Entries = append(b.Entries, gocql.BatchEntry{
		Stmt:       "INSERT INTO example.batches (pk, ck, description) VALUES (?, ?, ?)",
		Args:       []interface{}{1, 2, "1.2"},
		Idempotent: true,
	})
	b.Entries = append(b.Entries, gocql.BatchEntry{
		Stmt:       "INSERT INTO example.batches (pk, ck, description) VALUES (?, ?, ?)",
		Args:       []interface{}{1, 3, "1.3"},
		Idempotent: true,
	})
	err = s.ExecuteBatch(b)
	require.NoError(t, err)
	fmt.Println(time.Since(now))

	scanner := s.Query("SELECT pk, ck, description FROM example.batches").WithContext(ctx).Iter().Scanner()
	for scanner.Next() {
		var pk, ck int32
		var description string
		err = scanner.Scan(&pk, &ck, &description)
		require.NoError(t, err)
		require.NotEqual(t, int32(0), pk)
		require.NotEqual(t, int32(0), ck)
		require.NotEmpty(t, description)
		require.Equal(t, fmt.Sprintf("%v.%v", pk, ck), description)
	}
}

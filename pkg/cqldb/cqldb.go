package cqldb

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/gocql/gocql"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"go.opentelemetry.io/otel/trace"
)

type OnClearFn = func(context.Context) error

// Client implements an Client for CqlDB.
type Client struct {
	session   *gocql.Session
	config    Config
	logger    log.Logger
	closeFunc fn.FuncList

	onClear OnClearFn
}

func createKeyspace(ctx context.Context, session *gocql.Session, cfg Config) error {
	val, err := json.Marshal(cfg.Keyspace.Replication)
	if err != nil {
		return fmt.Errorf("failed to marshal replication(%v) for keyspace(%v): %w", cfg.Keyspace.Replication, cfg.Keyspace.Name, err)
	}
	q := "create keyspace if not exists " + cfg.Keyspace.Name + " with replication = " + strings.ReplaceAll(string(val), "\"", "'")
	err = session.Query(q).WithContext(ctx).Exec()
	if err != nil {
		return fmt.Errorf("failed to create keyspace(%v): %w", cfg.Keyspace.Name, err)
	}
	return nil
}

func (s *Client) createIndex(ctx context.Context, table string, index Index) error {
	indexArg := index.PartitionKey
	if index.SecondaryColumn != "" {
		if index.PartitionKey != "" {
			indexArg = "(" + indexArg + ")," + index.SecondaryColumn
		} else {
			indexArg = index.SecondaryColumn
		}
	}
	q := "create index if not exists " + index.Name + " on " + s.Keyspace() + "." + table + " (" + indexArg + ");"
	err := s.Session().Query(q).WithContext(ctx).Exec()
	if err != nil {
		var cqlErr gocql.RequestError
		if errors.As(err, &cqlErr) && cqlErr.Code() == 8704 {
			s.logger.Warnf("error during create index with query '%v': %v", q, err)
			return nil
		}
		return fmt.Errorf("failed to create index(%v) for table(%v): %w", index.Name, table, err)
	}
	return nil
}

type Index struct {
	Name            string
	PartitionKey    string
	SecondaryColumn string
}

func (s *Client) CreateIndexes(ctx context.Context, table string, indexes []Index) error {
	for _, index := range indexes {
		err := s.createIndex(ctx, table, index)
		if err != nil {
			return err
		}
	}
	return nil
}

// NewStore creates a new Client.
func New(ctx context.Context, cfg Config, tls *tls.Config, logger log.Logger, _ trace.TracerProvider) (*Client, error) {
	logger = logger.With("database", "cqlDB")
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	clusterCfg := gocql.NewCluster(cfg.Hosts...)
	clusterCfg.ConnectTimeout = cfg.ConnectTimeout
	clusterCfg.Port = cfg.Port
	clusterCfg.SslOpts = &gocql.SslOptions{
		Config:                 tls,
		EnableHostVerification: true,
	}
	clusterCfg.NumConns = cfg.NumConns
	clusterCfg.ReconnectionPolicy = &gocql.ConstantReconnectionPolicy{
		MaxRetries: cfg.ReconnectionPolicy.Constant.MaxRetries,
		Interval:   cfg.ReconnectionPolicy.Constant.Interval,
	}
	if cfg.UseHostnameResolution {
		clusterCfg.HostDialer = &defaultHostDialer{
			dialer: &net.Dialer{
				Timeout: cfg.ConnectTimeout,
			},
			tlsConfig: tls,
			logger:    logger,
		}
	}

	session, err := clusterCfg.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("failed to dial database: %w", err)
	}

	if err = session.Query("select * from system.local").WithContext(ctx).Exec(); err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if cfg.Keyspace.Create {
		err := createKeyspace(ctx, session, cfg)
		if err != nil {
			session.Close()
			return nil, err
		}
	}

	s := &Client{
		session: session,
		config:  cfg,
		logger:  logger,
	}

	s.onClear = func(clearCtx context.Context) error {
		// default clear function drops the whole database
		return s.DropKeyspace(clearCtx)
	}
	return s, nil
}

func (s *Client) Keyspace() string {
	return s.config.Keyspace.Name
}

// Set the function called on Clear
func (s *Client) SetOnClear(onClear OnClearFn) {
	s.onClear = onClear
}

// Clear clears the event storage.
func (s *Client) Clear(ctx context.Context) error {
	if s.onClear != nil {
		return s.onClear(ctx)
	}
	return nil
}

// Get mongodb client
func (s *Client) Session() *gocql.Session {
	return s.session
}

// Close closes the database session.
func (s *Client) Close() {
	s.session.Close()
	s.closeFunc.Execute()
}

// Drops the whole database
func (s *Client) DropKeyspace(ctx context.Context) error {
	return s.session.Query("drop keyspace " + s.config.Keyspace.Name).WithContext(ctx).Exec()
}

// Drops selected collection from database
func (s *Client) DropTable(ctx context.Context, table string) error {
	return s.session.Query("drop table " + s.config.Keyspace.Name + "." + table).WithContext(ctx).Exec()
}

func (s *Client) AddCloseFunc(f func()) {
	s.closeFunc.AddFunc(f)
}

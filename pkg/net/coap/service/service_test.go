package service

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/pion/dtls/v3"
	coapDtls "github.com/plgd-dev/go-coap/v3/dtls"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/go-coap/v3/options"
	"github.com/plgd-dev/go-coap/v3/tcp"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/service"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

func TestNew(t *testing.T) {
	type args struct {
		config  Config
		options []func(*Options)
	}
	tests := []struct {
		name    string
		args    args
		want    *service.Service
		wantErr bool
	}{
		{
			name: "invalid config",
			args: args{
				config: Config{},
			},
			wantErr: true,
		},
		{
			name: "valid config unsecure config",
			args: args{
				config: Config{
					Addr:            "localhost:0",
					Protocols:       []Protocol{TCP, UDP},
					MaxMessageSize:  1024,
					MessagePoolSize: 1024,
					BlockwiseTransfer: BlockwiseTransferConfig{
						Enabled: true,
						SZX:     "1024",
					},
					KeepAlive: &KeepAlive{
						Timeout: time.Second * 20,
					},
					TLS: TLSConfig{
						Enabled: new(bool), // TLS is disabled
						// Embedded: config.MakeTLSServerConfig(),
					},
				},
				options: []func(*Options){
					WithMessagePool(pool.New(uint32(1024), 1024)),
					WithOnNewConnection(func(mux.Conn) {}),
					WithOnInactivityConnection(func(mux.Conn) {}),
					WithOverrideTLS(func(cfg *tls.Config, _ VerifyByCRL) *tls.Config { return cfg }),
				},
			},
		},
		{
			name: "valid config secure config",
			args: args{
				config: Config{
					Addr:            "localhost:0",
					Protocols:       []Protocol{TCP, UDP},
					MaxMessageSize:  1024,
					MessagePoolSize: 1024,
					BlockwiseTransfer: BlockwiseTransferConfig{
						Enabled: true,
						SZX:     "1024",
					},
					InactivityMonitor: &InactivityMonitor{
						Timeout: time.Second * 20,
					},
					TLS: TLSConfig{
						Embedded: config.MakeTLSServerConfig(),
					},
				},
				options: []func(*Options){
					WithMessagePool(pool.New(uint32(1024), 1024)),
					WithOnNewConnection(func(mux.Conn) {}),
					WithOnInactivityConnection(func(mux.Conn) {}),
					WithOverrideTLS(func(cfg *tls.Config, _ VerifyByCRL) *tls.Config { return cfg }),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := mux.NewRouter()
			logger := log.NewLogger(log.MakeDefaultConfig())
			fileWatcher, err := fsnotify.NewWatcher(logger)
			require.NoError(t, err)
			got, err := New(context.Background(), tt.args.config, router, fileWatcher, logger, tt.args.options...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			go func() {
				errS := got.Serve()
				assert.NoError(t, errS)
			}()
			err = got.Close()
			require.NoError(t, err)
		})
	}
}

func TestOnClientInactivityTCP(t *testing.T) {
	router := mux.NewRouter()
	logCfg := log.MakeDefaultConfig()
	logCfg.Level = log.DebugLevel
	logger := log.NewLogger(logCfg)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	tlsCfg := config.MakeTLSServerConfig()
	tlsCfg.ClientCertificateRequired = false
	cfg := Config{
		Addr:            "127.0.0.1:23456",
		Protocols:       []Protocol{TCP},
		MaxMessageSize:  1024,
		MessagePoolSize: 1024,
		BlockwiseTransfer: BlockwiseTransferConfig{
			Enabled: true,
			SZX:     "1024",
		},
		InactivityMonitor: &InactivityMonitor{
			Timeout: time.Second * 1,
		},
		TLS: TLSConfig{
			Embedded: tlsCfg,
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	closeChan := make(chan struct{}, 2)
	got, err := New(ctx, cfg, router, fileWatcher, logger, WithOnNewConnection(func(conn mux.Conn) {
		conn.AddOnClose(func() {
			closeChan <- struct{}{}
		})
	}))
	require.NoError(t, err)
	go func() {
		errS := got.Serve()
		assert.NoError(t, errS)
	}()
	time.Sleep(time.Second * 3)

	// test TCP
	c, err := tcp.Dial(cfg.Addr, options.WithTLS(&tls.Config{
		InsecureSkipVerify: true,
	}), options.WithContext(ctx))
	require.NoError(t, err)
	_, err = c.Get(ctx, "/a")
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 1500)

	select {
	case <-c.Done():
		t.Log("TCP client closed in client")
	case <-closeChan:
		t.Log("TCP client closed in server")
	case <-ctx.Done():
		require.NoError(t, ctx.Err())
	}

	// clean up
	err = got.Close()
	require.NoError(t, err)
}

func TestOnClientInactivityUDP(t *testing.T) {
	router := mux.NewRouter()
	logCfg := log.MakeDefaultConfig()
	logCfg.Level = log.DebugLevel
	logger := log.NewLogger(logCfg)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	tlsCfg := config.MakeTLSServerConfig()
	tlsCfg.ClientCertificateRequired = false
	cfg := Config{
		Addr:            "127.0.0.1:23456",
		Protocols:       []Protocol{UDP},
		MaxMessageSize:  1024,
		MessagePoolSize: 1024,
		BlockwiseTransfer: BlockwiseTransferConfig{
			Enabled: true,
			SZX:     "1024",
		},
		InactivityMonitor: &InactivityMonitor{
			Timeout: time.Second * 1,
		},
		TLS: TLSConfig{
			Embedded: tlsCfg,
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	closeChan := make(chan struct{}, 2)
	got, err := New(ctx, cfg, router, fileWatcher, logger, WithOnNewConnection(func(conn mux.Conn) {
		conn.AddOnClose(func() {
			closeChan <- struct{}{}
		})
	}))
	require.NoError(t, err)
	go func() {
		errS := got.Serve()
		assert.NoError(t, errS)
	}()
	time.Sleep(time.Second * 3)

	// test DTLS
	cUDP, err := coapDtls.Dial(cfg.Addr, &dtls.Config{
		InsecureSkipVerify: true,
	}, options.WithContext(ctx))
	require.NoError(t, err)
	_, err = cUDP.Get(ctx, "/a")
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 1500)

	select {
	case <-cUDP.Done():
		t.Log("UDP client closed in client")
	case <-closeChan:
		t.Log("UDP client closed in server")
	case <-ctx.Done():
		require.NoError(t, ctx.Err())
	}

	// clean up
	err = got.Close()
	require.NoError(t, err)
}

func TestOnClientInactivityCustomTCP(t *testing.T) {
	router := mux.NewRouter()
	logCfg := log.MakeDefaultConfig()
	logCfg.Level = log.DebugLevel
	logger := log.NewLogger(logCfg)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	tlsCfg := config.MakeTLSServerConfig()
	tlsCfg.ClientCertificateRequired = false
	cfg := Config{
		Addr:            "127.0.0.1:23456",
		Protocols:       []Protocol{TCP},
		MaxMessageSize:  1024,
		MessagePoolSize: 1024,
		BlockwiseTransfer: BlockwiseTransferConfig{
			Enabled: true,
			SZX:     "1024",
		},
		InactivityMonitor: &InactivityMonitor{
			Timeout: time.Second * 1,
		},
		TLS: TLSConfig{
			Embedded: tlsCfg,
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	var numInactiveClients atomic.Int32
	closeChan := make(chan struct{}, 2)
	got, err := New(ctx, cfg, router, fileWatcher, logger, WithOnInactivityConnection(func(conn mux.Conn) {
		numInactiveClients.Inc()
		errC := conn.Close()
		require.NoError(t, errC)
	}), WithOnNewConnection(func(conn mux.Conn) {
		conn.AddOnClose(func() {
			closeChan <- struct{}{}
		})
	}))
	require.NoError(t, err)
	go func() {
		errS := got.Serve()
		assert.NoError(t, errS)
	}()
	time.Sleep(time.Second * 3)

	// test TCP
	c, err := tcp.Dial(cfg.Addr, options.WithTLS(&tls.Config{
		InsecureSkipVerify: true,
	}), options.WithContext(ctx))
	require.NoError(t, err)
	_, err = c.Get(ctx, "/a")
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 1500)

	select {
	case <-c.Done():
		t.Log("TCP client closed in client")
	case <-closeChan:
		t.Log("TCP client closed in server")
	case <-ctx.Done():
		require.NoError(t, ctx.Err())
	}

	// clean up
	err = got.Close()
	require.NoError(t, err)
	require.Equal(t, int32(1), numInactiveClients.Load())
}

func TestOnClientInactivityCustomUDP(t *testing.T) {
	router := mux.NewRouter()
	logCfg := log.MakeDefaultConfig()
	logCfg.Level = log.DebugLevel
	logger := log.NewLogger(logCfg)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	tlsCfg := config.MakeTLSServerConfig()
	tlsCfg.ClientCertificateRequired = false
	cfg := Config{
		Addr:            "127.0.0.1:23456",
		Protocols:       []Protocol{UDP},
		MaxMessageSize:  1024,
		MessagePoolSize: 1024,
		BlockwiseTransfer: BlockwiseTransferConfig{
			Enabled: true,
			SZX:     "1024",
		},
		InactivityMonitor: &InactivityMonitor{
			Timeout: time.Second * 1,
		},
		TLS: TLSConfig{
			Embedded: tlsCfg,
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	var numInactiveClients atomic.Int32
	closeChan := make(chan struct{}, 2)
	got, err := New(ctx, cfg, router, fileWatcher, logger, WithOnInactivityConnection(func(conn mux.Conn) {
		numInactiveClients.Inc()
		errC := conn.Close()
		require.NoError(t, errC)
	}), WithOnNewConnection(func(conn mux.Conn) {
		conn.AddOnClose(func() {
			closeChan <- struct{}{}
		})
	}))
	require.NoError(t, err)
	go func() {
		errS := got.Serve()
		assert.NoError(t, errS)
	}()
	time.Sleep(time.Second * 3)

	// test DTLS
	cUDP, err := coapDtls.Dial(cfg.Addr, &dtls.Config{
		InsecureSkipVerify: true,
	}, options.WithContext(ctx))
	require.NoError(t, err)
	_, err = cUDP.Get(ctx, "/a")
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 1500)

	select {
	case <-cUDP.Done():
		t.Log("UDP client closed in client")
	case <-closeChan:
		t.Log("UDP client closed in server")
	case <-ctx.Done():
		require.NoError(t, ctx.Err())
	}

	// clean up
	err = got.Close()
	require.NoError(t, err)
	require.Equal(t, int32(1), numInactiveClients.Load())
}

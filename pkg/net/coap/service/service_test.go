package service

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	coapOptionsConfig "github.com/plgd-dev/go-coap/v3/options/config"
	coapTcpClient "github.com/plgd-dev/go-coap/v3/tcp/client"
	coapUdpClient "github.com/plgd-dev/go-coap/v3/udp/client"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/service"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
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
					WithOnNewConnection(func(conn mux.Conn) {}),
					WithOnInactivityConnection(func(conn mux.Conn) {}),
					WithTCPGoPool(func(processReqFunc coapOptionsConfig.ProcessRequestFunc[*coapTcpClient.Conn], req *pool.Message, cc *coapTcpClient.Conn, handler coapOptionsConfig.HandlerFunc[*coapTcpClient.Conn]) error {
						processReqFunc(req, cc, handler)
						return nil
					}),
					WithUDPGoPool(func(processReqFunc coapOptionsConfig.ProcessRequestFunc[*coapUdpClient.Conn], req *pool.Message, cc *coapUdpClient.Conn, handler coapOptionsConfig.HandlerFunc[*coapUdpClient.Conn]) error {
						processReqFunc(req, cc, handler)
						return nil
					}),
					WithOverrideTLS(func(cfg *tls.Config) *tls.Config { return cfg }),
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
					WithOnNewConnection(func(conn mux.Conn) {}),
					WithOnInactivityConnection(func(conn mux.Conn) {}),
					WithTCPGoPool(func(processReqFunc coapOptionsConfig.ProcessRequestFunc[*coapTcpClient.Conn], req *pool.Message, cc *coapTcpClient.Conn, handler coapOptionsConfig.HandlerFunc[*coapTcpClient.Conn]) error {
						processReqFunc(req, cc, handler)
						return nil
					}),
					WithUDPGoPool(func(processReqFunc coapOptionsConfig.ProcessRequestFunc[*coapUdpClient.Conn], req *pool.Message, cc *coapUdpClient.Conn, handler coapOptionsConfig.HandlerFunc[*coapUdpClient.Conn]) error {
						processReqFunc(req, cc, handler)
						return nil
					}),
					WithOverrideTLS(func(cfg *tls.Config) *tls.Config { return cfg }),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := mux.NewRouter()
			logger := log.NewLogger(log.MakeDefaultConfig())
			fileWatcher, err := fsnotify.NewWatcher()
			require.NoError(t, err)
			got, err := New(context.Background(), tt.args.config, router, fileWatcher, logger, tt.args.options...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			go func() {
				err := got.Serve()
				require.NoError(t, err)
			}()
			err = got.Close()
			require.NoError(t, err)
		})
	}
}

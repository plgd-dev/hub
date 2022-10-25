package service

import (
	"crypto/tls"

	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	coapOptionsConfig "github.com/plgd-dev/go-coap/v3/options/config"
	coapTcpClient "github.com/plgd-dev/go-coap/v3/tcp/client"
	coapUdpClient "github.com/plgd-dev/go-coap/v3/udp/client"
)

type Options struct {
	OverrideTLSConfig      func(cfg *tls.Config) *tls.Config
	OnNewConnection        func(conn mux.Conn)
	OnInactivityConnection func(conn mux.Conn)
	TCPGoPool              coapOptionsConfig.GoPoolFunc[*coapTcpClient.Conn]
	UDPGoPool              coapOptionsConfig.GoPoolFunc[*coapUdpClient.Conn]
	MessagePool            *pool.Pool
}

func WithOverrideTLS(f func(cfg *tls.Config) *tls.Config) func(*Options) {
	return func(o *Options) {
		o.OverrideTLSConfig = f
	}
}

func WithOnNewConnection(f func(conn mux.Conn)) func(*Options) {
	return func(o *Options) {
		o.OnNewConnection = f
	}
}

func WithOnInactivityConnection(f func(conn mux.Conn)) func(*Options) {
	return func(o *Options) {
		o.OnInactivityConnection = f
	}
}

// Setup go pool for TCP/TCP-TLS connections
func WithTCPGoPool(f coapOptionsConfig.GoPoolFunc[*coapTcpClient.Conn]) func(*Options) {
	return func(o *Options) {
		o.TCPGoPool = f
	}
}

// Setup go pool for UDP/DTLS connections
func WithUDPGoPool(f coapOptionsConfig.GoPoolFunc[*coapUdpClient.Conn]) func(*Options) {
	return func(o *Options) {
		o.UDPGoPool = f
	}
}

func WithMessagePool(p *pool.Pool) func(*Options) {
	return func(o *Options) {
		o.MessagePool = p
	}
}

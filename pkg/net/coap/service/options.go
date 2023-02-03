package service

import (
	"crypto/tls"

	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
)

type Options struct {
	OverrideTLSConfig      func(cfg *tls.Config) *tls.Config
	OnNewConnection        func(conn mux.Conn)
	OnInactivityConnection func(conn mux.Conn)
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

func WithMessagePool(p *pool.Pool) func(*Options) {
	return func(o *Options) {
		o.MessagePool = p
	}
}

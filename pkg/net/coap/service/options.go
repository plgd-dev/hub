package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"

	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
)

type VerifyByCRL = func(context.Context, *x509.Certificate, []string) error

type Options struct {
	OverrideTLSConfig      func(cfg *tls.Config, verifyByCRL VerifyByCRL) *tls.Config
	OnNewConnection        func(conn mux.Conn)
	OnInactivityConnection func(conn mux.Conn)
	MessagePool            *pool.Pool
}

func WithOverrideTLS(f func(cfg *tls.Config, verifyByCRL VerifyByCRL) *tls.Config) func(*Options) {
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

package cqldb

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"

	"github.com/gocql/gocql"
	"github.com/plgd-dev/hub/v2/pkg/log"
)

// defaultHostDialer dials host in a default way.
type defaultHostDialer struct {
	dialer    gocql.Dialer
	tlsConfig *tls.Config
	logger    log.Logger
}

// cutLast slices s around the last instance of sep,
// returning the text before and after sep.
// The found result reports whether sep appears in s.
// If sep does not appear in s, cut returns s, "", false.
func cutLast(s, sep string) (before, after string, found bool) {
	if i := strings.LastIndex(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}

func validIpAddr(addr net.IP) bool {
	return addr != nil && !addr.IsUnspecified()
}

func toHostAndPort(v string) (string, string) {
	hostname, hostPort, _ := cutLast(v, ":")
	// strip ipv6 brackets
	if hostname[0] == '[' && hostname[len(hostname)-1] == ']' {
		hostname = hostname[1 : len(hostname)-1]
	}
	return hostname, hostPort
}

func (hd *defaultHostDialer) resolveHostnameAndPort(ctx context.Context, connAddr string) (string, bool) {
	// try to resolve connect address
	hostname, hostPort := toHostAndPort(connAddr)
	r := net.Resolver{}
	hostnames, err := r.LookupAddr(ctx, hostname)
	if err == nil && len(hostnames) > 0 {
		hostnameAndPort := hostnames[0]
		if strings.Contains(hostnameAndPort, ":") && hostnameAndPort[0] != '[' {
			hostnameAndPort = "[" + hostnameAndPort + "]"
		}
		if hostPort != "" {
			hostnameAndPort = hostnameAndPort + ":" + hostPort
		}
		return hostnameAndPort, true
	}
	return "", false
}

func (hd *defaultHostDialer) getAddr(ctx context.Context, host *gocql.HostInfo, connAddr string) string {
	hostnameAndPort := host.HostnameAndPort()
	originalHostnameAndPort := hostnameAndPort
	// if is ip try to resolve hostname
	hostname, _ := toHostAndPort(hostnameAndPort)
	if net.ParseIP(hostname) != nil {
		// try to resolve connect address
		resolvedHostnameAndPort, ok := hd.resolveHostnameAndPort(ctx, connAddr)
		if ok {
			hostnameAndPort = resolvedHostnameAndPort
			hd.logger.Debugf("resolved %v to %v", originalHostnameAndPort, hostnameAndPort)
		} else {
			hd.logger.Debugf("failed to resolve %v", originalHostnameAndPort)
		}
	} else {
		hd.logger.Debugf("skip resolving %v: not an ip address", originalHostnameAndPort)
	}
	return hostnameAndPort
}

func (hd *defaultHostDialer) DialHost(ctx context.Context, host *gocql.HostInfo) (*gocql.DialedHost, error) {
	ip := host.ConnectAddress()
	port := host.Port()

	if !validIpAddr(ip) {
		return nil, fmt.Errorf("host missing connect ip address: %v", ip)
	} else if port == 0 {
		return nil, fmt.Errorf("host missing port: %v", port)
	}

	connAddr := host.ConnectAddressAndPort()
	conn, err := hd.dialer.DialContext(ctx, "tcp", connAddr)
	if err != nil {
		return nil, err
	}
	addr := hd.getAddr(ctx, host, connAddr)
	return gocql.WrapTLS(ctx, conn, addr, hd.tlsConfig)
}

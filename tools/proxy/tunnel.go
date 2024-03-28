package main

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/log"
)

type Tunnel struct {
	cfg      TunnelConfig
	state    *State
	listener net.Listener
	failures int
	logger   log.Logger
	done     chan struct{}

	private struct {
		mux   sync.Mutex
		conns map[string]net.Conn
	}
}

func NewTunnel(cfg TunnelConfig, state *State, logger log.Logger) (*Tunnel, error) {
	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return nil, err
	}

	t := &Tunnel{
		cfg:      cfg,
		state:    state,
		listener: listener,
		logger:   logger.With("tunnel", cfg.Addr),
		done:     make(chan struct{}),
	}
	t.private.conns = make(map[string]net.Conn, 32)
	return t, nil
}

func (t *Tunnel) selectNextTarget(selectedTarget TargetConfig) error {
	nextState := TunnelState{
		SelectedTarget: t.cfg.Targets[0].Addr,
	}
	for i, target := range t.cfg.Targets {
		if i == len(t.cfg.Targets)-1 {
			return fmt.Errorf("cannot find next target: all next targets are already used")
		}
		if target.Addr == selectedTarget.Addr {
			nextIndex := (i + 1) % len(t.cfg.Targets)
			nextState.SelectedTarget = t.cfg.Targets[nextIndex].Addr
			break
		}
	}
	t.state.Set(t.cfg.Addr, nextState)
	return t.state.Dump()
}

func (t *Tunnel) getSelectedTarget() TargetConfig {
	state, ok := t.state.Get(t.cfg.Addr)
	if !ok {
		// If the state is not found, return the first target
		return t.cfg.Targets[0]
	}

	for _, target := range t.cfg.Targets {
		if target.Addr == state.SelectedTarget {
			return target
		}
	}

	// If the selected target is not found, return the first one
	return t.cfg.Targets[0]
}

func (t *Tunnel) dropConnections() {
	t.private.mux.Lock()
	defer t.private.mux.Unlock()

	for key, conn := range t.private.conns {
		_ = conn.Close()
		delete(t.private.conns, key)
	}
}

func (t *Tunnel) addConnection(conn net.Conn) TargetConfig {
	t.private.mux.Lock()
	defer t.private.mux.Unlock()

	t.private.conns[conn.RemoteAddr().String()] = conn
	return t.getSelectedTarget()
}

func (t *Tunnel) removeConnection(conn net.Conn) {
	t.private.mux.Lock()
	defer t.private.mux.Unlock()

	delete(t.private.conns, conn.RemoteAddr().String())
}

func (t *Tunnel) numConnections() int {
	t.private.mux.Lock()
	defer t.private.mux.Unlock()

	return len(t.private.conns)
}

func (t *Tunnel) incrementFailures(target TargetConfig) error {
	t.failures++
	if t.failures >= target.Liveness.FailureThreshold {
		if err := t.selectNextTarget(target); err != nil {
			return err
		}
		t.dropConnections()
		t.resetFailures()
		t.logger.Infof("selected next target: %v", t.getSelectedTarget().Addr)
	}
	return nil
}

func (t *Tunnel) resetFailures() {
	t.failures = 0
}

func (t *Tunnel) checkLiveness() {
	var originalTarget TargetConfig
	for {
		target := t.getSelectedTarget()
		if originalTarget.Addr != target.Addr {
			time.Sleep(target.Liveness.InitialDelay)
			originalTarget = target
		}
		start := time.Now()
		targetConn, err := net.DialTimeout("tcp", target.Addr, target.Liveness.Period)
		if err != nil {
			t.logger.Warnf("target %v is dead", target.Addr)
			err = t.incrementFailures(target)
			if err != nil {
				t.logger.Errorf("failed to increment failures %w", err)
			}
		} else {
			t.logger.Debugf("target %v is alive", target.Addr)
			_ = targetConn.Close()
			t.resetFailures()
		}
		select {
		case <-time.After(target.Liveness.Period - time.Since(start)):
			continue
		case <-t.done:
			return
		}
	}
}

func (t *Tunnel) handleConn(conn net.Conn, target TargetConfig) {
	logger := t.logger.With("client", conn.RemoteAddr(), "target", target.Addr)
	logger.Debugf("connected")
	defer func() {
		logger.Debugf("disconnected")
		t.removeConnection(conn)
	}()
	targetConn, err := net.DialTimeout("tcp", target.Addr, target.Timeouts.Dial)
	if err != nil {
		logger.Errorf("cannot dial target: %w", err)
		_ = conn.Close()
		return
	}
	defer targetConn.Close()

	go func() {
		defer targetConn.Close()
		defer conn.Close()

		buffer := make([]byte, target.BufferSize)
		for {
			err := targetConn.SetReadDeadline(time.Now().Add(target.Timeouts.Read))
			if err != nil {
				logger.Errorf("cannot set read deadline: %w", err)
				return
			}

			n, err := targetConn.Read(buffer)
			if err != nil {
				logger.Errorf("cannot read from target: %w", err)
				return
			}

			err = conn.SetWriteDeadline(time.Now().Add(target.Timeouts.Write))
			if err != nil {
				logger.Errorf("cannot set write deadline to client: %w", err)
				return
			}

			_, err = conn.Write(buffer[:n])
			if err != nil {
				logger.Errorf("cannot write to client: %w", err)
				return
			}
		}
	}()

	buffer := make([]byte, target.BufferSize)
	for {
		err := conn.SetReadDeadline(time.Now().Add(target.Timeouts.Read))
		if err != nil {
			logger.Errorf("cannot set read deadline from client: %w", err)
			return
		}

		n, err := conn.Read(buffer)
		if err != nil {
			logger.Errorf("cannot read from client: %w", err)
			return
		}

		err = targetConn.SetWriteDeadline(time.Now().Add(target.Timeouts.Write))
		if err != nil {
			logger.Errorf("cannot set write deadline to target: %w", err)
			return
		}

		_, err = targetConn.Write(buffer[:n])
		if err != nil {
			logger.Errorf("cannot write to target: %w", err)
			return
		}
	}
}

func (t *Tunnel) Serve() error {
	go t.checkLiveness()

	for {
		conn, err := t.listener.Accept()
		if err != nil {
			return fmt.Errorf("cannot accept connection: %w", err)
		}
		target := t.addConnection(conn)
		go t.handleConn(conn, target)
	}
}

func (t *Tunnel) Close() error {
	close(t.done)
	t.dropConnections()
	return t.listener.Close()
}

package manager

import (
	"context"
	"crypto/tls"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2/clientcredentials"

	"github.com/plgd-dev/kit/log"
	"golang.org/x/oauth2"
)

// Manager holds certificates from filesystem watched for changes
type Manager struct {
	mutex             sync.Mutex
	config            clientcredentials.Config
	requestTimeout    time.Duration
	tickFrequency     time.Duration
	startRefreshToken time.Time
	token             *oauth2.Token
	httpClient        *http.Client
	tokenErr          error
	doneWg            sync.WaitGroup
	done              chan struct{}
}

// NewManagerFromConfiguration creates a new oauth manager which refreshing token.
func NewManagerFromConfiguration(config Config, tlsCfg *tls.Config) (*Manager, error) {
	cfg := config.ToClientCrendtials()
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 1
	t.MaxConnsPerHost = 1
	t.MaxIdleConnsPerHost = 1
	t.IdleConnTimeout = time.Second * 30
	t.TLSClientConfig = tlsCfg
	httpClient := &http.Client{
		Transport: t,
		Timeout:   config.RequestTimeout,
	}
	token, startRefreshToken, err := getToken(cfg, httpClient, config.RequestTimeout)
	if err != nil {
		return nil, err
	}

	mgr := &Manager{
		config:            cfg,
		token:             token,
		startRefreshToken: startRefreshToken,
		requestTimeout:    config.RequestTimeout,
		httpClient:        httpClient,
		tickFrequency:     config.TickFrequency,

		done: make(chan struct{}),
	}
	mgr.doneWg.Add(1)

	go mgr.watchToken()

	return mgr, nil
}

// GetToken returns token for clients
func (a *Manager) GetToken(ctx context.Context) (*oauth2.Token, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.token, a.tokenErr
}

// Close ends watching token
func (a *Manager) Close() {
	if a.done != nil {
		close(a.done)
		a.doneWg.Wait()
	}
}

func (a *Manager) shouldRefresh() bool {
	return time.Now().After(a.startRefreshToken)
}

func getToken(cfg clientcredentials.Config, httpClient *http.Client, requestTimeout time.Duration) (*oauth2.Token, time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	token, err := cfg.Token(ctx)
	var startRefreshToken time.Time
	if err == nil {
		now := time.Now()
		startRefreshToken = now.Add(token.Expiry.Sub(now) * 2 / 3)
	}
	return token, startRefreshToken, err
}

func (a *Manager) refreshToken() {
	token, startRefreshToken, err := getToken(a.config, a.httpClient, a.requestTimeout)
	if err != nil {
		log.Errorf("cannot refresh token: %v", err)
	}
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.token = token
	a.tokenErr = err
	a.startRefreshToken = startRefreshToken
}

func (a *Manager) watchToken() {
	defer a.doneWg.Done()
	t := time.NewTicker(a.tickFrequency)
	defer t.Stop()

	for {
		select {
		case <-a.done:
			return
		case <-t.C:
			if a.shouldRefresh() {
				a.refreshToken()
			}
		}
	}
}
